package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/storage"
)

type timingCapture struct {
	requestStart  time.Time
	requestEnd    time.Time
	responseStart time.Time
	responseEnd   time.Time
	dnsDuration   int64
	connectTime   time.Time
	connectDone   time.Time
	tlsStart      time.Time
	tlsDone       time.Time
	serverAddr    string
}

type Proxy struct {
	httpServer  *http.Server
	listener    net.Listener
	transport   *http.Transport
	appSettings *config.AppSettings
	certCache   sync.Map
	caCert      *x509.Certificate
	caKey       *rsa.PrivateKey
	running     bool
	mu          sync.RWMutex
	onRequest   func(*storage.Request)
	onComplete  func(*storage.Request)
}

type bufferedConn struct {
	net.Conn
	reader io.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil {
		return c.reader.Read(p)
	}
	return c.Conn.Read(p)
}

type singleConnListener struct {
	conn net.Conn
	done int32
}

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if atomic.CompareAndSwapInt32(&l.done, 0, 1) {
		return l.conn, nil
	}
	return nil, io.EOF
}

func (l *singleConnListener) Close() error {
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	if l.conn == nil {
		return &net.TCPAddr{}
	}
	return l.conn.LocalAddr()
}

func stripHopByHopHeaders(h http.Header) {
	connectionValues := append([]string(nil), h.Values("Connection")...)
	for _, key := range hopByHopHeaders {
		h.Del(key)
	}
	for _, val := range connectionValues {
		for _, hdr := range strings.Split(val, ",") {
			h.Del(strings.TrimSpace(hdr))
		}
	}
}

type websocketCaptureResult struct {
	frames        []storage.WebSocketFrame
	statusCode    int
	responseStart time.Time
	responseEnd   time.Time
	statusReason  string
	respHeaders   storage.Headers
	serverAddr    string
	duration      int64
	respBodySize  int64
	errorMessage  string
}

type wsUpstreamResponse struct {
	conn       net.Conn
	reader     *bufio.Reader
	statusCode int
	statusText string
	headers    http.Header
	serverAddr string
}

// timingCtxKey is the context key for per-request timing capture.
type timingCtxKey struct{}

func New(settings *config.AppSettings) (*Proxy, error) {
	if settings == nil {
		return nil, fmt.Errorf("proxy settings are required")
	}
	p := &Proxy{
		transport: newSharedTransport(),
	}
	p.transport.Proxy = p.proxyResolver()
	if err := p.ApplySettings(settings); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Proxy) ApplySettings(settings *config.AppSettings) error {
	oldSettings := p.appSettingsSnapshot()
	newSettings := config.CloneForRuntime(settings)
	if err := validateProxyPort(newSettings); err != nil {
		return err
	}

	p.mu.Lock()
	p.appSettings = newSettings
	running := p.running
	p.mu.Unlock()

	if !running {
		return nil
	}

	if err := p.restartListenersIfNeeded(oldSettings, newSettings); err != nil {
		p.mu.Lock()
		p.appSettings = oldSettings
		p.mu.Unlock()
		return err
	}

	return nil
}

func (p *Proxy) appSettingsSnapshot() *config.AppSettings {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return config.CloneForRuntime(p.appSettings)
}

func (p *Proxy) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	if err := p.loadOrGenerateCAFromSettings(config.CloneForRuntime(p.appSettings)); err != nil {
		return fmt.Errorf("failed to load CA: %w", err)
	}
	_ = ctx

	settings := config.CloneForRuntime(p.appSettings)
	if err := validateProxyPort(settings); err != nil {
		return err
	}

	enabled := httpProxyListenerEnabled(settings) || socks5ProxyListenerEnabled(settings)
	port := effectivePort(settings)

	if enabled {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to bind unified listener on %s: %w", addr, err)
		}

		server := p.newHTTPServer(port)
		p.httpServer = server
		p.listener = listener
		go p.serveUnified(listener, port)
	}

	p.running = true
	return nil
}

func (p *Proxy) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	httpServer := p.httpServer
	listener := p.listener
	p.httpServer = nil
	p.listener = nil
	p.running = false
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			fmt.Printf("[Proxy] HTTP server shutdown error: %v\n", err)
		}
	}
	if listener != nil {
		_ = listener.Close()
	}

	fmt.Println("[Proxy] Stopped")
	return nil
}

func (p *Proxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *Proxy) SetOnRequest(fn func(*storage.Request)) {
	p.onRequest = fn
}

func (p *Proxy) SetOnComplete(fn func(*storage.Request)) {
	p.onComplete = fn
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	if !p.isClientAllowed(r.RemoteAddr) {
		p.recordMinimalError(r.Method, requestScheme(r), r.Host, requestPort(r), r.URL.String(), http.StatusForbidden, "client access denied", r.RemoteAddr, "", startTime)
		http.Error(w, "client access denied", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodConnect {
		p.handleConnect(w, r, startTime)
		return
	}

	p.handleHTTP(w, r, startTime)
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	if isWebSocketHandshakeRequest(r.Header) {
		p.handleHTTPWebSocket(w, r, startTime)
		return
	}

	record := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        startTime,
		Method:           r.Method,
		Scheme:           requestScheme(r),
		URL:              r.URL.String(),
		Host:             r.Host,
		Port:             requestPort(r),
		Path:             r.URL.Path,
		QueryString:      r.URL.RawQuery,
		HTTPVersion:      r.Proto,
		Headers:          make(storage.Headers),
		Cookies:          extractRequestCookies(r),
		ContentType:      r.Header.Get("Content-Type"),
		RemoteAddr:       r.RemoteAddr,
		ClientAddr:       r.RemoteAddr,
		KeepAlive:        !r.Close,
		RequestStartTime: startTime,
	}

	for k, v := range r.Header {
		record.Headers[k] = v
	}

	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		record.Body = body
		record.BodySize = int64(len(body))
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	record.RequestEndTime = time.Now()
	record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
	p.applyRequestRules(r.Header)
	p.saveRequestStart(record)

	outboundURL := record.URL
	if outboundURL == "" && r.URL != nil {
		outboundURL = r.URL.String()
	}
	var outboundBody io.Reader
	if record.Body != nil {
		outboundBody = bytes.NewReader(record.Body)
	}

	timingInfo := &timingCapture{}
	ctx := context.WithValue(r.Context(), timingCtxKey{}, timingInfo)
	outboundReq, err := http.NewRequestWithContext(ctx, r.Method, outboundURL, outboundBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		record.StatusCode = http.StatusBadGateway
		record.Duration = time.Since(startTime).Milliseconds()
		applyTiming(record, nil, time.Time{}, time.Time{})
		record.Error = err.Error()
		p.saveRequestStart(record)
		return
	}
	outboundReq.Header = r.Header.Clone()
	outboundReq.Host = r.Host

	client := p.sharedHTTPClient()
	resp, err := client.Do(outboundReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		record.StatusCode = http.StatusBadGateway
		record.Duration = time.Since(startTime).Milliseconds()
		applyTiming(record, timingInfo, time.Time{}, time.Time{})
		record.Error = err.Error()
		p.saveRequestStart(record)
		return
	}
	defer resp.Body.Close()
	record.ResponseStartTime = time.Now()

	record.StatusCode = resp.StatusCode
	record.StatusReason = resp.Status
	record.RespHeaders = make(storage.Headers)
	record.RespContentType = resp.Header.Get("Content-Type")
	record.KeepAlive = record.KeepAlive && !resp.Close
	record.ServerAddr = resolveServerAddr(record.ServerAddr, timingInfo.serverAddr, resp.Request)
	applyTLSMetadata(record, resp.TLS)

	for k, v := range resp.Header {
		record.RespHeaders[k] = v
	}
	p.applyResponseRules(resp.Header)

	// Copy response headers to client before streaming body
	stripHopByHopHeaders(resp.Header)
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	p.applyThrottle(resp.ContentLength)
	w.WriteHeader(resp.StatusCode)

	// Stream response body to client while capturing for recording
	capture := &captureWriter{limit: maxCaptureBodyBytes}
	_, _ = io.Copy(w, io.TeeReader(resp.Body, capture))

	record.ResponseEndTime = time.Now()
	record.RespBody = capture.buf.Bytes()
	record.RespBodySize = capture.total
	record.Duration = time.Since(startTime).Milliseconds()
	applyTiming(record, timingInfo, record.ResponseStartTime, record.ResponseEndTime)

	p.saveRequestComplete(record)
}

func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	host := r.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		p.recordMinimalError("CONNECT", "https", r.Host, 443, "https://"+r.Host, http.StatusInternalServerError, "hijacking not supported", r.RemoteAddr, "", startTime)
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		p.recordMinimalError("CONNECT", "https", r.Host, 443, "https://"+r.Host, http.StatusInternalServerError, fmt.Sprintf("hijack failed: %v", err), r.RemoteAddr, "", startTime)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		p.recordMinimalError("CONNECT", "https", r.Host, 443, "https://"+r.Host, http.StatusInternalServerError, fmt.Sprintf("failed to write connect response: %v", err), r.RemoteAddr, "", startTime)
		return
	}

	if p.mitmEnabled() {
		p.handleMITM(clientConn, r.Host, startTime)
		return
	}

	reqEndTime := time.Now()
	record := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        startTime,
		Method:           http.MethodConnect,
		Scheme:           "https",
		Host:             r.Host,
		Port:             requestPort(r),
		Path:             "",
		URL:              "https://" + r.Host,
		HTTPVersion:      r.Proto,
		Headers:          toStorageHeaders(r.Header),
		RemoteAddr:       r.RemoteAddr,
		ClientAddr:       r.RemoteAddr,
		ServerAddr:       host,
		KeepAlive:        true,
		RequestStartTime: startTime,
		RequestEndTime:   reqEndTime,
		RequestDuration:  safeDurationMillis(startTime, reqEndTime),
		StatusCode:       http.StatusOK,
		StatusReason:     "200 Connection Established",
	}
	p.saveRequestStart(record)

	targetConn, err := p.dialTunnelTarget(context.Background(), host)
	if err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Error = err.Error()
		record.Duration = time.Since(startTime).Milliseconds()
		record.ResponseStartTime = time.Now()
		record.ResponseEndTime = record.ResponseStartTime
		p.saveRequestStart(record)
		return
	}
	defer targetConn.Close()

	p.tunnel(clientConn, targetConn)
}

func (p *Proxy) handleMITM(clientConn net.Conn, host string, startTime time.Time) {
	clientConn, clientHelloRaw := wrapConnWithCapturedClientHello(clientConn)

	targetHost := host
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}

	cert, err := p.getCert(hostname)
	if err != nil {
		fmt.Printf("[Proxy] Failed to get cert for %s: %v\n", hostname, err)
		targetConn, dialErr := p.dialTunnelTarget(context.Background(), host)
		if dialErr != nil {
			fmt.Printf("[Proxy] Tunnel fallback failed for %s: %v\n", host, dialErr)
			p.recordMinimalError("CONNECT", "https", hostname, 443, "https://"+host, 0, fmt.Sprintf("cert generation failed: %v; tunnel fallback failed: %v", err, dialErr), clientConn.RemoteAddr().String(), host, startTime)
			return
		}
		defer targetConn.Close()
		p.tunnel(clientConn, targetConn)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}

	tlsClientConn := tls.Server(clientConn, tlsConfig)
	if err := tlsClientConn.Handshake(); err != nil {
		fmt.Printf("[Proxy] TLS handshake failed: %v\n", err)
		p.recordMinimalError("CONNECT", "https", hostname, 443, "https://"+host, 0, err.Error(), clientConn.RemoteAddr().String(), host, startTime)
		return
	}
	defer tlsClientConn.Close()

	reader := bufio.NewReader(tlsClientConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[Proxy] Read request error: %v\n", err)
				p.recordMinimalError("-", "https", hostname, 443, "https://"+host, 0, fmt.Sprintf("malformed request: %v", err), clientConn.RemoteAddr().String(), host, startTime)
			}
			return
		}

		req.URL.Scheme = "https"
		req.URL.Host = targetHost
		if req.Host == "" {
			req.Host = targetHost
		}

		reqStartTime := time.Now()
		if isWebSocketHandshakeRequest(req.Header) {
			p.handleMITMWebSocket(tlsClientConn, reader, hostname, targetHost, req, reqStartTime)
			return
		}

		record := &storage.Request{
			ID:                uuid.New().String(),
			CreatedAt:         reqStartTime,
			Method:            req.Method,
			URL:               req.URL.String(),
			Host:              hostname,
			Port:              requestPort(req),
			Path:              req.URL.Path,
			QueryString:       req.URL.RawQuery,
			HTTPVersion:       req.Proto,
			Headers:           make(storage.Headers),
			Cookies:           extractRequestCookies(req),
			ContentType:       req.Header.Get("Content-Type"),
			Scheme:            "https",
			RemoteAddr:        clientConn.RemoteAddr().String(),
			ClientAddr:        clientConn.RemoteAddr().String(),
			ServerAddr:        targetHost,
			KeepAlive:         !req.Close,
			RequestStartTime:  reqStartTime,
			TLSClientHelloRaw: append([]byte(nil), clientHelloRaw...),
		}

		for k, v := range req.Header {
			record.Headers[k] = v
		}

		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			record.Body = body
			record.BodySize = int64(len(body))
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
		record.RequestEndTime = time.Now()
		record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
		p.applyRequestRules(req.Header)
		p.saveRequestStart(record)

		req.RequestURI = ""

		resp, err := p.ExecuteRequestWithClientHello(req, clientHelloRaw)
		if err != nil {
			fmt.Printf("[Proxy] Request error: %v\n", err)
			record.StatusCode = http.StatusBadGateway
			record.Error = err.Error()
			record.Duration = time.Since(reqStartTime).Milliseconds()
			errResp := fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nProxy error: %v", err)
			_, _ = tlsClientConn.Write([]byte(errResp))
			p.saveRequestStart(record)
			return
		}
		record.ResponseStartTime = time.Now()

		record.StatusCode = resp.StatusCode
		record.StatusReason = resp.Status
		record.RespHeaders = make(storage.Headers)
		record.RespContentType = resp.Header.Get("Content-Type")
		record.KeepAlive = record.KeepAlive && !resp.Close
		applyTLSMetadata(record, resp.TLS)

		for k, v := range resp.Header {
			record.RespHeaders[k] = v
		}
		p.applyResponseRules(resp.Header)

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		record.ResponseEndTime = time.Now()
		record.RespBody = respBody
		record.RespBodySize = int64(len(respBody))
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.ConnectDuration = 0
		record.DNSDuration = 0
		record.TLSHandshakeDuration = 0
		record.ResponseDuration = safeDurationMillis(record.ResponseStartTime, record.ResponseEndTime)
		record.LatencyDuration = safeDurationMillis(record.RequestEndTime, record.ResponseStartTime)

		stripHopByHopHeaders(resp.Header)
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(respBody)))
		resp.Header.Del("Transfer-Encoding")

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "%s %s\r\n", resp.Proto, resp.Status)
		resp.Header.Write(&buf)
		buf.Write([]byte("\r\n"))
		buf.Write(respBody)
		p.applyThrottle(record.RespBodySize)

		tlsClientConn.Write(buf.Bytes())

		p.saveRequestComplete(record)

		if req.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
			return
		}
	}
}

func (p *Proxy) handleSOCKS5MITM(clientConn net.Conn, clientReader *bufio.Reader, host string, startTime time.Time) {
	p.handleMITM(&bufferedConn{Conn: clientConn, reader: clientReader}, host, startTime)
}

func (p *Proxy) tunnel(client, target net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(target, client)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(client, target)
		done <- struct{}{}
	}()

	<-done
}

func (p *Proxy) recordingEnabled() bool {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return true
	}
	return settings.Proxy.Recording.Enabled
}

func (p *Proxy) mitmEnabled() bool {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return false
	}
	return settings.Proxy.Listener.MITMEnabled && settings.Proxy.SSLProxying.Enabled
}

func (p *Proxy) isClientAllowed(remoteAddr string) bool {
	settings := p.appSettingsSnapshot()
	if settings == nil || !settings.Proxy.AccessControl.Enabled || len(settings.Proxy.AccessControl.AllowedClients) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	for _, allowed := range settings.Proxy.AccessControl.AllowedClients {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if host == allowed || remoteAddr == allowed {
			return true
		}
	}
	return false
}

func (p *Proxy) applyRequestRules(header http.Header) {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return
	}
	if settings.Tools.NoCaching {
		header.Set("Cache-Control", "no-cache")
		header.Set("Pragma", "no-cache")
		header.Del("If-Modified-Since")
		header.Del("If-None-Match")
	}
	if settings.Tools.BlockCookies {
		header.Del("Cookie")
	}
}

func (p *Proxy) applyResponseRules(header http.Header) {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return
	}
	if settings.Tools.NoCaching {
		header.Set("Cache-Control", "no-store, no-cache, must-revalidate")
		header.Set("Pragma", "no-cache")
		header.Set("Expires", "0")
	}
	if settings.Tools.BlockCookies {
		header.Del("Set-Cookie")
	}
}

func (p *Proxy) applyThrottle(bodySize int64) {
	settings := p.appSettingsSnapshot()
	if settings == nil || !settings.Proxy.Throttling.Enabled {
		return
	}
	if settings.Proxy.Throttling.LatencyMs > 0 {
		time.Sleep(time.Duration(settings.Proxy.Throttling.LatencyMs) * time.Millisecond)
	}
	if settings.Proxy.Throttling.DownstreamKBPS > 0 && bodySize > 0 {
		bytesPerSecond := int64(settings.Proxy.Throttling.DownstreamKBPS) * 1024
		delay := time.Duration(float64(bodySize) / float64(bytesPerSecond) * float64(time.Second))
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

const (
	socks5Version  = 5
	socks5NoAuth   = 0
	socks5Connect  = 1
	socks5UserPass = 2

	// SOCKS5 reply codes
	socks5RepSuccess           = 0x00
	socks5RepConnectionRefused = 0x05

	// SOCKS5 address types
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04

	// maxCaptureBodyBytes is the maximum body size to capture (5MB)
	maxCaptureBodyBytes = 5 * 1024 * 1024
)

func (p *Proxy) restartListenersIfNeeded(oldSettings, newSettings *config.AppSettings) error {
	if err := validateProxyPort(newSettings); err != nil {
		return err
	}

	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()
	if !running {
		return nil
	}

	oldEnabled := httpProxyListenerEnabled(oldSettings) || socks5ProxyListenerEnabled(oldSettings)
	newEnabled := httpProxyListenerEnabled(newSettings) || socks5ProxyListenerEnabled(newSettings)
	oldPort := effectivePort(oldSettings)
	newPort := effectivePort(newSettings)
	if oldEnabled == newEnabled && oldPort == newPort {
		return nil
	}

	var newListener net.Listener
	if newEnabled {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", newPort))
		if err != nil {
			return fmt.Errorf("failed to bind unified listener on port %d: %w", newPort, err)
		}
		newListener = listener
	}

	var (
		oldHTTPServer *http.Server
		oldListener   net.Listener
	)

	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		if newListener != nil {
			_ = newListener.Close()
		}
		return nil
	}

	oldHTTPServer = p.httpServer
	oldListener = p.listener
	p.httpServer = nil
	p.listener = nil
	if newEnabled {
		server := p.newHTTPServer(newPort)
		p.httpServer = server
		p.listener = newListener
		go p.serveUnified(newListener, newPort)
		newListener = nil
	}
	p.mu.Unlock()

	if oldHTTPServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := oldHTTPServer.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("[Proxy] Old HTTP shutdown error: %v\n", err)
		}
		cancel()
	}
	if oldListener != nil {
		_ = oldListener.Close()
		fmt.Printf("[Proxy] Old unified listener shut down\n")
	}

	if newListener != nil {
		_ = newListener.Close()
	}

	return nil
}

func validateProxyPort(settings *config.AppSettings) error {
	if settings == nil {
		return nil
	}
	if !settings.Proxy.Listener.HTTPEnabled && !settings.Proxy.Listener.HTTPSEnabled && !settings.Proxy.Listener.SOCKS5Enabled {
		return nil
	}
	port := settings.Proxy.Listener.Port
	if port <= 0 || port >= 65536 {
		return fmt.Errorf("invalid unified proxy port: %d", port)
	}
	return nil
}

func httpProxyListenerEnabled(settings *config.AppSettings) bool {
	if settings == nil {
		return true
	}
	return settings.Proxy.Listener.HTTPEnabled || settings.Proxy.Listener.HTTPSEnabled
}

func socks5ProxyListenerEnabled(settings *config.AppSettings) bool {
	if settings == nil {
		return true
	}
	return settings.Proxy.Listener.SOCKS5Enabled
}

func effectivePort(settings *config.AppSettings) int {
	if settings != nil && settings.Proxy.Listener.Port > 0 {
		return settings.Proxy.Listener.Port
	}
	return 8888
}

func (p *Proxy) newHTTPServer(port int) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      p,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 0,
	}
}

func (p *Proxy) serveUnified(listener net.Listener, port int) {
	fmt.Printf("[Proxy] Starting unified proxy on :%d (HTTP/HTTPS/SOCKS5)\n", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if p.IsRunning() && !errors.Is(err, net.ErrClosed) {
				fmt.Printf("[Proxy] Accept error: %v\n", err)
			}
			return
		}
		go p.detectAndHandle(conn)
	}
}

func (p *Proxy) detectAndHandle(conn net.Conn) {
	reader := bufio.NewReader(conn)
	peekByte, err := reader.Peek(1)
	if err != nil {
		_ = conn.Close()
		return
	}

	buffered := &bufferedConn{Conn: conn, reader: reader}
	if peekByte[0] == socks5Version {
		if !p.socks5Enabled() {
			_ = conn.Close()
			return
		}
		p.handleSOCKS5Connection(buffered)
		return
	}

	if !p.httpEnabled() {
		_ = conn.Close()
		return
	}

	p.mu.RLock()
	server := p.httpServer
	p.mu.RUnlock()
	if server == nil {
		_ = conn.Close()
		return
	}

	single := &singleConnListener{conn: buffered}
	if err := server.Serve(single); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, io.EOF) {
		fmt.Printf("[Proxy] HTTP serve conn error: %v\n", err)
	}
}

func (p *Proxy) httpEnabled() bool {
	settings := p.appSettingsSnapshot()
	return httpProxyListenerEnabled(settings)
}

func (p *Proxy) socks5Enabled() bool {
	settings := p.appSettingsSnapshot()
	return socks5ProxyListenerEnabled(settings)
}

func (p *Proxy) handleSOCKS5Connection(conn net.Conn) {
	defer conn.Close()
	startTime := time.Now()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	if err := p.socks5Handshake(conn); err != nil {
		fmt.Printf("[SOCKS5] Handshake failed: %v\n", err)
		p.recordMinimalError("-", "socks5", "", 0, "", 0, fmt.Sprintf("socks5 handshake: %v", err), conn.RemoteAddr().String(), "", startTime)
		return
	}

	targetAddr, err := p.socks5ReadRequest(conn)
	if err != nil {
		fmt.Printf("[SOCKS5] Read request failed: %v\n", err)
		p.recordMinimalError("-", "socks5", "", 0, "", 0, fmt.Sprintf("socks5 request: %v", err), conn.RemoteAddr().String(), "", startTime)
		return
	}

	conn.SetDeadline(time.Time{})

	p.socks5SendReply(conn, socks5RepSuccess)

	// Peek at the first bytes to detect protocol
	clientReader := bufio.NewReader(conn)
	peekBytes, err := clientReader.Peek(8)
	if err != nil {
		targetConn, dialErr := p.dialTunnelTarget(context.Background(), targetAddr)
		if dialErr != nil {
			p.recordSOCKS5Request(targetAddr, conn.RemoteAddr().String(), startTime, http.StatusBadGateway, 0, 0)
			return
		}
		defer targetConn.Close()
		// If we can't peek, fall back to raw tunnel
		p.tunnelSOCKS5(conn, clientReader, targetConn, targetAddr, startTime)
		return
	}

	// Check if this is TLS and MITM is enabled
	if isTLSHandshake(peekBytes) && p.mitmEnabled() {
		fmt.Printf("[SOCKS5] Detected TLS traffic for %s, MITM enabled\n", targetAddr)
		p.handleSOCKS5MITM(conn, clientReader, targetAddr, startTime)
		return
	}

	targetConn, err := p.dialTunnelTarget(context.Background(), targetAddr)
	if err != nil {
		p.recordSOCKS5Request(targetAddr, conn.RemoteAddr().String(), startTime, http.StatusBadGateway, 0, 0)
		return
	}
	defer targetConn.Close()

	// Check if this is HTTP traffic
	if looksLikeHTTP(peekBytes) {
		fmt.Printf("[SOCKS5] Detected HTTP traffic for %s\n", targetAddr)
		p.handleSOCKS5HTTP(conn, targetConn, targetAddr, startTime, clientReader)
		return
	}

	// Default: raw tunnel for unknown protocols or TLS without MITM
	p.tunnelSOCKS5(conn, clientReader, targetConn, targetAddr, startTime)
}

func (p *Proxy) handleSOCKS5HTTP(client, target net.Conn, targetAddr string, startTime time.Time, clientReader *bufio.Reader) {
	host, port := parseHostPort(targetAddr)
	remoteAddr := client.RemoteAddr().String()
	targetReader := bufio.NewReader(target)
	scheme := "http"
	if port == 443 {
		scheme = "https"
	}

	for {
		// Set deadline for reading request
		client.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read HTTP request
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[SOCKS5-HTTP] Read request error: %v\n", err)
				p.recordMinimalError("-", scheme, host, port, fmt.Sprintf("%s://%s", scheme, targetAddr), 0, fmt.Sprintf("read request: %v", err), remoteAddr, targetAddr, startTime)
			}
			return
		}

		reqStartTime := time.Now()
		if isWebSocketHandshakeRequest(req.Header) {
			p.handleSOCKS5WebSocket(client, target, targetAddr, startTime, clientReader, req, reqStartTime, targetReader)
			return
		}

		// Build full URL
		fullURL := fmt.Sprintf("%s://%s%s", scheme, host, req.RequestURI)

		// Capture request body
		var reqBodyBuf *limitedWriter
		var reqBodyBytes int64
		if req.Body != nil {
			reqBodyBuf = newLimitedWriter(maxCaptureBodyBytes)
			// Use TeeReader to capture while reading
			teeReader := io.TeeReader(req.Body, reqBodyBuf)
			bodyBytes, _ := io.ReadAll(teeReader)
			reqBodyBytes = int64(len(bodyBytes))
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Prepare request for forwarding
		req.RequestURI = ""
		req.URL.Scheme = scheme
		req.URL.Host = host
		if port != 0 && port != 80 && port != 443 {
			req.URL.Host = fmt.Sprintf("%s:%d", host, port)
		}

		// Build headers map
		reqHeaders := make(storage.Headers)
		for k, v := range req.Header {
			reqHeaders[k] = v
		}

		// Build cookies
		cookies := make(storage.Cookies)
		for _, c := range req.Cookies() {
			cookies[c.Name] = c.Value
		}

		record := &storage.Request{
			ID:               uuid.New().String(),
			CreatedAt:        reqStartTime,
			Method:           req.Method,
			Scheme:           scheme,
			URL:              fullURL,
			Host:             host,
			Port:             port,
			Path:             req.URL.Path,
			QueryString:      req.URL.RawQuery,
			HTTPVersion:      req.Proto,
			Headers:          reqHeaders,
			Cookies:          cookies,
			ContentType:      req.Header.Get("Content-Type"),
			BodySize:         reqBodyBytes,
			RemoteAddr:       remoteAddr,
			ClientAddr:       remoteAddr,
			ServerAddr:       target.RemoteAddr().String(),
			KeepAlive:        !req.Close,
			RequestStartTime: reqStartTime,
		}
		if reqBodyBuf != nil {
			record.Body = reqBodyBuf.Bytes()
		}
		record.RequestEndTime = time.Now()
		record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
		p.saveRequestStart(record)

		// Forward request to target
		target.SetDeadline(time.Now().Add(30 * time.Second))
		if err := req.Write(target); err != nil {
			fmt.Printf("[SOCKS5-HTTP] Write to target error: %v\n", err)
			record.StatusCode = http.StatusBadGateway
			record.Error = err.Error()
			record.Duration = time.Since(reqStartTime).Milliseconds()
			record.ResponseStartTime = time.Now()
			record.ResponseEndTime = record.ResponseStartTime
			p.saveRequestStart(record)
			return
		}

		// Read response from target
		resp, err := http.ReadResponse(targetReader, req)
		if err != nil {
			fmt.Printf("[SOCKS5-HTTP] Read response error: %v\n", err)
			record.StatusCode = http.StatusBadGateway
			record.StatusReason = http.StatusText(http.StatusBadGateway)
			record.Error = err.Error()
			record.Duration = time.Since(reqStartTime).Milliseconds()
			record.ResponseStartTime = time.Now()
			record.ResponseEndTime = record.ResponseStartTime
			p.saveRequestStart(record)
			return
		}

		// Capture response body
		var respBodyBuf *limitedWriter
		var respBodyBytes int64
		if resp.Body != nil {
			respBodyBuf = newLimitedWriter(maxCaptureBodyBytes)
			teeReader := io.TeeReader(resp.Body, respBodyBuf)
			bodyBytes, _ := io.ReadAll(teeReader)
			respBodyBytes = int64(len(bodyBytes))
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		p.applyRequestRules(req.Header)
		p.applyResponseRules(resp.Header)

		// Write response to client
		client.SetDeadline(time.Now().Add(30 * time.Second))
		p.applyThrottle(respBodyBytes)
		if err := resp.Write(client); err != nil {
			fmt.Printf("[SOCKS5-HTTP] Write to client error: %v\n", err)
			return
		}

		respHeaders := make(storage.Headers)
		for k, v := range resp.Header {
			respHeaders[k] = v
		}

		record.StatusCode = resp.StatusCode
		record.StatusReason = resp.Status
		record.RespHeaders = respHeaders
		record.RespContentType = resp.Header.Get("Content-Type")
		record.RespBodySize = respBodyBytes
		record.ResponseStartTime = record.RequestEndTime
		if respBodyBuf != nil {
			record.RespBody = respBodyBuf.Bytes()
		}
		record.ResponseEndTime = time.Now()
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.ResponseDuration = safeDurationMillis(record.ResponseStartTime, record.ResponseEndTime)
		record.LatencyDuration = safeDurationMillis(record.RequestEndTime, record.ResponseStartTime)
		record.KeepAlive = !req.Close && !resp.Close

		p.saveRequestComplete(record)

		// Check for connection close
		if req.Close || resp.Close {
			return
		}

		// Reset deadline for next request
		client.SetDeadline(time.Time{})
		target.SetDeadline(time.Time{})
	}
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return ""
	}
	if r.URL != nil && r.URL.Scheme != "" {
		return r.URL.Scheme
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func requestPort(r *http.Request) int {
	if r == nil {
		return 0
	}
	host := r.Host
	if r.URL != nil && r.URL.Host != "" {
		host = r.URL.Host
	}
	_, port, err := net.SplitHostPort(host)
	if err == nil {
		var parsed int
		fmt.Sscanf(port, "%d", &parsed)
		return parsed
	}
	if requestScheme(r) == "https" {
		return 443
	}
	return 80
}

func serverNameFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func resolveServerAddr(existing, fallback string, req *http.Request) string {
	if existing != "" {
		return existing
	}
	if fallback != "" {
		return fallback
	}
	if req != nil && req.URL != nil {
		return req.URL.Host
	}
	return ""
}

func safeDurationMillis(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func applyTLSMetadata(record *storage.Request, state *tls.ConnectionState) {
	if record == nil || state == nil {
		return
	}
	record.TLSVersion = tlsVersionString(state.Version)
	record.TLSCipherSuite = tls.CipherSuiteName(state.CipherSuite)
	record.TLSServerName = state.ServerName
	record.TLSDidResume = state.DidResume
	record.TLSALPN = state.NegotiatedProtocol
	record.TLSCurveID = curveIDString(state.CurveID)
	record.TLSOCSPStapled = len(state.OCSPResponse) > 0
	record.TLSSCTCount = len(state.SignedCertificateTimestamps)
	record.TLSServerCertificates = buildTLSCertificates(state.PeerCertificates)
	record.TLSServerExtensions = buildTLSConnectionExtensions(state)
}

func applyTiming(record *storage.Request, info *timingCapture, responseStart, responseEnd time.Time) {
	if record == nil {
		return
	}
	if info != nil {
		record.ServerAddr = resolveServerAddr(record.ServerAddr, info.serverAddr, nil)
		record.ConnectDuration = safeDurationMillis(info.connectTime, info.connectDone)
		record.TLSHandshakeDuration = safeDurationMillis(info.tlsStart, info.tlsDone)
		record.DNSDuration = 0
	}
	if !responseStart.IsZero() {
		record.ResponseStartTime = responseStart
	}
	if !responseEnd.IsZero() {
		record.ResponseEndTime = responseEnd
	}
	record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
	record.ResponseDuration = safeDurationMillis(record.ResponseStartTime, record.ResponseEndTime)
	record.LatencyDuration = safeDurationMillis(record.RequestEndTime, record.ResponseStartTime)
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLSv1.0"
	case tls.VersionTLS11:
		return "TLSv1.1"
	case tls.VersionTLS12:
		return "TLSv1.2"
	case tls.VersionTLS13:
		return "TLSv1.3"
	default:
		if version == 0 {
			return ""
		}
		return fmt.Sprintf("TLS(0x%04x)", version)
	}
}

func curveIDString(id tls.CurveID) string {
	if id == 0 {
		return ""
	}
	return id.String()
}

func isWebSocketHandshakeRequest(header http.Header) bool {
	if header == nil {
		return false
	}
	connection := strings.ToLower(header.Get("Connection"))
	upgrade := strings.ToLower(header.Get("Upgrade"))
	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}

func extractRequestCookies(r *http.Request) storage.Cookies {
	if r == nil {
		return nil
	}
	cookies := make(storage.Cookies)
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}
	if len(cookies) == 0 {
		return nil
	}
	return cookies
}

func websocketSchemeFromRequest(r *http.Request, defaultScheme string) string {
	if r != nil && r.TLS != nil {
		return "wss"
	}
	if defaultScheme == "https" {
		return "wss"
	}
	return "ws"
}

func websocketFrameType(opcode int) string {
	switch opcode {
	case websocket.TextMessage:
		return "text"
	case websocket.BinaryMessage:
		return "binary"
	case websocket.CloseMessage:
		return "close"
	case websocket.PingMessage:
		return "ping"
	case websocket.PongMessage:
		return "pong"
	default:
		return fmt.Sprintf("opcode-%d", opcode)
	}
}

func cloneHTTPHeader(header http.Header) http.Header {
	if header == nil {
		return make(http.Header)
	}
	return header.Clone()
}

func toStorageHeaders(header http.Header) storage.Headers {
	result := make(storage.Headers)
	for k, v := range header {
		result[k] = append([]string(nil), v...)
	}
	return result
}

func cloneWebSocketRequest(req *http.Request) *http.Request {
	if req == nil {
		return nil
	}
	clone := req.Clone(req.Context())
	clone.Header = cloneHTTPHeader(req.Header)
	clone.URL = new(url.URL)
	*clone.URL = *req.URL
	clone.RequestURI = req.RequestURI
	clone.Host = req.Host
	clone.Body = nil
	return clone
}

func (p *Proxy) websocketCaptureLimitBytes() int64 {
	settings := p.appSettingsSnapshot()
	if settings == nil || settings.Proxy.Recording.MaxCaptureBodySizeMB <= 0 {
		return maxCaptureBodyBytes
	}
	return int64(settings.Proxy.Recording.MaxCaptureBodySizeMB) * 1024 * 1024
}

func (p *Proxy) truncateWebSocketPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return nil
	}
	limit := p.websocketCaptureLimitBytes()
	if limit <= 0 || int64(len(payload)) <= limit {
		return append([]byte(nil), payload...)
	}
	return append([]byte(nil), payload[:limit]...)
}

func (p *Proxy) appendWebSocketFrame(frames []storage.WebSocketFrame, direction string, opcode int, payload []byte, masked bool, at time.Time) []storage.WebSocketFrame {
	return append(frames, storage.WebSocketFrame{
		ID:          uuid.New().String(),
		Direction:   direction,
		Opcode:      opcode,
		FrameType:   websocketFrameType(opcode),
		Payload:     p.truncateWebSocketPayload(payload),
		PayloadSize: int64(len(payload)),
		CreatedAt:   at,
		Fin:         true,
		Masked:      masked,
	})
}

type websocketRawFrame struct {
	raw     []byte
	payload []byte
	opcode  int
	fin     bool
	masked  bool
}

func buildTLSCertificates(certs []*x509.Certificate) []storage.TLSCertificate {
	if len(certs) == 0 {
		return nil
	}
	out := make([]storage.TLSCertificate, 0, len(certs))
	for _, cert := range certs {
		if cert == nil {
			continue
		}
		out = append(out, storage.TLSCertificate{
			SubjectCommonName:     cert.Subject.CommonName,
			Subject:               cert.Subject.String(),
			IssuerCommonName:      cert.Issuer.CommonName,
			Issuer:                cert.Issuer.String(),
			SerialNumber:          cert.SerialNumber.Text(16),
			DNSNames:              append([]string(nil), cert.DNSNames...),
			EmailAddresses:        append([]string(nil), cert.EmailAddresses...),
			IPAddresses:           stringIPs(cert.IPAddresses),
			Version:               cert.Version,
			IsCA:                  cert.IsCA,
			SignatureAlgorithm:    cert.SignatureAlgorithm.String(),
			PublicKeyAlgorithm:    publicKeyAlgorithmLabel(cert),
			NotBefore:             cert.NotBefore,
			NotAfter:              cert.NotAfter,
			OCSPServers:           append([]string(nil), cert.OCSPServer...),
			IssuingCertificateURL: append([]string(nil), cert.IssuingCertificateURL...),
			Extensions:            buildTLSCertificateExtensions(cert),
		})
	}
	return out
}

func buildTLSConnectionExtensions(state *tls.ConnectionState) []storage.TLSExtension {
	if state == nil {
		return nil
	}
	exts := make([]storage.TLSExtension, 0, 5)
	if state.ServerName != "" {
		exts = append(exts, storage.TLSExtension{ID: 0, Name: "server_name", Value: state.ServerName})
	}
	if state.NegotiatedProtocol != "" {
		exts = append(exts, storage.TLSExtension{ID: 16, Name: "application_layer_protocol_negotiation", Value: state.NegotiatedProtocol})
	}
	if version := tlsVersionString(state.Version); version != "" {
		exts = append(exts, storage.TLSExtension{ID: 43, Name: "supported_versions", Value: version})
	}
	if curve := curveIDString(state.CurveID); curve != "" {
		exts = append(exts, storage.TLSExtension{ID: 51, Name: "key_share", Value: curve})
	}
	if state.DidResume {
		exts = append(exts, storage.TLSExtension{ID: 41, Name: "pre_shared_key", Value: "session resumed"})
	}
	return exts
}

func buildTLSCertificateExtensions(cert *x509.Certificate) []storage.TLSExtension {
	if cert == nil || len(cert.Extensions) == 0 {
		return nil
	}
	exts := make([]storage.TLSExtension, 0, len(cert.Extensions))
	for _, ext := range cert.Extensions {
		exts = append(exts, storage.TLSExtension{
			ID:    oidLastInt(ext.Id),
			Name:  tlsExtensionName(ext.Id.String()),
			Value: hex.EncodeToString(ext.Value),
		})
	}
	return exts
}

func tlsExtensionName(oid string) string {
	switch oid {
	case "2.5.29.14":
		return "subject_key_identifier"
	case "2.5.29.15":
		return "key_usage"
	case "2.5.29.17":
		return "subject_alternative_name"
	case "2.5.29.19":
		return "basic_constraints"
	case "2.5.29.31":
		return "crl_distribution_points"
	case "2.5.29.35":
		return "authority_key_identifier"
	case "1.3.6.1.5.5.7.1.1":
		return "authority_information_access"
	case "1.3.6.1.5.5.7.1.11":
		return "subject_information_access"
	default:
		return oid
	}
}

func oidLastInt(oid any) int {
	stringer, ok := oid.(interface{ String() string })
	if !ok {
		return 0
	}
	parts := strings.Split(stringer.String(), ".")
	if len(parts) == 0 {
		return 0
	}
	var value int
	fmt.Sscanf(parts[len(parts)-1], "%d", &value)
	return value
}

func stringIPs(ips []net.IP) []string {
	if len(ips) == 0 {
		return nil
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func publicKeyAlgorithmLabel(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	base := cert.PublicKeyAlgorithm.String()
	switch key := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return fmt.Sprintf("%s (%d bits)", base, key.N.BitLen())
	case *ecdsa.PublicKey:
		return fmt.Sprintf("%s (%d bits)", base, key.Params().BitSize)
	case ed25519.PublicKey:
		return fmt.Sprintf("%s (%d bits)", base, len(key)*8)
	case *ecdh.PublicKey:
		return fmt.Sprintf("%s (%d bytes)", base, len(key.Bytes()))
	default:
		return base
	}
}
