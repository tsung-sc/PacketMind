package proxy

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/packetmind/packetmind/internal/storage"
)

func readWebSocketRawFrame(r io.Reader) (*websocketRawFrame, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	fin := header[0]&0x80 != 0
	opcode := int(header[0] & 0x0F)
	masked := header[1]&0x80 != 0
	payloadLen := int64(header[1] & 0x7F)
	raw := append([]byte(nil), header...)

	switch payloadLen {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, err
		}
		raw = append(raw, ext...)
		payloadLen = int64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err := io.ReadFull(r, ext); err != nil {
			return nil, err
		}
		raw = append(raw, ext...)
		payloadLen = int64(binary.BigEndian.Uint64(ext))
	}

	maskKey := []byte(nil)
	if masked {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(r, maskKey); err != nil {
			return nil, err
		}
		raw = append(raw, maskKey...)
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
		raw = append(raw, payload...)
	}

	decoded := append([]byte(nil), payload...)
	if masked {
		for i := range decoded {
			decoded[i] ^= maskKey[i%4]
		}
	}

	return &websocketRawFrame{
		raw:     raw,
		payload: decoded,
		opcode:  opcode,
		fin:     fin,
		masked:  masked,
	}, nil
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (p *Proxy) captureWebSocketTraffic(clientConn net.Conn, targetConn net.Conn) websocketCaptureResult {
	frames := make([]storage.WebSocketFrame, 0)
	responseStart := time.Now()

	var mu sync.Mutex
	var wg sync.WaitGroup
	done := make(chan struct{}, 2)
	copyDir := func(dst net.Conn, src net.Conn, direction string) {
		defer wg.Done()
		for {
			frame, err := readWebSocketRawFrame(src)
			if frame != nil {
				mu.Lock()
				frames = append(frames, storage.WebSocketFrame{
					ID:          uuid.New().String(),
					Direction:   direction,
					Opcode:      frame.opcode,
					FrameType:   websocketFrameType(frame.opcode),
					Payload:     p.truncateWebSocketPayload(frame.payload),
					PayloadSize: int64(len(frame.payload)),
					CreatedAt:   time.Now(),
					Fin:         frame.fin,
					Masked:      frame.masked,
				})
				mu.Unlock()
				if writeErr := writeAll(dst, frame.raw); writeErr != nil {
					done <- struct{}{}
					return
				}
			}
			if err != nil {
				done <- struct{}{}
				return
			}
		}
	}

	wg.Add(2)
	go copyDir(targetConn, clientConn, "sent")
	go copyDir(clientConn, targetConn, "received")
	<-done
	_ = clientConn.SetDeadline(time.Now())
	_ = targetConn.SetDeadline(time.Now())
	wg.Wait()
	return websocketCaptureResult{
		frames:        frames,
		statusCode:    http.StatusSwitchingProtocols,
		responseStart: responseStart,
		responseEnd:   time.Now(),
	}
}

func (p *Proxy) writeWebSocketHandshakeResponse(clientConn net.Conn, statusCode int, statusText string, headers http.Header) error {
	var buf bytes.Buffer
	if statusText == "" {
		statusText = fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))
	}
	fmt.Fprintf(&buf, "HTTP/1.1 %s\r\n", statusText)
	clone := cloneHTTPHeader(headers)
	if clone == nil {
		clone = make(http.Header)
	}
	if clone.Get("Connection") == "" {
		clone.Set("Connection", "Upgrade")
	}
	if clone.Get("Upgrade") == "" {
		clone.Set("Upgrade", "websocket")
	}
	if err := clone.Write(&buf); err != nil {
		return err
	}
	buf.WriteString("\r\n")
	_, err := clientConn.Write(buf.Bytes())
	return err
}

func (p *Proxy) finalizeWebSocketRecord(record *storage.Request, result websocketCaptureResult, reqStartTime time.Time) {
	if record == nil {
		return
	}
	record.IsWebSocket = true
	record.WebSocketFrames = result.frames
	if result.statusCode != 0 {
		record.StatusCode = result.statusCode
	} else {
		record.StatusCode = http.StatusSwitchingProtocols
	}
	if result.statusReason != "" {
		record.StatusReason = result.statusReason
	} else {
		record.StatusReason = "101 Switching Protocols"
	}
	if len(result.respHeaders) > 0 {
		record.RespHeaders = result.respHeaders
	}
	record.ResponseStartTime = result.responseStart
	record.ResponseEndTime = result.responseEnd
	record.ResponseDuration = safeDurationMillis(result.responseStart, result.responseEnd)
	record.LatencyDuration = safeDurationMillis(record.RequestEndTime, result.responseStart)
	record.Duration = safeDurationMillis(reqStartTime, result.responseEnd)
	record.RespBodySize = result.respBodySize
	if result.serverAddr != "" {
		record.ServerAddr = result.serverAddr
	}
	if result.errorMessage != "" {
		record.Error = result.errorMessage
	}
}

func (p *Proxy) handleHTTPWebSocket(w http.ResponseWriter, r *http.Request, startTime time.Time) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		p.recordMinimalError(r.Method, websocketSchemeFromRequest(r, requestScheme(r)), r.Host, requestPort(r), r.URL.String(), http.StatusInternalServerError, "hijacking not supported", r.RemoteAddr, r.Host, startTime)
		return
	}
	clientConn, clientRW, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		p.recordMinimalError(r.Method, websocketSchemeFromRequest(r, requestScheme(r)), r.Host, requestPort(r), r.URL.String(), http.StatusInternalServerError, fmt.Sprintf("hijack failed: %v", err), r.RemoteAddr, r.Host, startTime)
		return
	}
	defer clientConn.Close()
	targetHost := r.Host
	if !strings.Contains(targetHost, ":") {
		targetHost = fmt.Sprintf("%s:%d", targetHost, requestPort(r))
	}
	reqStartTime := time.Now()
	record := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        reqStartTime,
		Method:           r.Method,
		Scheme:           websocketSchemeFromRequest(r, requestScheme(r)),
		Host:             r.Host,
		Port:             requestPort(r),
		Path:             r.URL.Path,
		URL:              r.URL.String(),
		QueryString:      r.URL.RawQuery,
		HTTPVersion:      r.Proto,
		Headers:          toStorageHeaders(r.Header),
		Cookies:          extractRequestCookies(r),
		IsWebSocket:      true,
		ContentType:      r.Header.Get("Content-Type"),
		RemoteAddr:       r.RemoteAddr,
		ClientAddr:       r.RemoteAddr,
		ServerAddr:       targetHost,
		KeepAlive:        true,
		RequestStartTime: reqStartTime,
	}
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		record.Body = body
		record.BodySize = int64(len(body))
	}
	record.RequestEndTime = time.Now()
	record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
	p.saveRequestStart(record)
	upstream, err := p.dialWebSocketUpstream(r, record.Scheme, targetHost)
	if err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseStartTime = time.Now()
		record.ResponseEndTime = record.ResponseStartTime
		p.saveRequestStart(record)
		return
	}
	defer upstream.conn.Close()
	record.RespHeaders = toStorageHeaders(upstream.headers)
	record.ResponseStartTime = time.Now()
	record.ServerAddr = upstream.serverAddr
	if err := p.writeWebSocketHandshakeResponse(clientConn, upstream.statusCode, upstream.statusText, upstream.headers); err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseEndTime = time.Now()
		p.saveRequestStart(record)
		return
	}
	clientStream := net.Conn(clientConn)
	if clientRW != nil {
		clientStream = &bufferedConn{Conn: clientConn, reader: clientRW}
	}
	targetStream := net.Conn(upstream.conn)
	if upstream.reader != nil {
		targetStream = &bufferedConn{Conn: upstream.conn, reader: upstream.reader}
	}
	result := p.captureWebSocketTraffic(clientStream, targetStream)
	result.statusCode = upstream.statusCode
	result.statusReason = upstream.statusText
	result.respHeaders = toStorageHeaders(upstream.headers)
	result.serverAddr = upstream.serverAddr
	p.finalizeWebSocketRecord(record, result, reqStartTime)
	p.saveRequestComplete(record)
}

func (p *Proxy) handleMITMWebSocket(clientConn net.Conn, reader *bufio.Reader, hostname, targetHost string, req *http.Request, reqStartTime time.Time) {
	record := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        reqStartTime,
		Method:           req.Method,
		Scheme:           websocketSchemeFromRequest(req, "https"),
		Host:             hostname,
		Port:             requestPort(req),
		Path:             req.URL.Path,
		URL:              req.URL.String(),
		QueryString:      req.URL.RawQuery,
		HTTPVersion:      req.Proto,
		Headers:          toStorageHeaders(req.Header),
		Cookies:          extractRequestCookies(req),
		IsWebSocket:      true,
		ContentType:      req.Header.Get("Content-Type"),
		RemoteAddr:       clientConn.RemoteAddr().String(),
		ClientAddr:       clientConn.RemoteAddr().String(),
		ServerAddr:       targetHost,
		KeepAlive:        true,
		RequestStartTime: reqStartTime,
	}
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		record.Body = body
		record.BodySize = int64(len(body))
	}
	record.RequestEndTime = time.Now()
	record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
	p.saveRequestStart(record)
	upstream, err := p.dialWebSocketUpstream(req, record.Scheme, targetHost)
	if err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseStartTime = time.Now()
		record.ResponseEndTime = record.ResponseStartTime
		p.saveRequestStart(record)
		return
	}
	defer upstream.conn.Close()
	record.RespHeaders = toStorageHeaders(upstream.headers)
	record.ResponseStartTime = time.Now()
	record.ServerAddr = upstream.serverAddr
	if err := p.writeWebSocketHandshakeResponse(clientConn, upstream.statusCode, upstream.statusText, upstream.headers); err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseEndTime = time.Now()
		p.saveRequestStart(record)
		return
	}
	targetStream := net.Conn(upstream.conn)
	if upstream.reader != nil {
		targetStream = &bufferedConn{Conn: upstream.conn, reader: upstream.reader}
	}
	result := p.captureWebSocketTraffic(&bufferedConn{Conn: clientConn, reader: reader}, targetStream)
	result.statusCode = upstream.statusCode
	result.statusReason = upstream.statusText
	result.respHeaders = toStorageHeaders(upstream.headers)
	result.serverAddr = upstream.serverAddr
	p.finalizeWebSocketRecord(record, result, reqStartTime)
	p.saveRequestComplete(record)
}

func (p *Proxy) handleSOCKS5WebSocket(client, target net.Conn, targetAddr string, startTime time.Time, clientReader *bufio.Reader, req *http.Request, reqStartTime time.Time, targetReader *bufio.Reader) {
	host, port := parseHostPort(targetAddr)
	scheme := websocketSchemeFromRequest(req, "http")
	if port == 443 {
		scheme = "wss"
	}
	fullURL := fmt.Sprintf("%s://%s%s", scheme, host, req.RequestURI)
	record := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        reqStartTime,
		Method:           req.Method,
		Scheme:           scheme,
		Host:             host,
		Port:             port,
		Path:             req.URL.Path,
		URL:              fullURL,
		QueryString:      req.URL.RawQuery,
		HTTPVersion:      req.Proto,
		Headers:          toStorageHeaders(req.Header),
		Cookies:          extractRequestCookies(req),
		IsWebSocket:      true,
		ContentType:      req.Header.Get("Content-Type"),
		RemoteAddr:       client.RemoteAddr().String(),
		ClientAddr:       client.RemoteAddr().String(),
		ServerAddr:       target.RemoteAddr().String(),
		KeepAlive:        true,
		RequestStartTime: reqStartTime,
	}
	record.RequestEndTime = time.Now()
	record.RequestDuration = safeDurationMillis(record.RequestStartTime, record.RequestEndTime)
	p.saveRequestStart(record)
	if err := req.Write(target); err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseStartTime = time.Now()
		record.ResponseEndTime = record.ResponseStartTime
		p.saveRequestStart(record)
		return
	}
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseStartTime = time.Now()
		record.ResponseEndTime = record.ResponseStartTime
		p.saveRequestStart(record)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols && resp.Body != nil {
		_ = resp.Body.Close()
	}
	record.RespHeaders = toStorageHeaders(resp.Header)
	record.ResponseStartTime = time.Now()
	if err := p.writeWebSocketHandshakeResponse(client, resp.StatusCode, resp.Status, resp.Header); err != nil {
		record.StatusCode = http.StatusBadGateway
		record.StatusReason = http.StatusText(http.StatusBadGateway)
		record.Duration = time.Since(reqStartTime).Milliseconds()
		record.Error = err.Error()
		record.ResponseEndTime = time.Now()
		p.saveRequestStart(record)
		return
	}
	result := p.captureWebSocketTraffic(&bufferedConn{Conn: client, reader: clientReader}, &bufferedConn{Conn: target, reader: targetReader})
	result.statusCode = resp.StatusCode
	result.statusReason = resp.Status
	result.respHeaders = toStorageHeaders(resp.Header)
	result.serverAddr = target.RemoteAddr().String()
	p.finalizeWebSocketRecord(record, result, reqStartTime)
	p.saveRequestComplete(record)
	_ = startTime
}
