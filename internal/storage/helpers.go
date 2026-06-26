package storage

import (
	"strings"
	"time"
)

func cloneRequest(req *Request) *Request {
	if req == nil {
		return nil
	}
	cp := *req
	cp.Headers = cloneHeaders(req.Headers)
	cp.Cookies = cloneCookies(req.Cookies)
	cp.RespHeaders = cloneHeaders(req.RespHeaders)
	cp.TLSServerCertificates = cloneTLSCertificates(req.TLSServerCertificates)
	cp.TLSServerExtensions = cloneTLSExtensions(req.TLSServerExtensions)
	if req.Body != nil {
		cp.Body = append([]byte(nil), req.Body...)
	}
	if req.RespBody != nil {
		cp.RespBody = append([]byte(nil), req.RespBody...)
	}
	if req.TLSClientHelloRaw != nil {
		cp.TLSClientHelloRaw = append([]byte(nil), req.TLSClientHelloRaw...)
	}
	if req.WebSocketFrames != nil {
		cp.WebSocketFrames = append([]WebSocketFrame(nil), req.WebSocketFrames...)
		for i := range cp.WebSocketFrames {
			if req.WebSocketFrames[i].Payload != nil {
				cp.WebSocketFrames[i].Payload = append([]byte(nil), req.WebSocketFrames[i].Payload...)
			}
		}
	}
	return &cp
}

func cloneChatMessage(msg *ChatMessage) *ChatMessage {
	if msg == nil {
		return nil
	}
	cp := *msg
	return &cp
}

func cloneTLSCertificates(items []TLSCertificate) []TLSCertificate {
	if items == nil {
		return nil
	}
	cloned := make([]TLSCertificate, len(items))
	for i := range items {
		cloned[i] = items[i]
		cloned[i].DNSNames = append([]string(nil), items[i].DNSNames...)
		cloned[i].EmailAddresses = append([]string(nil), items[i].EmailAddresses...)
		cloned[i].IPAddresses = append([]string(nil), items[i].IPAddresses...)
		cloned[i].OCSPServers = append([]string(nil), items[i].OCSPServers...)
		cloned[i].IssuingCertificateURL = append([]string(nil), items[i].IssuingCertificateURL...)
		cloned[i].Extensions = cloneTLSExtensions(items[i].Extensions)
	}
	return cloned
}

func cloneTLSExtensions(items []TLSExtension) []TLSExtension {
	if items == nil {
		return nil
	}
	return append([]TLSExtension(nil), items...)
}

func cloneHeaders(headers Headers) Headers {
	if headers == nil {
		return nil
	}
	cloned := make(Headers, len(headers))
	for key, values := range headers {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func cloneCookies(cookies Cookies) Cookies {
	if cookies == nil {
		return nil
	}
	cloned := make(Cookies, len(cookies))
	for key, value := range cookies {
		cloned[key] = value
	}
	return cloned
}

func normalizeSortField(sortBy string) string {
	field := strings.ToLower(strings.TrimSpace(sortBy))
	field = strings.TrimSuffix(field, " asc")
	field = strings.TrimSuffix(field, " desc")
	field = strings.Trim(field, "`")
	return field
}

func generateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
