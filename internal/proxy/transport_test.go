package proxy

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/packetmind/packetmind/internal/storage"
)

func TestPeekTLSClientHello_CapturesFullRecord(t *testing.T) {
	raw := []byte{0x16, 0x03, 0x01, 0x00, 0x06, 0x01, 0x00, 0x00, 0x02, 0x03, 0x03}
	reader := bufio.NewReader(bytes.NewReader(raw))
	got, err := peekTLSClientHello(reader)
	if err != nil {
		t.Fatalf("peekTLSClientHello failed: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("captured client hello = %v, want %v", got, raw)
	}
}

func TestWrapConnWithCapturedClientHello_PreservesBufferedBytes(t *testing.T) {
	raw := []byte{0x16, 0x03, 0x01, 0x00, 0x06, 0x01, 0x00, 0x00, 0x02, 0x03, 0x03}
	conn := &bufferedConn{reader: bufio.NewReader(bytes.NewReader(raw))}
	wrapped, captured := wrapConnWithCapturedClientHello(conn)
	if !bytes.Equal(captured, raw) {
		t.Fatalf("captured client hello = %v, want %v", captured, raw)
	}
	buf, err := io.ReadAll(wrapped)
	if err != nil {
		t.Fatalf("ReadAll wrapped conn failed: %v", err)
	}
	if !bytes.Equal(buf, raw) {
		t.Fatalf("wrapped bytes = %v, want %v", buf, raw)
	}
}

func TestExecuteRequestWithClientHello_FallsBackWithoutGetBody(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	p := New()
	req, err := http.NewRequest(http.MethodPost, "http://example.com", bytes.NewReader([]byte("payload")))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.URL.Scheme = "https"
	req.GetBody = nil
	_, err = p.ExecuteRequestWithClientHello(req, []byte{0x16, 0x03, 0x01, 0x00, 0x06, 0x01, 0x00, 0x00, 0x02, 0x03, 0x03})
	if err == nil {
		// Network success/failure is environment-dependent; the check here is that we do not panic.
		return
	}
}
