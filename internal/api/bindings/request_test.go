package bindings

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/proxy"
	"github.com/packetmind/packetmind/internal/storage"
)

func TestComposeRequest_UsesProxyInstanceForExternalProxy(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	proxyHits := 0
	externalProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		w.Header().Set("X-Proxy-Hit", "1")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "proxied compose")
	}))
	defer externalProxy.Close()

	settings := config.DefaultPacketMindSettings()
	settings.Proxy.ExternalProxy.Enabled = true
	settings.Proxy.ExternalProxy.Scheme = "http"
	settings.Proxy.ExternalProxy.Host = externalProxy.Listener.Addr().(*net.TCPAddr).IP.String()
	settings.Proxy.ExternalProxy.Port = externalProxy.Listener.Addr().(*net.TCPAddr).Port
	settings.Proxy.ExternalProxy.BypassHosts = nil

	prox := proxy.New()
	storage.Default = store
	proxy.Default = prox
	if err := prox.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	api := NewRequestAPI()
	resp := api.ComposeRequest(ComposeRequestOptions{
		Method: "GET",
		URL:    "http://service.invalid/compose",
	})
	if resp.Code != 0 {
		t.Fatalf("ComposeRequest code = %d, want 0, message=%s", resp.Code, resp.Message)
	}
	result, ok := resp.Data.(ReplayResult)
	if !ok {
		t.Fatalf("ComposeRequest data type = %T, want ReplayResult", resp.Data)
	}
	if proxyHits != 1 {
		t.Fatalf("external proxy hits = %d, want 1", proxyHits)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil {
		t.Fatalf("expected active session")
	}

	requests, err := store.ListRequests(storage.RequestListOptions{SessionID: active.ID})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	var composedRecords []*storage.Request
	for _, req := range requests {
		if req.ID == result.RequestID {
			composedRecords = append(composedRecords, req)
		}
	}
	if len(composedRecords) != 1 {
		t.Fatalf("composed records len=%d, want 1", len(composedRecords))
	}
	final := composedRecords[0]
	if final.StatusCode != http.StatusCreated {
		t.Fatalf("stored status = %d, want %d", final.StatusCode, http.StatusCreated)
	}
	if string(final.RespBody) != "proxied compose" {
		t.Fatalf("stored response body = %q, want %q", string(final.RespBody), "proxied compose")
	}
	if final.RespHeaders.Get("X-Proxy-Hit") != "1" {
		t.Fatalf("expected proxy response header to be recorded")
	}
	if final.Notes != "Composed request" {
		t.Fatalf("stored notes = %q, want %q", final.Notes, "Composed request")
	}
}

func TestReplayRequest_UsesProxyInstanceForExternalProxy(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := &storage.Session{Name: "replay-session", IsActive: true}
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	original := &storage.Request{
		Method:      http.MethodPost,
		Scheme:      "http",
		Host:        "service.invalid",
		Port:        80,
		Path:        "/replay",
		URL:         "http://service.invalid/replay",
		Headers:     storage.Headers{"Content-Type": {"text/plain"}},
		ContentType: "text/plain",
		Body:        []byte("original body"),
		BodySize:    int64(len("original body")),
	}
	if err := store.SaveRequest(original); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}

	proxyHits := 0
	externalProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits++
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Proxy-Hit", "1")
		w.Header().Set("X-Seen-Method", r.Method)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(body)
	}))
	defer externalProxy.Close()

	proxyAddr := externalProxy.Listener.Addr().(*net.TCPAddr)
	settings := config.DefaultPacketMindSettings()
	settings.Proxy.ExternalProxy.Enabled = true
	settings.Proxy.ExternalProxy.Scheme = "http"
	settings.Proxy.ExternalProxy.Host = proxyAddr.IP.String()
	settings.Proxy.ExternalProxy.Port = proxyAddr.Port
	settings.Proxy.ExternalProxy.BypassHosts = nil

	prox := proxy.New()
	storage.Default = store
	proxy.Default = prox
	if err := prox.ApplySettings(settings); err != nil {
		t.Fatalf("ApplySettings failed: %v", err)
	}

	api := NewRequestAPI()
	resp := api.ReplayRequest(session.ID, original.ID, ReplayRequestOptions{Body: "rewritten body"})
	if resp.Code != 0 {
		t.Fatalf("ReplayRequest code = %d, want 0, message=%s", resp.Code, resp.Message)
	}
	result, ok := resp.Data.(ReplayResult)
	if !ok {
		t.Fatalf("ReplayRequest data type = %T, want ReplayResult", resp.Data)
	}
	if proxyHits != 1 {
		t.Fatalf("external proxy hits = %d, want 1", proxyHits)
	}

	requests, err := store.ListRequests(storage.RequestListOptions{SessionID: session.ID})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	var replayedRecords []*storage.Request
	for _, req := range requests {
		if req.ID == result.RequestID {
			replayedRecords = append(replayedRecords, req)
		}
	}
	if len(replayedRecords) != 1 {
		t.Fatalf("replayed records len=%d, want 1", len(replayedRecords))
	}

	replayed := replayedRecords[0]
	if replayed.StatusCode != http.StatusAccepted {
		t.Fatalf("replayed status = %d, want %d", replayed.StatusCode, http.StatusAccepted)
	}
	if string(replayed.RespBody) != "rewritten body" {
		t.Fatalf("replayed response body = %q, want %q", string(replayed.RespBody), "rewritten body")
	}
	if replayed.RespHeaders.Get("X-Proxy-Hit") != "1" {
		t.Fatalf("expected proxy response header to be recorded")
	}
	if replayed.Notes != "Replayed from "+original.ID {
		t.Fatalf("replayed notes = %q, want %q", replayed.Notes, "Replayed from "+original.ID)
	}
	if string(replayed.Body) != "rewritten body" {
		t.Fatalf("replayed body = %q, want %q", string(replayed.Body), "rewritten body")
	}
}
