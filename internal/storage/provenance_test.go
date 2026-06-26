package storage

import (
	"testing"
	"time"
)

func TestExtractRequestArtifacts_JSONAndHeaders(t *testing.T) {
	req := &Request{
		QueryString: "token=abc123&uid=42",
		Headers: Headers{
			"Authorization": {"Bearer abc123"},
		},
		Cookies: Cookies{
			"session": "sess-1",
		},
		ContentType: "application/json",
		Body:        []byte(`{"user":{"id":"42"},"token":"abc123"}`),
	}

	artifacts := ExtractRequestArtifacts(req)
	if len(artifacts) < 5 {
		t.Fatalf("expected multiple artifacts, got %d", len(artifacts))
	}

	assertArtifactExists(t, artifacts, ParamLocationQuery, "token", "abc123")
	assertArtifactExists(t, artifacts, ParamLocationHeader, "Authorization", "Bearer abc123")
	assertArtifactExists(t, artifacts, ParamLocationCookie, "session", "sess-1")
	assertArtifactExists(t, artifacts, ParamLocationJSONBody, "id", "42")
	assertArtifactExists(t, artifacts, ParamLocationJSONBody, "token", "abc123")
}

func TestFindPriorResponseSourcesAndTraceValueFlow(t *testing.T) {
	store := mustNewProvenanceTestStorage(t)

	mustCreateTestSession(t, store, &Session{ID: "sess_1", Name: "Test Session"})

	base := time.Now().Add(-2 * time.Minute)
	a := &Request{
		ID:              "req_a",
		CreatedAt:       base,
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/login",
		StatusCode:      200,
		RespContentType: "application/json",
		RespBody:        []byte(`{"token":"abc123","user_id":"u-1"}`),
	}
	b := &Request{
		ID:          "req_b",
		CreatedAt:   base.Add(5 * time.Second),
		Method:      "POST",
		Host:        "api.example.com",
		Path:        "/orders",
		StatusCode:  201,
		Headers:     Headers{"Authorization": {"Bearer abc123"}},
		ContentType: "application/json",
		Body:        []byte(`{"user_id":"u-1"}`),
	}

	mustSaveTestRequest(t, store, a)
	mustSaveTestRequest(t, store, b)

	sources, err := store.FindPriorResponseSources("sess_1", "req_b", "abc123", 10)
	if err != nil {
		t.Fatalf("FindPriorResponseSources failed: %v", err)
	}
	if len(sources) == 0 {
		t.Fatal("expected at least one prior response source")
	}
	if sources[0].RequestID != "req_a" {
		t.Fatalf("expected source req_a, got %s", sources[0].RequestID)
	}

	chain, err := store.TraceValueFlow("sess_1", "req_b", "Authorization", ParamLocationHeader, 10)
	if err != nil {
		t.Fatalf("TraceValueFlow failed: %v", err)
	}
	if len(chain.Links) == 0 {
		t.Fatal("expected at least one provenance link")
	}
	if chain.Links[0].SourceRequestID != "req_a" {
		t.Fatalf("expected best link source req_a, got %s", chain.Links[0].SourceRequestID)
	}
	if chain.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", chain.Confidence)
	}
}

func TestProvenanceQueriesUseExplicitSessionID(t *testing.T) {
	store := mustNewProvenanceTestStorage(t)

	mustCreateTestSession(t, store, &Session{ID: "sess_active", Name: "Active"})
	mustCreateTestSession(t, store, &Session{ID: "sess_trace", Name: "Trace"})
	mustSetActiveTestSession(t, store, "sess_active")

	base := time.Now().Add(-2 * time.Minute)
	a := &Request{
		ID:              "req_a",
		SessionID:       "sess_trace",
		CreatedAt:       base,
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/login",
		StatusCode:      200,
		RespContentType: "application/json",
		RespBody:        []byte(`{"token":"abc123"}`),
	}
	b := &Request{
		ID:          "req_b",
		SessionID:   "sess_trace",
		CreatedAt:   base.Add(5 * time.Second),
		Method:      "POST",
		Host:        "api.example.com",
		Path:        "/orders",
		StatusCode:  201,
		Headers:     Headers{"Authorization": {"Bearer abc123"}},
		ContentType: "application/json",
		Body:        []byte(`{"user_id":"u-1"}`),
	}

	mustSaveTestRequest(t, store, a)
	mustSaveTestRequest(t, store, b)

	sources, err := store.FindPriorResponseSources("sess_trace", "req_b", "abc123", 10)
	if err != nil {
		t.Fatalf("FindPriorResponseSources failed: %v", err)
	}
	if len(sources) == 0 || sources[0].RequestID != "req_a" {
		t.Fatalf("unexpected sources: %+v", sources)
	}

	chain, err := store.TraceValueFlow("sess_trace", "req_b", "Authorization", ParamLocationHeader, 10)
	if err != nil {
		t.Fatalf("TraceValueFlow failed: %v", err)
	}
	if len(chain.Links) == 0 || chain.Links[0].SourceRequestID != "req_a" {
		t.Fatalf("unexpected chain: %+v", chain)
	}
}

func assertArtifactExists(t *testing.T, artifacts []ParamArtifact, location ParamLocation, name, value string) {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Location == location && artifact.Name == name && artifact.Value == value {
			return
		}
	}
	t.Fatalf("expected artifact location=%s name=%s value=%s", location, name, value)
}

func mustNewProvenanceTestStorage(t *testing.T) *Storage {
	t.Helper()
	store, err := NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return store
}

func mustCreateTestSession(t *testing.T, store *Storage, session *Session) {
	t.Helper()
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
}

func mustSetActiveTestSession(t *testing.T, store *Storage, sessionID string) {
	t.Helper()
	if err := store.SetActiveSession(sessionID); err != nil {
		t.Fatalf("SetActiveSession failed: %v", err)
	}
}

func mustSaveTestRequest(t *testing.T, store *Storage, req *Request) {
	t.Helper()
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}
}
