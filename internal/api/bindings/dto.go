package bindings

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
	"github.com/packetmind/packetmind/internal/storage"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

type JSONTime string

func toJSONTime(t time.Time) JSONTime {
	if t.IsZero() {
		return ""
	}
	return JSONTime(t.Format(time.RFC3339Nano))
}

type SessionDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CreatedAt   JSONTime `json:"created_at"`
	UpdatedAt   JSONTime `json:"updated_at"`
	IsActive    bool     `json:"is_active"`
	Description string   `json:"description"`
}

func toSessionDTO(session *storage.Session) *SessionDTO {
	if session == nil {
		return nil
	}

	return &SessionDTO{
		ID:          session.ID,
		Name:        session.Name,
		CreatedAt:   toJSONTime(session.CreatedAt),
		UpdatedAt:   toJSONTime(session.UpdatedAt),
		IsActive:    session.IsActive,
		Description: session.Description,
	}
}

func toSessionDTOs(items []*storage.Session) []*SessionDTO {
	result := make([]*SessionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toSessionDTO(item))
	}
	return result
}

type TLSExtensionDTO struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func toTLSExtensionDTOs(items []storage.TLSExtension) []TLSExtensionDTO {
	result := make([]TLSExtensionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, TLSExtensionDTO{
			ID:    item.ID,
			Name:  item.Name,
			Value: item.Value,
		})
	}
	return result
}

type TLSCertificateDTO struct {
	SubjectCommonName     string            `json:"subject_common_name"`
	Subject               string            `json:"subject"`
	IssuerCommonName      string            `json:"issuer_common_name"`
	Issuer                string            `json:"issuer"`
	SerialNumber          string            `json:"serial_number"`
	DNSNames              []string          `json:"dns_names"`
	EmailAddresses        []string          `json:"email_addresses"`
	IPAddresses           []string          `json:"ip_addresses"`
	Version               int               `json:"version"`
	IsCA                  bool              `json:"is_ca"`
	SignatureAlgorithm    string            `json:"signature_algorithm"`
	PublicKeyAlgorithm    string            `json:"public_key_algorithm"`
	NotBefore             JSONTime          `json:"not_before"`
	NotAfter              JSONTime          `json:"not_after"`
	OCSPServers           []string          `json:"ocsp_servers"`
	IssuingCertificateURL []string          `json:"issuing_certificate_url"`
	Extensions            []TLSExtensionDTO `json:"extensions"`
}

type WebSocketFrameDTO struct {
	ID          string   `json:"id"`
	Direction   string   `json:"direction"`
	Opcode      int      `json:"opcode"`
	FrameType   string   `json:"frame_type"`
	Payload     string   `json:"payload"`
	PayloadSize int64    `json:"payload_size"`
	CreatedAt   JSONTime `json:"created_at"`
	Fin         bool     `json:"fin"`
	Masked      bool     `json:"masked"`
}

func toWebSocketFrameDTOs(items []storage.WebSocketFrame) []WebSocketFrameDTO {
	result := make([]WebSocketFrameDTO, 0, len(items))
	for _, item := range items {
		result = append(result, WebSocketFrameDTO{
			ID:          item.ID,
			Direction:   item.Direction,
			Opcode:      item.Opcode,
			FrameType:   item.FrameType,
			Payload:     bytesToTransportString(item.Payload, "", nil),
			PayloadSize: item.PayloadSize,
			CreatedAt:   toJSONTime(item.CreatedAt),
			Fin:         item.Fin,
			Masked:      item.Masked,
		})
	}
	return result
}

func toTLSCertificateDTOs(items []storage.TLSCertificate) []TLSCertificateDTO {
	result := make([]TLSCertificateDTO, 0, len(items))
	for _, item := range items {
		result = append(result, TLSCertificateDTO{
			SubjectCommonName:     item.SubjectCommonName,
			Subject:               item.Subject,
			IssuerCommonName:      item.IssuerCommonName,
			Issuer:                item.Issuer,
			SerialNumber:          item.SerialNumber,
			DNSNames:              item.DNSNames,
			EmailAddresses:        item.EmailAddresses,
			IPAddresses:           item.IPAddresses,
			Version:               item.Version,
			IsCA:                  item.IsCA,
			SignatureAlgorithm:    item.SignatureAlgorithm,
			PublicKeyAlgorithm:    item.PublicKeyAlgorithm,
			NotBefore:             toJSONTime(item.NotBefore),
			NotAfter:              toJSONTime(item.NotAfter),
			OCSPServers:           item.OCSPServers,
			IssuingCertificateURL: item.IssuingCertificateURL,
			Extensions:            toTLSExtensionDTOs(item.Extensions),
		})
	}
	return result
}

type RequestDTO struct {
	ID                    string              `json:"id"`
	SessionID             string              `json:"session_id"`
	CreatedAt             JSONTime            `json:"created_at"`
	UpdatedAt             JSONTime            `json:"updated_at"`
	Method                string              `json:"method"`
	Scheme                string              `json:"scheme"`
	Host                  string              `json:"host"`
	Port                  int                 `json:"port"`
	Path                  string              `json:"path"`
	URL                   string              `json:"url"`
	QueryString           string              `json:"query_string"`
	HTTPVersion           string              `json:"http_version"`
	Headers               map[string][]string `json:"headers"`
	Cookies               map[string]string   `json:"cookies"`
	IsWebSocket           bool                `json:"is_websocket"`
	ContentType           string              `json:"content_type"`
	Body                  string              `json:"body"`
	BodySize              int64               `json:"body_size"`
	StatusCode            int                 `json:"status_code"`
	StatusReason          string              `json:"status_reason"`
	RespHeaders           map[string][]string `json:"resp_headers"`
	RespContentType       string              `json:"resp_content_type"`
	RespBody              string              `json:"resp_body"`
	RespBodySize          int64               `json:"resp_body_size"`
	Duration              int64               `json:"duration"`
	RemoteAddr            string              `json:"remote_addr"`
	ClientAddr            string              `json:"client_addr"`
	ServerAddr            string              `json:"server_addr"`
	Notes                 string              `json:"notes"`
	Error                 string              `json:"error"`
	RequestStartTime      JSONTime            `json:"request_start_time"`
	RequestEndTime        JSONTime            `json:"request_end_time"`
	ResponseStartTime     JSONTime            `json:"response_start_time"`
	ResponseEndTime       JSONTime            `json:"response_end_time"`
	DNSDuration           int64               `json:"dns_duration"`
	ConnectDuration       int64               `json:"connect_duration"`
	TLSHandshakeDuration  int64               `json:"tls_handshake_duration"`
	RequestDuration       int64               `json:"request_duration"`
	ResponseDuration      int64               `json:"response_duration"`
	LatencyDuration       int64               `json:"latency_duration"`
	KeepAlive             bool                `json:"keep_alive"`
	TLSVersion            string              `json:"tls_version"`
	TLSCipherSuite        string              `json:"tls_cipher_suite"`
	TLSServerName         string              `json:"tls_server_name"`
	TLSDidResume          bool                `json:"tls_did_resume"`
	TLSALPN               string              `json:"tls_alpn"`
	TLSCurveID            string              `json:"tls_curve_id"`
	TLSOCSPStapled        bool                `json:"tls_ocsp_stapled"`
	TLSSCTCount           int                 `json:"tls_sct_count"`
	TLSServerCertificates []TLSCertificateDTO `json:"tls_server_certificates"`
	TLSServerExtensions   []TLSExtensionDTO   `json:"tls_server_extensions"`
	WebSocketFrames       []WebSocketFrameDTO `json:"websocket_frames"`
}

type RequestEventDTO = RequestDTO

// bytesToTransportString converts body bytes to a transport-safe string.
// It handles content-encoding aware decompression (gzip/deflate/brotli),
// charset-aware transcoding to UTF-8 for text content-types, and falls
// back to base64 only for truly unreadable binary content.
func bytesToTransportString(data []byte, contentType string, headers map[string][]string) string {
	if len(data) == 0 {
		return ""
	}

	// Get content-encoding from headers
	contentEncoding := getHeader(headers, "Content-Encoding")

	// Try to decompress if content-encoding is present.
	// ok=false means unsupported encoding or decompression failure.
	decoded, ok, _ := decodeContent(data, contentEncoding)
	if !ok {
		// Decompression failed or unsupported encoding, fall back to base64
		return base64.StdEncoding.EncodeToString(data)
	}

	// Check if content-type indicates text
	if isTextLikeContentType(contentType) {
		// Fast path: already valid UTF-8
		if isValidUTF8OrASCII(decoded) {
			return string(decoded)
		}
		// Attempt charset-aware transcoding to UTF-8
		charset := extractCharset(contentType)
		if charset != "" {
			if transcoded, err := transcodeToUTF8(decoded, charset); err == nil {
				return string(transcoded)
			}
		}
		// Text content but could not decode charset; fall back to base64
		// using the decoded (decompressed) bytes, not the original compressed bytes.
		return base64.StdEncoding.EncodeToString(decoded)
	}

	// Non-text content-type: use byte heuristics
	if isLikelyText(decoded) {
		// Fast path: valid UTF-8
		if isValidUTF8OrASCII(decoded) {
			return string(decoded)
		}
		// Heuristic text but non-UTF8: try charset extraction from content-type
		// (some binary-labeled responses still carry charset info)
		charset := extractCharset(contentType)
		if charset != "" {
			if transcoded, err := transcodeToUTF8(decoded, charset); err == nil {
				return string(transcoded)
			}
		}
	}

	// Binary or unreadable content — use decoded bytes for base64
	return base64.StdEncoding.EncodeToString(decoded)
}

// getHeader retrieves a header value case-insensitively
func getHeader(headers map[string][]string, key string) string {
	if headers == nil {
		return ""
	}
	// Try exact match first
	if values, ok := headers[key]; ok && len(values) > 0 {
		return values[0]
	}
	// Try case-insensitive lookup
	lowerKey := strings.ToLower(key)
	for k, values := range headers {
		if strings.ToLower(k) == lowerKey && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// decodeContent attempts to decompress content based on Content-Encoding header.
// Supports gzip, deflate, and brotli. Returns:
// - decoded bytes
// - ok: whether the operation succeeded (false for unsupported encodings)
// - wasDecoded: whether the content was actually decompressed (true only for gzip/deflate/brotli)
func decodeContent(data []byte, contentEncoding string) (decoded []byte, ok bool, wasDecoded bool) {
	if contentEncoding == "" {
		return data, true, false
	}

	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))

	// Handle multiple encodings (e.g., "gzip, deflate" - though uncommon)
	// For simplicity, we handle single encoding cases
	switch {
	case encoding == "gzip" || encoding == "x-gzip":
		decoded, ok := decodeGzip(data)
		return decoded, ok, true
	case encoding == "deflate":
		decoded, ok := decodeDeflate(data)
		return decoded, ok, true
	case encoding == "br" || encoding == "brotli":
		decoded, ok := decodeBrotli(data)
		return decoded, ok, true
	case encoding == "identity" || encoding == "":
		// No transformation needed - data is already in final form
		return data, true, false
	default:
		// Unknown encoding - cannot decode safely
		return nil, false, false
	}
}

// decodeGzip decompresses gzip-encoded data
func decodeGzip(data []byte) ([]byte, bool) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer r.Close()

	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// decodeDeflate decompresses deflate-encoded data
func decodeDeflate(data []byte) ([]byte, bool) {
	// Many servers label zlib-wrapped payloads as deflate.
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer zr.Close()
		decoded, readErr := io.ReadAll(zr)
		if readErr == nil {
			return decoded, true
		}
	}

	// Fall back to raw deflate stream.
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()

	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// decodeBrotli decompresses brotli-encoded data
func decodeBrotli(data []byte) ([]byte, bool) {
	r := brotli.NewReader(bytes.NewReader(data))
	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// extractCharset parses the charset parameter from a Content-Type header value.
// Returns an empty string if no charset is found.
func extractCharset(contentType string) string {
	ct := strings.ToLower(contentType)
	// Look for "charset=" in the content-type string
	idx := strings.Index(ct, "charset=")
	if idx < 0 {
		return ""
	}
	// Extract the value after "charset="
	charset := ct[idx+8:]
	// Trim leading/trailing whitespace and semicolons
	charset = strings.TrimSpace(charset)
	if semi := strings.Index(charset, ";"); semi >= 0 {
		charset = charset[:semi]
	}
	// Strip quotes if present
	charset = strings.Trim(charset, `"'`)
	if charset == "" || charset == "utf-8" || charset == "utf8" {
		return ""
	}
	return charset
}

// transcodeToUTF8 converts bytes encoded in the given charset to UTF-8.
// Uses golang.org/x/text/encoding/htmlindex for standard IANA charset lookup.
func transcodeToUTF8(data []byte, charset string) ([]byte, error) {
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return nil, fmt.Errorf("unsupported charset %q: %w", charset, err)
	}
	// If the encoding is UTF-8 (Nop), no conversion needed
	transcoded, _, err := transform.Bytes(enc.NewDecoder(), data)
	if err != nil {
		return nil, fmt.Errorf("charset transcode from %q to UTF-8 failed: %w", charset, err)
	}
	return transcoded, nil
}

// isTextLikeContentType determines if a content-type indicates textual content
func isTextLikeContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	// Handle empty content-type as potentially text
	if ct == "" {
		return false
	}
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "ecmascript") ||
		strings.Contains(ct, "x-www-form-urlencoded") ||
		strings.Contains(ct, "graphql") ||
		strings.Contains(ct, "svg")
}

// isLikelyText uses byte heuristics to determine if data might be text
func isLikelyText(data []byte) bool {
	// Check for null bytes or problematic control characters
	for _, b := range data {
		if b == 0 {
			return false
		}
		// Allow common whitespace: tab (0x09), LF (0x0A), CR (0x0D)
		if b < 0x09 {
			return false
		}
		if b > 0x0D && b < 0x20 {
			return false
		}
	}
	return true
}

// isValidUTF8OrASCII validates that the data is valid UTF-8
func isValidUTF8OrASCII(data []byte) bool {
	return utf8.Valid(data)
}

func toRequestDTO(req *storage.Request, sessionID string) *RequestDTO {
	if req == nil {
		return nil
	}

	sid := sessionID
	if sid == "" {
		sid = req.SessionID
	}

	return &RequestDTO{
		ID:                    req.ID,
		SessionID:             sid,
		CreatedAt:             toJSONTime(req.CreatedAt),
		UpdatedAt:             toJSONTime(req.UpdatedAt),
		Method:                req.Method,
		Scheme:                req.Scheme,
		Host:                  req.Host,
		Port:                  req.Port,
		Path:                  req.Path,
		URL:                   req.URL,
		QueryString:           req.QueryString,
		HTTPVersion:           req.HTTPVersion,
		Headers:               req.Headers,
		Cookies:               req.Cookies,
		IsWebSocket:           req.IsWebSocket,
		ContentType:           req.ContentType,
		Body:                  bytesToTransportString(req.Body, req.ContentType, req.Headers),
		BodySize:              req.BodySize,
		StatusCode:            req.StatusCode,
		StatusReason:          req.StatusReason,
		RespHeaders:           req.RespHeaders,
		RespContentType:       req.RespContentType,
		RespBody:              bytesToTransportString(req.RespBody, req.RespContentType, req.RespHeaders),
		RespBodySize:          req.RespBodySize,
		Duration:              req.Duration,
		RemoteAddr:            req.RemoteAddr,
		ClientAddr:            req.ClientAddr,
		ServerAddr:            req.ServerAddr,
		Notes:                 req.Notes,
		Error:                 req.Error,
		RequestStartTime:      toJSONTime(req.RequestStartTime),
		RequestEndTime:        toJSONTime(req.RequestEndTime),
		ResponseStartTime:     toJSONTime(req.ResponseStartTime),
		ResponseEndTime:       toJSONTime(req.ResponseEndTime),
		DNSDuration:           req.DNSDuration,
		ConnectDuration:       req.ConnectDuration,
		TLSHandshakeDuration:  req.TLSHandshakeDuration,
		RequestDuration:       req.RequestDuration,
		ResponseDuration:      req.ResponseDuration,
		LatencyDuration:       req.LatencyDuration,
		KeepAlive:             req.KeepAlive,
		TLSVersion:            req.TLSVersion,
		TLSCipherSuite:        req.TLSCipherSuite,
		TLSServerName:         req.TLSServerName,
		TLSDidResume:          req.TLSDidResume,
		TLSALPN:               req.TLSALPN,
		TLSCurveID:            req.TLSCurveID,
		TLSOCSPStapled:        req.TLSOCSPStapled,
		TLSSCTCount:           req.TLSSCTCount,
		TLSServerCertificates: toTLSCertificateDTOs(req.TLSServerCertificates),
		TLSServerExtensions:   toTLSExtensionDTOs(req.TLSServerExtensions),
		WebSocketFrames:       toWebSocketFrameDTOs(req.WebSocketFrames),
	}
}

func ToRequestEvent(req *storage.Request, sessionID string) *RequestEventDTO {
	return toRequestDTO(req, sessionID)
}

func toRequestDTOs(items []*storage.Request, sessionID string) []*RequestDTO {
	result := make([]*RequestDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toRequestDTO(item, sessionID))
	}
	return result
}

type FileInfoDTO struct {
	Path     string   `json:"path"`
	Exists   bool     `json:"exists"`
	Download string   `json:"download"`
	Size     int64    `json:"size,omitempty"`
	Modified JSONTime `json:"modified,omitempty"`
}

type LocalIPAddressDTO struct {
	InterfaceName string `json:"interface_name"`
	IPAddress     string `json:"ip_address"`
}
