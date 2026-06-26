package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
)

// newSharedTransport creates a shared http.Transport with connection pooling,
// proxy resolver, and timing-aware dial functions.
func newSharedTransport() *http.Transport {
	return &http.Transport{
		Proxy: nil, // Set per-request via sharedHTTPClient
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			timing, _ := ctx.Value(timingCtxKey{}).(*timingCapture)
			connectStart := time.Now()
			conn, err := (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, network, addr)
			connectEnd := time.Now()
			if timing != nil {
				timing.connectTime = connectStart
				timing.connectDone = connectEnd
				if conn != nil {
					timing.serverAddr = conn.RemoteAddr().String()
				}
			}
			return conn, err
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			timing, _ := ctx.Value(timingCtxKey{}).(*timingCapture)
			connectStart := time.Now()
			dialer := &net.Dialer{Timeout: 30 * time.Second}
			rawConn, err := dialer.DialContext(ctx, network, addr)
			connectEnd := time.Now()
			if timing != nil {
				timing.connectTime = connectStart
				timing.connectDone = connectEnd
			}
			if err != nil {
				return nil, err
			}
			if timing != nil {
				timing.serverAddr = rawConn.RemoteAddr().String()
			}
			tlsStart := time.Now()
			tlsConn := tls.Client(rawConn, &tls.Config{InsecureSkipVerify: true, ServerName: serverNameFromAddr(addr)})
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				rawConn.Close()
				return nil, err
			}
			tlsEnd := time.Now()
			if timing != nil {
				timing.tlsStart = tlsStart
				timing.tlsDone = tlsEnd
			}
			return tlsConn, nil
		},
		TLSHandshakeTimeout:   30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

// sharedHTTPClient returns an http.Client using the shared transport with proxy resolver.
func (p *Proxy) sharedHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   120 * time.Second,
		Transport: p.transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// ExecuteRequest sends a request through the proxy's shared upstream client.
func (p *Proxy) ExecuteRequest(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	return p.sharedHTTPClient().Do(req)
}

// ExecuteRequestWithClientHello sends an HTTPS request using best-effort uTLS mimicry when raw ClientHello bytes are available.
func (p *Proxy) ExecuteRequestWithClientHello(req *http.Request, clientHelloRaw []byte) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if req.URL == nil || !strings.EqualFold(req.URL.Scheme, "https") || len(clientHelloRaw) == 0 {
		return p.ExecuteRequest(req)
	}

	if req.Body != nil && req.GetBody == nil {
		return p.ExecuteRequest(req)
	}

	utlsReq := req.Clone(req.Context())
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return p.ExecuteRequest(req)
		}
		utlsReq.Body = body
		utlsReq.GetBody = req.GetBody
	}

	transport := &http.Transport{
		DialTLSContext:        p.utlsDialTLSContext(clientHelloRaw),
		TLSHandshakeTimeout:   30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	client := &http.Client{
		Timeout:   120 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(utlsReq)
	if err != nil {
		return p.ExecuteRequest(req)
	}
	return resp, nil
}

func (p *Proxy) utlsDialTLSContext(clientHelloRaw []byte) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := p.dialTunnelTarget(ctx, addr)
		if err != nil {
			return nil, err
		}

		fingerprinter := &utls.Fingerprinter{AllowBluntMimicry: true}
		spec, err := fingerprinter.FingerprintClientHello(clientHelloRaw)
		if err != nil {
			conn.Close()
			return nil, err
		}

		host := serverNameFromAddr(addr)
		uConn := utls.UClient(conn, &utls.Config{ServerName: host, InsecureSkipVerify: true}, utls.HelloCustom)
		if err := uConn.ApplyPreset(spec); err != nil {
			conn.Close()
			return nil, err
		}
		if err := uConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}
		return uConn, nil
	}
}

func wrapConnWithCapturedClientHello(conn net.Conn) (net.Conn, []byte) {
	baseConn := conn
	var reader *bufio.Reader
	if buffered, ok := conn.(*bufferedConn); ok {
		baseConn = buffered.Conn
		if existing, ok := buffered.reader.(*bufio.Reader); ok {
			reader = existing
		}
	}
	if reader == nil {
		reader = bufio.NewReader(baseConn)
	}
	raw, err := peekTLSClientHello(reader)
	if err != nil {
		return &bufferedConn{Conn: baseConn, reader: reader}, nil
	}
	return &bufferedConn{Conn: baseConn, reader: reader}, raw
}

func peekTLSClientHello(reader *bufio.Reader) ([]byte, error) {
	header, err := reader.Peek(5)
	if err != nil {
		return nil, err
	}
	if len(header) < 5 || header[0] != 0x16 {
		return nil, fmt.Errorf("not a tls handshake record")
	}
	recordLength := int(binary.BigEndian.Uint16(header[3:5]))
	record, err := reader.Peek(5 + recordLength)
	if err != nil {
		return nil, err
	}
	if len(record) < 6 || record[5] != 0x01 {
		return nil, fmt.Errorf("not a client hello handshake")
	}
	return append([]byte(nil), record...), nil
}

func (p *Proxy) proxyResolver() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		proxyURL, enabled := p.externalProxyURL()
		if !enabled {
			return nil, nil
		}
		target := ""
		if req != nil {
			if req.URL != nil {
				target = req.URL.Host
			}
			if target == "" {
				target = req.Host
			}
		}
		if p.shouldBypassExternalProxy(target) {
			return nil, nil
		}
		return proxyURL, nil
	}
}

func (p *Proxy) externalProxyURL() (*url.URL, bool) {
	settings := p.appSettingsSnapshot()
	if settings == nil || !settings.Proxy.ExternalProxy.Enabled || settings.Proxy.ExternalProxy.Host == "" || settings.Proxy.ExternalProxy.Port <= 0 {
		return nil, false
	}
	proxyURL := &url.URL{
		Scheme: settings.Proxy.ExternalProxy.Scheme,
		Host:   fmt.Sprintf("%s:%d", settings.Proxy.ExternalProxy.Host, settings.Proxy.ExternalProxy.Port),
	}
	if settings.Proxy.ExternalProxy.Username != "" {
		proxyURL.User = url.UserPassword(settings.Proxy.ExternalProxy.Username, settings.Proxy.ExternalProxy.Password)
	}
	return proxyURL, true
}

func (p *Proxy) shouldBypassExternalProxy(targetAddr string) bool {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return true
	}
	host, port := splitHostPortPreserveHost(targetAddr)
	normalizedHost := normalizeProxyHost(host)
	if normalizedHost == "" {
		return false
	}
	if normalizedHost == "localhost" {
		return true
	}
	if ip := net.ParseIP(normalizedHost); ip != nil && ip.IsLoopback() {
		return true
	}
	upstreamHost := normalizeProxyHost(settings.Proxy.ExternalProxy.Host)
	if upstreamHost != "" && normalizedHost == upstreamHost {
		if port == 0 || settings.Proxy.ExternalProxy.Port == 0 || port == settings.Proxy.ExternalProxy.Port {
			return true
		}
	}
	for _, rule := range settings.Proxy.ExternalProxy.BypassHosts {
		if matchProxyBypassRule(rule, normalizedHost, targetAddr) {
			return true
		}
	}
	return false
}

func matchProxyBypassRule(rule, normalizedHost, targetAddr string) bool {
	rule = strings.TrimSpace(strings.ToLower(rule))
	if rule == "" {
		return false
	}
	if rule == normalizedHost || rule == strings.ToLower(targetAddr) {
		return true
	}
	if strings.HasPrefix(rule, "*.") {
		suffix := strings.TrimPrefix(rule, "*.")
		return normalizedHost == suffix || strings.HasSuffix(normalizedHost, "."+suffix)
	}
	if _, cidr, err := net.ParseCIDR(rule); err == nil {
		if ip := net.ParseIP(normalizedHost); ip != nil {
			return cidr.Contains(ip)
		}
	}
	return false
}

func splitHostPortPreserveHost(addr string) (string, int) {
	host := strings.TrimSpace(addr)
	port := 0
	if host == "" {
		return "", 0
	}
	parsedHost, parsedPort, err := net.SplitHostPort(host)
	if err == nil {
		host = parsedHost
		fmt.Sscanf(parsedPort, "%d", &port)
		return host, port
	}
	return strings.Trim(host, "[]"), 0
}

func normalizeProxyHost(host string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
}

func (p *Proxy) dialTunnelTarget(ctx context.Context, targetAddr string) (net.Conn, error) {
	if p.shouldBypassExternalProxy(targetAddr) {
		return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp", targetAddr)
	}
	proxyURL, enabled := p.externalProxyURL()
	if !enabled {
		return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp", targetAddr)
	}
	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https":
		return p.dialViaHTTPProxy(ctx, proxyURL, targetAddr)
	case "socks5", "socks5h":
		return p.dialViaSOCKS5Proxy(ctx, proxyURL, targetAddr)
	default:
		return nil, fmt.Errorf("unsupported external proxy scheme: %s", proxyURL.Scheme)
	}
}

func (p *Proxy) dialViaHTTPProxy(ctx context.Context, proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	if strings.EqualFold(proxyURL.Scheme, "https") {
		conn, err = tls.DialWithDialer(dialer, "tcp", proxyURL.Host, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         serverNameFromAddr(proxyURL.Host),
		})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", proxyURL.Host)
	}
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}
	var connectReq bytes.Buffer
	connectReq.WriteString(fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr))
	if proxyURL.User != nil {
		username := proxyURL.User.Username()
		password, _ := proxyURL.User.Password()
		token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		connectReq.WriteString("Proxy-Authorization: Basic " + token + "\r\n")
	}
	connectReq.WriteString("\r\n")
	if _, err := conn.Write(connectReq.Bytes()); err != nil {
		conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("external proxy CONNECT failed: %s", resp.Status)
	}
	_ = conn.SetDeadline(time.Time{})
	if reader.Buffered() > 0 {
		return &bufferedConn{Conn: conn, reader: reader}, nil
	}
	return conn, nil
}

func (p *Proxy) dialViaSOCKS5Proxy(ctx context.Context, proxyURL *url.URL, targetAddr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", proxyURL.Host)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}
	methods := []byte{socks5NoAuth}
	if proxyURL.User != nil && proxyURL.User.Username() != "" {
		methods = []byte{socks5UserPass, socks5NoAuth}
	}
	if _, err := conn.Write(append([]byte{socks5Version, byte(len(methods))}, methods...)); err != nil {
		conn.Close()
		return nil, err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}
	if resp[0] != socks5Version || resp[1] == 0xFF {
		conn.Close()
		return nil, fmt.Errorf("external socks5 proxy rejected auth methods")
	}
	if resp[1] == socks5UserPass {
		username := proxyURL.User.Username()
		password, _ := proxyURL.User.Password()
		authReq := []byte{1, byte(len(username))}
		authReq = append(authReq, []byte(username)...)
		authReq = append(authReq, byte(len(password)))
		authReq = append(authReq, []byte(password)...)
		if _, err := conn.Write(authReq); err != nil {
			conn.Close()
			return nil, err
		}
		authResp := make([]byte, 2)
		if _, err := io.ReadFull(conn, authResp); err != nil {
			conn.Close()
			return nil, err
		}
		if authResp[1] != 0 {
			conn.Close()
			return nil, fmt.Errorf("external socks5 proxy authentication failed")
		}
	}
	host, port := splitHostPortPreserveHost(targetAddr)
	if host == "" || port <= 0 {
		conn.Close()
		return nil, fmt.Errorf("invalid target address: %s", targetAddr)
	}
	request := []byte{socks5Version, socks5Connect, 0}
	if ip := net.ParseIP(host); ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			request = append(request, socks5AtypIPv4)
			request = append(request, ipv4...)
		} else {
			request = append(request, socks5AtypIPv6)
			request = append(request, ip.To16()...)
		}
	} else {
		request = append(request, socks5AtypDomain, byte(len(host)))
		request = append(request, []byte(host)...)
	}
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(port))
	request = append(request, portBuf...)
	if _, err := conn.Write(request); err != nil {
		conn.Close()
		return nil, err
	}
	replyHead := make([]byte, 4)
	if _, err := io.ReadFull(conn, replyHead); err != nil {
		conn.Close()
		return nil, err
	}
	if replyHead[1] != socks5RepSuccess {
		conn.Close()
		return nil, fmt.Errorf("external socks5 proxy connect failed: %d", replyHead[1])
	}
	switch replyHead[3] {
	case socks5AtypIPv4:
		_, err = io.ReadFull(conn, make([]byte, 4+2))
	case socks5AtypIPv6:
		_, err = io.ReadFull(conn, make([]byte, 16+2))
	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err = io.ReadFull(conn, lenBuf); err == nil {
			_, err = io.ReadFull(conn, make([]byte, int(lenBuf[0])+2))
		}
	default:
		err = fmt.Errorf("unsupported external socks5 address type: %d", replyHead[3])
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func (p *Proxy) dialWebSocketUpstream(req *http.Request, scheme, targetHost string) (*wsUpstreamResponse, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("nil websocket request")
	}
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()
	conn, err := p.dialTunnelTarget(ctx, targetHost)
	if err != nil {
		return nil, err
	}
	if scheme == "wss" {
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true, ServerName: serverNameFromAddr(targetHost)})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}
		conn = tlsConn
	}
	reader := bufio.NewReader(conn)
	upstreamReq := cloneWebSocketRequest(req)
	upstreamReq.URL.Scheme = strings.TrimSuffix(scheme, "s")
	upstreamReq.URL.Host = targetHost
	upstreamReq.Host = req.Host
	upstreamReq.RequestURI = upstreamReq.URL.RequestURI()
	if err := upstreamReq.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(reader, upstreamReq)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols && resp.Body != nil {
		_ = resp.Body.Close()
	}
	return &wsUpstreamResponse{
		conn:       conn,
		reader:     reader,
		statusCode: resp.StatusCode,
		statusText: resp.Status,
		headers:    cloneHTTPHeader(resp.Header),
		serverAddr: conn.RemoteAddr().String(),
	}, nil
}
