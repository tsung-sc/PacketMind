package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/packetmind/packetmind/internal/storage"
)

func TestHandleSOCKS5Connection_PreservesPeekedTLSBytes(t *testing.T) {
	store, err := storage.NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store

	p := New()
	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.MITMEnabled = false
	settings.Proxy.SSLProxying.Enabled = false
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}
	p.running = true

	targetListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen target failed: %v", err)
	}
	defer targetListener.Close()

	tlsClientHelloPrefix := []byte{0x16, 0x03, 0x01, 0x00, 0x20, 0x01, 0x00, 0x00, 0x1c, 0x03, 0x03}
	upstreamReply := []byte("upstream-response")
	upstreamRead := make(chan []byte, 1)
	upstreamErr := make(chan error, 1)

	go func() {
		conn, err := targetListener.Accept()
		if err != nil {
			upstreamErr <- err
			return
		}
		defer conn.Close()

		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			upstreamErr <- err
			return
		}

		buf := make([]byte, len(tlsClientHelloPrefix))
		if _, err := io.ReadFull(conn, buf); err != nil {
			upstreamErr <- err
			return
		}
		upstreamRead <- buf

		if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
			upstreamErr <- err
			return
		}

		if _, err := conn.Write(upstreamReply); err != nil {
			upstreamErr <- err
			return
		}

		upstreamErr <- nil
	}()

	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen socks failed: %v", err)
	}
	defer socksListener.Close()

	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		conn, err := socksListener.Accept()
		if err != nil {
			return
		}
		p.handleSOCKS5Connection(conn)
	}()

	clientConn, err := net.Dial("tcp", socksListener.Addr().String())
	if err != nil {
		t.Fatalf("dial socks failed: %v", err)
	}
	defer clientConn.Close()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}

	if _, err := clientConn.Write([]byte{socks5Version, 1, socks5NoAuth}); err != nil {
		t.Fatalf("write handshake failed: %v", err)
	}

	handshakeResp := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, handshakeResp); err != nil {
		t.Fatalf("read handshake reply failed: %v", err)
	}
	if !bytes.Equal(handshakeResp, []byte{socks5Version, socks5NoAuth}) {
		t.Fatalf("unexpected handshake reply: %v", handshakeResp)
	}

	host, portString, err := net.SplitHostPort(targetListener.Addr().String())
	if err != nil {
		t.Fatalf("split target addr failed: %v", err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		t.Fatalf("expected IPv4 target host, got %q", host)
	}

	port, err := net.LookupPort("tcp", portString)
	if err != nil {
		t.Fatalf("lookup port failed: %v", err)
	}

	request := []byte{socks5Version, socks5Connect, 0, socks5AtypIPv4}
	request = append(request, ip...)
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(port))
	request = append(request, portBuf...)

	if _, err := clientConn.Write(request); err != nil {
		t.Fatalf("write connect request failed: %v", err)
	}

	connectResp := make([]byte, 10)
	if _, err := io.ReadFull(clientConn, connectResp); err != nil {
		t.Fatalf("read connect reply failed: %v", err)
	}
	if connectResp[1] != socks5RepSuccess {
		t.Fatalf("unexpected connect reply: %v", connectResp)
	}

	if _, err := clientConn.Write(tlsClientHelloPrefix); err != nil {
		t.Fatalf("write tls payload failed: %v", err)
	}

	select {
	case got := <-upstreamRead:
		if !bytes.Equal(got, tlsClientHelloPrefix) {
			t.Fatalf("upstream received %v, want %v", got, tlsClientHelloPrefix)
		}
	case err := <-upstreamErr:
		if err != nil {
			t.Fatalf("upstream read failed: %v", err)
		}
		t.Fatal("upstream finished before payload assertion")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for upstream payload")
	}

	resp := make([]byte, len(upstreamReply))
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("read upstream reply failed: %v", err)
	}
	if !bytes.Equal(resp, upstreamReply) {
		t.Fatalf("unexpected upstream reply: %q", resp)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client failed: %v", err)
	}

	select {
	case err := <-upstreamErr:
		if err != nil {
			t.Fatalf("upstream error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for upstream completion")
	}

	select {
	case <-handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SOCKS5 handler to exit")
	}

	requests, err := store.ListRequests(storage.RequestListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 recorded SOCKS5 request, got len=%d", len(requests))
	}
	if requests[0].RespBodySize != int64(len(upstreamReply)) {
		t.Fatalf("expected response byte count %d, got %d", len(upstreamReply), requests[0].RespBodySize)
	}
}

func TestHandleSOCKS5Connection_MITMRecordsHTTPSRequest(t *testing.T) {
	store, err := storage.NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store

	p := New()
	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.HTTPEnabled = true
	settings.Proxy.Listener.HTTPSEnabled = true
	settings.Proxy.Listener.SOCKS5Enabled = true
	settings.Proxy.Listener.Port = freeTCPPort(t)
	settings.Proxy.Listener.MITMEnabled = true
	settings.Proxy.SSLProxying.Enabled = true
	settings.Cert.Organization = "PacketMind Test"
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	tempDir := t.TempDir()
	if err := p.generateCA(filepath.Join(tempDir, "ca.crt"), filepath.Join(tempDir, "ca.key"), "TestOrg"); err != nil {
		t.Fatalf("generateCA failed: %v", err)
	}
	p.running = true

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream url failed: %v", err)
	}

	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen socks failed: %v", err)
	}
	defer socksListener.Close()

	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		conn, err := socksListener.Accept()
		if err != nil {
			return
		}
		p.handleSOCKS5Connection(conn)
	}()

	clientConn, err := net.Dial("tcp", socksListener.Addr().String())
	if err != nil {
		t.Fatalf("dial socks failed: %v", err)
	}
	defer clientConn.Close()

	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}

	if _, err := clientConn.Write([]byte{socks5Version, 1, socks5NoAuth}); err != nil {
		t.Fatalf("write handshake failed: %v", err)
	}

	handshakeResp := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, handshakeResp); err != nil {
		t.Fatalf("read handshake reply failed: %v", err)
	}
	if !bytes.Equal(handshakeResp, []byte{socks5Version, socks5NoAuth}) {
		t.Fatalf("unexpected handshake reply: %v", handshakeResp)
	}

	host, portString, err := net.SplitHostPort(upstreamURL.Host)
	if err != nil {
		t.Fatalf("split upstream addr failed: %v", err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		t.Fatalf("expected IPv4 upstream host, got %q", host)
	}

	port, err := net.LookupPort("tcp", portString)
	if err != nil {
		t.Fatalf("lookup port failed: %v", err)
	}

	connectReq := []byte{socks5Version, socks5Connect, 0, socks5AtypIPv4}
	connectReq = append(connectReq, ip...)
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(port))
	connectReq = append(connectReq, portBuf...)

	if _, err := clientConn.Write(connectReq); err != nil {
		t.Fatalf("write connect request failed: %v", err)
	}

	connectResp := make([]byte, 10)
	if _, err := io.ReadFull(clientConn, connectResp); err != nil {
		t.Fatalf("read connect reply failed: %v", err)
	}
	if connectResp[1] != socks5RepSuccess {
		t.Fatalf("unexpected connect reply: %v", connectResp)
	}

	tlsConn := tls.Client(clientConn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	})
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("TLS handshake failed: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://"+upstreamURL.Host+"/hello?x=1", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set("Connection", "close")

	if err := req.Write(tlsConn); err != nil {
		t.Fatalf("write https request failed: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), req)
	if err != nil {
		t.Fatalf("read https response failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body failed: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("unexpected response body: %q", body)
	}

	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for SOCKS5 MITM handler to exit")
	}

	requests, err := store.ListRequests(storage.RequestListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 MITM request record, got len=%d", len(requests))
	}
	if requests[0].Method != http.MethodGet {
		t.Fatalf("expected recorded method GET, got %s", requests[0].Method)
	}
	if requests[0].Scheme != "https" {
		t.Fatalf("expected recorded scheme https, got %s", requests[0].Scheme)
	}
	if requests[0].StatusCode != http.StatusOK {
		t.Fatalf("expected recorded status 200, got %d", requests[0].StatusCode)
	}
	if requests[0].Path != "/hello" {
		t.Fatalf("expected recorded path /hello, got %s", requests[0].Path)
	}
	if requests[0].URL != "https://"+upstreamURL.Host+"/hello?x=1" {
		t.Fatalf("unexpected recorded url: %s", requests[0].URL)
	}
}

func TestHandleMITMWebSocket_CapturesFrames(t *testing.T) {
	proxyPort := freeTCPPort(t)
	tmpDir := t.TempDir()
	store, err := storage.NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	p := New()
	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.Port = proxyPort
	settings.Proxy.Listener.HTTPEnabled = true
	settings.Proxy.Listener.HTTPSEnabled = true
	settings.Proxy.Listener.MITMEnabled = true
	settings.Proxy.SSLProxying.Enabled = true
	settings.Cert.CACertFile = filepath.Join(tmpDir, "ca.crt")
	settings.Cert.CAKeyFile = filepath.Join(tmpDir, "ca.key")
	settings.Cert.Organization = "PacketMind Test"
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}
	capturedCh := make(chan *storage.Request, 1)
	p.SetOnRequest(func(req *storage.Request) {
		if req != nil && req.IsWebSocket {
			select {
			case capturedCh <- req:
			default:
			}
		}
	})
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = p.Stop() }()

	if err := waitTCPPort(proxyPort, 2*time.Second); err != nil {
		t.Fatalf("wait proxy port failed: %v", err)
	}

	upgrader := websocket.Upgrader{}
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		mt, message, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.WriteMessage(mt, append([]byte("echo:"), message...)); err != nil {
			return
		}
	}))
	defer upstream.Close()

	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	if err != nil {
		t.Fatalf("parse proxy url failed: %v", err)
	}

	dialer := websocket.Dialer{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	wsURL := "wss" + strings.TrimPrefix(upstream.URL, "https") + "/ws"
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		_ = conn.Close()
		t.Fatalf("write websocket message failed: %v", err)
	}

	_, payload, err := conn.ReadMessage()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("read websocket message failed: %v", err)
	}
	_ = conn.Close()

	if string(payload) != "echo:hello" {
		t.Fatalf("unexpected websocket payload: %q", payload)
	}

	var captured *storage.Request
	select {
	case captured = <-capturedCh:
	case <-time.After(10 * time.Second):
		requests, listErr := storage.Default.ListRequests(storage.RequestListOptions{})
		if listErr != nil {
			t.Fatalf("ListRequests failed while waiting: %v", listErr)
		}
		t.Fatalf("expected websocket request to be captured, requests=%d", len(requests))
	}

	if len(captured.WebSocketFrames) < 2 {
		t.Fatalf("expected websocket frames, got %d", len(captured.WebSocketFrames))
	}
	if captured.WebSocketFrames[0].Direction != "sent" {
		t.Fatalf("expected first frame direction sent, got %s", captured.WebSocketFrames[0].Direction)
	}
	if captured.WebSocketFrames[1].Direction != "received" {
		t.Fatalf("expected second frame direction received, got %s", captured.WebSocketFrames[1].Direction)
	}
	if !bytes.Contains(captured.WebSocketFrames[0].Payload, []byte("hello")) {
		t.Fatalf("expected sent payload to contain hello, got %q", captured.WebSocketFrames[0].Payload)
	}
	if !bytes.Contains(captured.WebSocketFrames[1].Payload, []byte("echo:hello")) {
		t.Fatalf("expected received payload to contain echo:hello, got %q", captured.WebSocketFrames[1].Payload)
	}
}

func TestMITMEnabled_RespectsSSLProxyingToggle(t *testing.T) {
	p := newTestProxyForPort(t, freeTCPPort(t))

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.MITMEnabled = true
	settings.Proxy.SSLProxying.Enabled = false
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}
	if p.mitmEnabled() {
		t.Fatal("expected mitm disabled when ssl proxying disabled")
	}

	settings = p.appSettingsSnapshot()
	settings.Proxy.SSLProxying.Enabled = true
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}
	if !p.mitmEnabled() {
		t.Fatal("expected mitm enabled when both listener mitm and ssl proxying are enabled")
	}
}

func TestHotReloadProxyPort(t *testing.T) {
	proxyPortA := freeTCPPort(t)
	proxyPortB := freeTCPPort(t)
	if proxyPortA == proxyPortB {
		proxyPortB = freeTCPPort(t)
	}

	p := newStartedTestProxy(t, proxyPortA)
	defer func() { _ = p.Stop() }()

	if err := waitTCPPort(proxyPortA, 2*time.Second); err != nil {
		t.Fatalf("wait old http port failed: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	if err := assertProxyHTTPResponse(proxyPortA, upstream.URL, "ok"); err != nil {
		t.Fatalf("old http proxy check failed: %v", err)
	}

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.Port = proxyPortB

	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	if err := waitTCPPort(proxyPortB, 2*time.Second); err != nil {
		t.Fatalf("wait new http port failed: %v", err)
	}

	if err := assertProxyHTTPResponse(proxyPortB, upstream.URL, "ok"); err != nil {
		t.Fatalf("new http proxy check failed: %v", err)
	}

	if err := assertPortClosed(proxyPortA, 2*time.Second); err != nil {
		t.Fatalf("old http port should be closed: %v", err)
	}
}

func TestHotReloadProxyPort_ConflictError(t *testing.T) {
	proxyPortA := freeTCPPort(t)
	proxyPortB := freeTCPPort(t)

	p := newStartedTestProxy(t, proxyPortA)
	defer func() { _ = p.Stop() }()

	if err := waitTCPPort(proxyPortA, 2*time.Second); err != nil {
		t.Fatalf("wait old http port failed: %v", err)
	}

	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", proxyPortB))
	if err != nil {
		t.Fatalf("listen blocker failed: %v", err)
	}
	defer blocker.Close()

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.Port = proxyPortB

	if err := p.ApplySettings(settings); err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("still-old"))
	}))
	defer upstream.Close()

	if err := assertProxyHTTPResponse(proxyPortA, upstream.URL, "still-old"); err != nil {
		t.Fatalf("old http proxy should still work: %v", err)
	}

	if err := assertPortClosed(proxyPortB, 200*time.Millisecond); err == nil {
		t.Fatal("new conflicted port unexpectedly accepted proxy connections")
	}
}

func TestHotReloadProxyPort_GracefulDrain(t *testing.T) {
	proxyPortA := freeTCPPort(t)
	proxyPortB := freeTCPPort(t)

	p := newStartedTestProxy(t, proxyPortA)
	defer func() { _ = p.Stop() }()

	reqStarted := make(chan struct{}, 1)
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reqStarted <- struct{}{}
		<-release
		_, _ = w.Write([]byte("drained"))
	}))
	defer upstream.Close()

	result := make(chan error, 1)
	go func() {
		result <- assertProxyHTTPResponseWithTimeout(proxyPortA, upstream.URL, "drained", 6*time.Second)
	}()

	select {
	case <-reqStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting slow request start")
	}

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.Port = proxyPortB

	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	close(release)

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("in-flight request failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting in-flight request completion")
	}

	if err := assertProxyHTTPResponseWithTimeout(proxyPortB, upstream.URL, "drained", 6*time.Second); err != nil {
		t.Fatalf("new port request failed: %v", err)
	}
}

func TestHotReloadSOCKS5Toggle(t *testing.T) {
	proxyPort := freeTCPPort(t)
	p := newStartedTestProxy(t, proxyPort)
	defer func() { _ = p.Stop() }()

	if err := waitTCPPort(proxyPort, 2*time.Second); err != nil {
		t.Fatalf("wait proxy port failed: %v", err)
	}

	if err := assertSOCKS5Handshake(proxyPort); err != nil {
		t.Fatalf("initial socks5 handshake failed: %v", err)
	}

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.SOCKS5Enabled = false
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	if err := assertSOCKS5Handshake(proxyPort); err == nil {
		t.Fatal("expected socks5 handshake failure when socks5 is disabled")
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("http-still-works"))
	}))
	defer upstream.Close()

	if err := assertProxyHTTPResponse(proxyPort, upstream.URL, "http-still-works"); err != nil {
		t.Fatalf("http proxy should still work when socks5 is disabled: %v", err)
	}

	settings = p.appSettingsSnapshot()
	settings.Proxy.Listener.SOCKS5Enabled = true
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	if err := assertSOCKS5Handshake(proxyPort); err != nil {
		t.Fatalf("socks5 handshake should recover after enabling: %v", err)
	}
}

func TestUnifiedPortValidation_InvalidPortError(t *testing.T) {
	p := newTestProxyForPort(t, freeTCPPort(t))
	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.HTTPEnabled = true
	settings.Proxy.Listener.HTTPSEnabled = true
	settings.Proxy.Listener.SOCKS5Enabled = true
	settings.Proxy.Listener.Port = 70000

	if err := p.ApplySettings(settings); err == nil {
		t.Fatal("expected validation error for invalid unified port")
	}
}

func TestProxyPortValidation_Valid(t *testing.T) {
	proxyPort := freeTCPPort(t)
	p := newStartedTestProxy(t, proxyPort)
	defer func() { _ = p.Stop() }()

	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.HTTPEnabled = true
	settings.Proxy.Listener.HTTPSEnabled = true
	settings.Proxy.Listener.Port = proxyPort

	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("expected success for valid proxy port, got: %v", err)
	}
}

func newTestProxyForPort(t *testing.T, port int) *Proxy {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := storage.NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	p := New()
	settings := p.appSettingsSnapshot()
	settings.Proxy.Listener.Port = port
	settings.Proxy.Listener.HTTPEnabled = true
	settings.Proxy.Listener.HTTPSEnabled = true
	settings.Proxy.Listener.MITMEnabled = false
	settings.Proxy.Listener.SOCKS5Enabled = true
	settings.Cert.CACertFile = filepath.Join(tmpDir, "ca.crt")
	settings.Cert.CAKeyFile = filepath.Join(tmpDir, "ca.key")
	settings.Cert.Organization = "PacketMind Test"
	if err := p.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}
	return p
}

func newStartedTestProxy(t *testing.T, port int) *Proxy {
	t.Helper()
	p := newTestProxyForPort(t, port)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	return p
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve free port: %v", err)
	}
	defer l.Close()
	_, portStr, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}
	port, err := net.LookupPort("tcp", portStr)
	if err != nil {
		t.Fatalf("lookup port failed: %v", err)
	}
	return port
}

func waitTCPPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("port %d not reachable within %s", port, timeout)
}

func assertPortClosed(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return nil
		}
		_ = conn.Close()
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("port %d still reachable after %s", port, timeout)
}

func assertProxyHTTPResponse(proxyPort int, targetURL, expectedBody string) error {
	return assertProxyHTTPResponseWithTimeout(proxyPort, targetURL, expectedBody, 2*time.Second)
}

func assertProxyHTTPResponseWithTimeout(proxyPort int, targetURL, expectedBody string, timeout time.Duration) error {
	proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}
	resp, err := client.Get(targetURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != expectedBody {
		return fmt.Errorf("unexpected body: %q", string(body))
	}
	return nil
}

func assertSOCKS5Handshake(port int) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{socks5Version, 1, socks5NoAuth}); err != nil {
		return err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if !bytes.Equal(resp, []byte{socks5Version, socks5NoAuth}) {
		return fmt.Errorf("unexpected socks5 handshake response: %v", resp)
	}
	return nil
}
