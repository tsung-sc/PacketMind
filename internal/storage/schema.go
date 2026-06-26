package storage

import (
	"time"

	"github.com/hashicorp/go-memdb"
)

const (
	tableSession = "session"
	tableRequest = "request"
	tableChat    = "chat_message"
)

func newSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			tableSession: {
				Name: tableSession,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"is_active": {
						Name:    "is_active",
						Unique:  false,
						Indexer: &memdb.BoolFieldIndex{Field: "IsActive"},
					},
					"created_at": {
						Name:    "created_at",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "CreatedAtUnix"},
					},
				},
			},
			tableChat: {
				Name: tableChat,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"session_id": {
						Name:    "session_id",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "SessionID"},
					},
					"session_created_at": {
						Name:   "session_created_at",
						Unique: false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "SessionID"},
								&memdb.IntFieldIndex{Field: "CreatedAtUnix"},
							},
						},
					},
				},
			},
			tableRequest: {
				Name: tableRequest,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"session_id": {
						Name:    "session_id",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "SessionID"},
					},
					"session_host": {
						Name:   "session_host",
						Unique: false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "SessionID"},
								&memdb.StringFieldIndex{Field: "Host"},
							},
						},
					},
					"session_created_at": {
						Name:   "session_created_at",
						Unique: false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "SessionID"},
								&memdb.IntFieldIndex{Field: "CreatedAtUnix"},
							},
						},
					},
					"session_status_code": {
						Name:   "session_status_code",
						Unique: false,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "SessionID"},
								&memdb.IntFieldIndex{Field: "StatusCode"},
							},
						},
					},
					"host": {
						Name:    "host",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Host"},
					},
				},
			},
		},
	}
}

type chatMessageRecord struct {
	ID            string
	SessionID     string
	Role          string
	Content       string
	CreatedAtUnix int64
}

func chatMessageToRecord(msg *ChatMessage) *chatMessageRecord {
	if msg == nil {
		return nil
	}
	return &chatMessageRecord{
		ID:            msg.ID,
		SessionID:     msg.SessionID,
		Role:          msg.Role,
		Content:       msg.Content,
		CreatedAtUnix: msg.CreatedAt.UnixNano(),
	}
}

func recordToChatMessage(rec *chatMessageRecord) *ChatMessage {
	if rec == nil {
		return nil
	}
	return &ChatMessage{
		ID:        rec.ID,
		SessionID: rec.SessionID,
		Role:      rec.Role,
		Content:   rec.Content,
		CreatedAt: timeFromUnixNano(rec.CreatedAtUnix),
	}
}

// sessionRecord is the memdb row type for sessions.
type sessionRecord struct {
	ID            string
	Name          string
	CreatedAtUnix int64
	UpdatedAtUnix int64
	IsActive      bool
	Description   string
}

func sessionToRecord(s *Session) *sessionRecord {
	if s == nil {
		return nil
	}
	return &sessionRecord{
		ID:            s.ID,
		Name:          s.Name,
		CreatedAtUnix: s.CreatedAt.UnixNano(),
		UpdatedAtUnix: s.UpdatedAt.UnixNano(),
		IsActive:      s.IsActive,
		Description:   s.Description,
	}
}

func recordToSession(r *sessionRecord) *Session {
	if r == nil {
		return nil
	}
	return &Session{
		ID:          r.ID,
		Name:        r.Name,
		CreatedAt:   timeFromUnixNano(r.CreatedAtUnix),
		UpdatedAt:   timeFromUnixNano(r.UpdatedAtUnix),
		IsActive:    r.IsActive,
		Description: r.Description,
	}
}

// requestRecord is the memdb row type for requests.
// It mirrors Request but uses int64 for time fields to support memdb IntFieldIndex.
type requestRecord struct {
	ID            string
	SessionID     string
	CreatedAtUnix int64
	UpdatedAtUnix int64

	Method      string
	Scheme      string
	Host        string
	Port        int
	Path        string
	URL         string
	QueryString string
	HTTPVersion string

	Headers     Headers
	Cookies     Cookies
	IsWebSocket bool
	ContentType string
	Body        []byte
	BodySize    int64

	StatusCode      int
	StatusReason    string
	RespHeaders     Headers
	RespContentType string
	RespBody        []byte
	RespBodySize    int64

	Duration   int64
	RemoteAddr string
	ClientAddr string
	ServerAddr string
	Notes      string
	Error      string

	RequestStartTimeUnix  int64
	RequestEndTimeUnix    int64
	ResponseStartTimeUnix int64
	ResponseEndTimeUnix   int64

	DNSDuration          int64
	ConnectDuration      int64
	TLSHandshakeDuration int64
	RequestDuration      int64
	ResponseDuration     int64
	LatencyDuration      int64

	KeepAlive      bool
	TLSVersion     string
	TLSCipherSuite string

	TLSServerName         string
	TLSDidResume          bool
	TLSALPN               string
	TLSCurveID            string
	TLSOCSPStapled        bool
	TLSSCTCount           int
	TLSClientHelloRaw     []byte
	TLSServerCertificates []TLSCertificate
	TLSServerExtensions   []TLSExtension
	WebSocketFrames       []WebSocketFrame
}

func requestToRecord(r *Request) *requestRecord {
	if r == nil {
		return nil
	}
	return &requestRecord{
		ID:                    r.ID,
		SessionID:             r.SessionID,
		CreatedAtUnix:         r.CreatedAt.UnixNano(),
		UpdatedAtUnix:         r.UpdatedAt.UnixNano(),
		Method:                r.Method,
		Scheme:                r.Scheme,
		Host:                  r.Host,
		Port:                  r.Port,
		Path:                  r.Path,
		URL:                   r.URL,
		QueryString:           r.QueryString,
		HTTPVersion:           r.HTTPVersion,
		Headers:               r.Headers,
		Cookies:               r.Cookies,
		IsWebSocket:           r.IsWebSocket,
		ContentType:           r.ContentType,
		Body:                  r.Body,
		BodySize:              r.BodySize,
		StatusCode:            r.StatusCode,
		StatusReason:          r.StatusReason,
		RespHeaders:           r.RespHeaders,
		RespContentType:       r.RespContentType,
		RespBody:              r.RespBody,
		RespBodySize:          r.RespBodySize,
		Duration:              r.Duration,
		RemoteAddr:            r.RemoteAddr,
		ClientAddr:            r.ClientAddr,
		ServerAddr:            r.ServerAddr,
		Notes:                 r.Notes,
		Error:                 r.Error,
		RequestStartTimeUnix:  r.RequestStartTime.UnixNano(),
		RequestEndTimeUnix:    r.RequestEndTime.UnixNano(),
		ResponseStartTimeUnix: r.ResponseStartTime.UnixNano(),
		ResponseEndTimeUnix:   r.ResponseEndTime.UnixNano(),
		DNSDuration:           r.DNSDuration,
		ConnectDuration:       r.ConnectDuration,
		TLSHandshakeDuration:  r.TLSHandshakeDuration,
		RequestDuration:       r.RequestDuration,
		ResponseDuration:      r.ResponseDuration,
		LatencyDuration:       r.LatencyDuration,
		KeepAlive:             r.KeepAlive,
		TLSVersion:            r.TLSVersion,
		TLSCipherSuite:        r.TLSCipherSuite,
		TLSServerName:         r.TLSServerName,
		TLSDidResume:          r.TLSDidResume,
		TLSALPN:               r.TLSALPN,
		TLSCurveID:            r.TLSCurveID,
		TLSOCSPStapled:        r.TLSOCSPStapled,
		TLSSCTCount:           r.TLSSCTCount,
		TLSClientHelloRaw:     r.TLSClientHelloRaw,
		TLSServerCertificates: r.TLSServerCertificates,
		TLSServerExtensions:   r.TLSServerExtensions,
		WebSocketFrames:       r.WebSocketFrames,
	}
}

func recordToRequest(r *requestRecord) *Request {
	if r == nil {
		return nil
	}
	return &Request{
		ID:                    r.ID,
		SessionID:             r.SessionID,
		CreatedAt:             timeFromUnixNano(r.CreatedAtUnix),
		UpdatedAt:             timeFromUnixNano(r.UpdatedAtUnix),
		Method:                r.Method,
		Scheme:                r.Scheme,
		Host:                  r.Host,
		Port:                  r.Port,
		Path:                  r.Path,
		URL:                   r.URL,
		QueryString:           r.QueryString,
		HTTPVersion:           r.HTTPVersion,
		Headers:               r.Headers,
		Cookies:               r.Cookies,
		IsWebSocket:           r.IsWebSocket,
		ContentType:           r.ContentType,
		Body:                  r.Body,
		BodySize:              r.BodySize,
		StatusCode:            r.StatusCode,
		StatusReason:          r.StatusReason,
		RespHeaders:           r.RespHeaders,
		RespContentType:       r.RespContentType,
		RespBody:              r.RespBody,
		RespBodySize:          r.RespBodySize,
		Duration:              r.Duration,
		RemoteAddr:            r.RemoteAddr,
		ClientAddr:            r.ClientAddr,
		ServerAddr:            r.ServerAddr,
		Notes:                 r.Notes,
		Error:                 r.Error,
		RequestStartTime:      timeFromUnixNano(r.RequestStartTimeUnix),
		RequestEndTime:        timeFromUnixNano(r.RequestEndTimeUnix),
		ResponseStartTime:     timeFromUnixNano(r.ResponseStartTimeUnix),
		ResponseEndTime:       timeFromUnixNano(r.ResponseEndTimeUnix),
		DNSDuration:           r.DNSDuration,
		ConnectDuration:       r.ConnectDuration,
		TLSHandshakeDuration:  r.TLSHandshakeDuration,
		RequestDuration:       r.RequestDuration,
		ResponseDuration:      r.ResponseDuration,
		LatencyDuration:       r.LatencyDuration,
		KeepAlive:             r.KeepAlive,
		TLSVersion:            r.TLSVersion,
		TLSCipherSuite:        r.TLSCipherSuite,
		TLSServerName:         r.TLSServerName,
		TLSDidResume:          r.TLSDidResume,
		TLSALPN:               r.TLSALPN,
		TLSCurveID:            r.TLSCurveID,
		TLSOCSPStapled:        r.TLSOCSPStapled,
		TLSSCTCount:           r.TLSSCTCount,
		TLSClientHelloRaw:     r.TLSClientHelloRaw,
		TLSServerCertificates: r.TLSServerCertificates,
		TLSServerExtensions:   r.TLSServerExtensions,
		WebSocketFrames:       r.WebSocketFrames,
	}
}

func timeFromUnixNano(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}
