package proxy

import (
	"bytes"
	"fmt"
	"strings"
)

// parseHostPort parses host:port string and returns host and port
func parseHostPort(addr string) (host string, port int) {
	host = addr
	port = 0

	// Handle [IPv6]:port format
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]: "); idx != -1 {
			host = addr[1:idx]
			fmt.Sscanf(addr[idx+2:], "%d", &port)
		}
		return
	}

	// Handle host:port format
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		host = addr[:idx]
		fmt.Sscanf(addr[idx+1:], "%d", &port)
	}

	return
}

var httpMethods = [][]byte{
	[]byte("GET "),
	[]byte("POST "),
	[]byte("HEAD "),
	[]byte("PUT "),
	[]byte("DELETE "),
	[]byte("OPTIONS "),
	[]byte("PATCH "),
	[]byte("CONNECT "),
	[]byte("TRACE "),
	[]byte("PROPFIND "),
}

func looksLikeHTTP(peek []byte) bool {
	if len(peek) == 0 {
		return false
	}
	for _, method := range httpMethods {
		if bytes.HasPrefix(peek, method) {
			return true
		}
	}
	return false
}

func isTLSHandshake(peek []byte) bool {
	if len(peek) < 3 {
		return false
	}
	// TLS record type 0x16 (Handshake) and version 0x03XX (TLS 1.x)
	return peek[0] == 0x16 && peek[1] == 0x03
}

// limitedWriter writes to underlying writer up to limit bytes
type limitedWriter struct {
	buf       *bytes.Buffer
	limit     int
	truncated bool
}

func newLimitedWriter(limit int) *limitedWriter {
	return &limitedWriter{
		buf:   bytes.NewBuffer(nil),
		limit: limit,
	}
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.buf.Len() >= lw.limit {
		lw.truncated = true
		return len(p), nil
	}
	remaining := lw.limit - lw.buf.Len()
	if len(p) > remaining {
		lw.truncated = true
		lw.buf.Write(p[:remaining])
		return len(p), nil
	}
	return lw.buf.Write(p)
}

func (lw *limitedWriter) Bytes() []byte {
	return lw.buf.Bytes()
}

func (lw *limitedWriter) IsTruncated() bool {
	return lw.truncated
}

// captureWriter captures streamed response body up to a limit while
// tracking total bytes written. Used for streaming proxy responses.
type captureWriter struct {
	buf   bytes.Buffer
	limit int
	total int64
}

func (cw *captureWriter) Write(p []byte) (int, error) {
	cw.total += int64(len(p))
	if cw.buf.Len() < cw.limit {
		remaining := cw.limit - cw.buf.Len()
		if len(p) >= remaining {
			cw.buf.Write(p[:remaining])
		} else {
			cw.buf.Write(p)
		}
	}
	return len(p), nil
}
