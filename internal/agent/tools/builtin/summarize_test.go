package builtin

import (
	"strings"
	"testing"

	"github.com/packetmind/packetmind/internal/storage"
)

func TestSummarizeSessionHandler_DoesNotPanicOnFirstHost(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	session := &storage.Session{Name: "capture"}
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	req := &storage.Request{
		SessionID:   session.ID,
		Method:      "GET",
		Scheme:      "https",
		Host:        "api.example.test",
		Path:        "/api/user",
		URL:         "https://api.example.test/api/user",
		StatusCode:  200,
		ContentType: "application/json",
		Headers:     storage.Headers{"Authorization": {"Bearer token"}},
		RespHeaders: storage.Headers{},
		RespBody:    []byte(`{"ok":true}`),
	}
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	handler := newSummarizeSessionHandler(store)
	result, err := handler(map[string]any{"session_id": session.ID}, session.ID)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if result == nil || !strings.Contains(result.Content, "api.example.test") {
		t.Fatalf("summary content = %q, want host", result.Content)
	}
}
