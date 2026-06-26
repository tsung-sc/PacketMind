package storage

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Session struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsActive    bool      `json:"is_active"`
	Description string    `json:"description"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Request struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Method      string `json:"method"`
	Scheme      string `json:"scheme"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	QueryString string `json:"query_string"`
	HTTPVersion string `json:"http_version"`

	Headers     Headers `json:"headers"`
	Cookies     Cookies `json:"cookies"`
	IsWebSocket bool    `json:"is_websocket"`
	ContentType string  `json:"content_type"`
	Body        []byte  `json:"body"`
	BodySize    int64   `json:"body_size"`

	StatusCode      int     `json:"status_code"`
	StatusReason    string  `json:"status_reason"`
	RespHeaders     Headers `json:"resp_headers"`
	RespContentType string  `json:"resp_content_type"`
	RespBody        []byte  `json:"resp_body"`
	RespBodySize    int64   `json:"resp_body_size"`

	Duration   int64  `json:"duration"`
	RemoteAddr string `json:"remote_addr"`
	ClientAddr string `json:"client_addr"`
	ServerAddr string `json:"server_addr"`
	Notes      string `json:"notes"`
	Error      string `json:"error"`

	RequestStartTime  time.Time `json:"request_start_time"`
	RequestEndTime    time.Time `json:"request_end_time"`
	ResponseStartTime time.Time `json:"response_start_time"`
	ResponseEndTime   time.Time `json:"response_end_time"`

	DNSDuration          int64 `json:"dns_duration"`
	ConnectDuration      int64 `json:"connect_duration"`
	TLSHandshakeDuration int64 `json:"tls_handshake_duration"`
	RequestDuration      int64 `json:"request_duration"`
	ResponseDuration     int64 `json:"response_duration"`
	LatencyDuration      int64 `json:"latency_duration"`
	KeepAlive            bool  `json:"keep_alive"`

	TLSVersion        string `json:"tls_version"`
	TLSCipherSuite    string `json:"tls_cipher_suite"`
	TLSServerName     string `json:"tls_server_name"`
	TLSDidResume      bool   `json:"tls_did_resume"`
	TLSALPN           string `json:"tls_alpn"`
	TLSCurveID        string `json:"tls_curve_id"`
	TLSOCSPStapled    bool   `json:"tls_ocsp_stapled"`
	TLSSCTCount       int    `json:"tls_sct_count"`
	TLSClientHelloRaw []byte `json:"tls_client_hello_raw,omitempty"`

	TLSServerCertificates []TLSCertificate `json:"tls_server_certificates"`
	TLSServerExtensions   []TLSExtension   `json:"tls_server_extensions"`
	WebSocketFrames       []WebSocketFrame `json:"websocket_frames"`
}

type WebSocketFrame struct {
	ID          string    `json:"id"`
	Direction   string    `json:"direction"`
	Opcode      int       `json:"opcode"`
	FrameType   string    `json:"frame_type"`
	Payload     []byte    `json:"payload"`
	PayloadSize int64     `json:"payload_size"`
	CreatedAt   time.Time `json:"created_at"`
	Fin         bool      `json:"fin"`
	Masked      bool      `json:"masked"`
}

type TLSCertificate struct {
	SubjectCommonName     string         `json:"subject_common_name"`
	Subject               string         `json:"subject"`
	IssuerCommonName      string         `json:"issuer_common_name"`
	Issuer                string         `json:"issuer"`
	SerialNumber          string         `json:"serial_number"`
	DNSNames              []string       `json:"dns_names"`
	EmailAddresses        []string       `json:"email_addresses"`
	IPAddresses           []string       `json:"ip_addresses"`
	Version               int            `json:"version"`
	IsCA                  bool           `json:"is_ca"`
	SignatureAlgorithm    string         `json:"signature_algorithm"`
	PublicKeyAlgorithm    string         `json:"public_key_algorithm"`
	NotBefore             time.Time      `json:"not_before"`
	NotAfter              time.Time      `json:"not_after"`
	OCSPServers           []string       `json:"ocsp_servers"`
	IssuingCertificateURL []string       `json:"issuing_certificate_url"`
	Extensions            []TLSExtension `json:"extensions"`
}

type TLSExtension struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Headers map[string][]string

func (h Headers) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}
	return json.Marshal(h)
}

func (h *Headers) Scan(value interface{}) error {
	if value == nil {
		*h = make(Headers)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, h)
}

func (h Headers) Get(key string) string {
	if values, ok := h[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

type Cookies map[string]string

func (c Cookies) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

func (c *Cookies) Scan(value interface{}) error {
	if value == nil {
		*c = make(Cookies)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

type RequestListOptions struct {
	SessionID  string     `json:"session_id"`
	Host       string     `json:"host"`
	Method     string     `json:"method"`
	StatusCode int        `json:"status_code"`
	Search     string     `json:"search"`
	StartTime  *time.Time `json:"start_time"`
	EndTime    *time.Time `json:"end_time"`
	SortBy     string     `json:"sort_by"`
	SortOrder  string     `json:"sort_order"`
}

type FindInSessionOptions struct {
	SessionID         string `json:"session_id"`
	Query             string `json:"query"`
	IsRegex           bool   `json:"is_regex"`
	IsCaseSensitive   bool   `json:"is_case_sensitive"`
	IsWholeWord       bool   `json:"is_whole_word"`
	IncludeReqURL     bool   `json:"include_req_url"`
	IncludeReqHeader  bool   `json:"include_req_header"`
	IncludeReqBody    bool   `json:"include_req_body"`
	IncludeRespHeader bool   `json:"include_resp_header"`
	IncludeRespBody   bool   `json:"include_resp_body"`
	IncludeNotes      bool   `json:"include_notes"`
	IncludeError      bool   `json:"include_error"`
}

type FindInSessionMatch struct {
	RequestID        string `json:"request_id"`
	SessionID        string `json:"session_id"`
	RequestCreatedAt string `json:"request_created_at"`
	Method           string `json:"method"`
	Host             string `json:"host"`
	Path             string `json:"path"`
	URL              string `json:"url"`
	StatusCode       int    `json:"status_code"`
	Field            string `json:"field"`
	FieldKey         string `json:"field_key,omitempty"`
	Preview          string `json:"preview"`
	MatchText        string `json:"match_text"`
}

type AnalysisRequest struct {
	RequestID      string `json:"request_id"`
	TargetField    string `json:"target_field"`
	TargetLocation string `json:"target_location"`
	TargetValue    string `json:"target_value"`
	Query          string `json:"query"`
	ModelID        string `json:"model_id"`
	SessionID      string `json:"session_id"`
}
