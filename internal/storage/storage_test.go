package storage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCreateSession_AutoActivatesFirstOnly(t *testing.T) {
	store := mustNewTestStorage(t)

	first := &Session{Name: "first"}
	if err := store.CreateSession(first); err != nil {
		t.Fatalf("CreateSession first failed: %v", err)
	}
	if !first.IsActive {
		t.Fatalf("expected first session active")
	}

	second := &Session{Name: "second"}
	if err := store.CreateSession(second); err != nil {
		t.Fatalf("CreateSession second failed: %v", err)
	}
	if second.IsActive {
		t.Fatalf("expected second session inactive by default")
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != first.ID {
		t.Fatalf("expected first session to remain active")
	}
}

func TestSetActiveSession_ValidateAndSwitch(t *testing.T) {
	store := mustNewTestStorage(t)

	a := &Session{Name: "a"}
	b := &Session{Name: "b"}
	mustCreateSession(t, store, a)
	mustCreateSession(t, store, b)

	if err := store.SetActiveSession("missing"); err == nil {
		t.Fatalf("expected error for missing session")
	}

	if err := store.SetActiveSession(b.ID); err != nil {
		t.Fatalf("SetActiveSession failed: %v", err)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != b.ID {
		t.Fatalf("expected session b active")
	}
}

func TestUpdateSession_UpdatesNameAndDescription(t *testing.T) {
	store := mustNewTestStorage(t)

	s := &Session{Name: "old", Description: "old desc"}
	mustCreateSession(t, store, s)

	updated, err := store.UpdateSession(s.ID, "new", "new desc")
	if err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}
	if updated.Name != "new" || updated.Description != "new desc" {
		t.Fatalf("unexpected updated session: %+v", updated)
	}

	if _, err := store.UpdateSession("missing", "x", "y"); err == nil {
		t.Fatalf("expected error for missing session")
	}
}

func TestDeleteSession_ReassignsActiveSession(t *testing.T) {
	store := mustNewTestStorage(t)

	a := &Session{Name: "a"}
	b := &Session{Name: "b"}
	mustCreateSession(t, store, a)
	mustCreateSession(t, store, b)
	mustSetActiveSession(t, store, b.ID)

	if err := store.DeleteSession(b.ID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != a.ID {
		t.Fatalf("expected fallback active session a, got %+v", active)
	}

	if err := store.DeleteSession("missing"); err == nil {
		t.Fatalf("expected error for missing session delete")
	}
}

func TestGetSession_DoesNotEmbedRequests(t *testing.T) {
	store := mustNewTestStorage(t)

	session := &Session{Name: "active"}
	mustCreateSession(t, store, session)
	req := &Request{Method: "GET", Host: "example.com", Path: "/", URL: "http://example.com/"}
	mustSaveRequest(t, store, req)

	got, err := store.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session")
	}
}

func TestGetSessionView_EmbedsDerivedRequestsAndHostGroups(t *testing.T) {
	store := mustNewTestStorage(t)

	session := &Session{Name: "active"}
	mustCreateSession(t, store, session)
	req1 := &Request{Method: "GET", Host: "example.com", Path: "/a", URL: "http://example.com/a"}
	req2 := &Request{Method: "GET", Host: "example.com", Path: "/b", URL: "http://example.com/b"}
	mustSaveRequest(t, store, req1)
	mustSaveRequest(t, store, req2)

	view, err := store.GetSessionView(session.ID)
	if err != nil {
		t.Fatalf("GetSessionView failed: %v", err)
	}
	if view == nil || len(view.Requests) != 2 {
		t.Fatalf("expected 2 derived requests, got %+v", view)
	}
	if len(view.HostGroups["example.com"]) != 2 {
		t.Fatalf("expected host group for example.com, got %+v", view.HostGroups)
	}
}

func TestSaveRequest_UsesActiveSessionWhenSessionIDBlank(t *testing.T) {
	store := mustNewTestStorage(t)

	session := &Session{Name: "active"}
	mustCreateSession(t, store, session)

	req := &Request{Method: "GET", Host: "example.com", Path: "/", URL: "http://example.com/"}
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}
	if req.SessionID != session.ID {
		t.Fatalf("request sessionID = %q, want %q", req.SessionID, session.ID)
	}
}

func TestSaveRequest_WithoutActiveSessionReturnsError(t *testing.T) {
	store := mustNewTestStorage(t)

	req := &Request{Method: "GET", Host: "example.com", Path: "/", URL: "http://example.com/"}
	if err := store.SaveRequest(req); err == nil {
		t.Fatal("expected no active session error")
	}
}

func TestGetRequest_RequiresExplicitSessionID(t *testing.T) {
	store := mustNewTestStorage(t)

	session := &Session{Name: "active"}
	mustCreateSession(t, store, session)

	req := &Request{Method: "GET", Host: "example.com", Path: "/", URL: "http://example.com/"}
	mustSaveRequest(t, store, req)

	if _, err := store.GetRequest("", req.ID); err == nil {
		t.Fatal("expected blank sessionID to fail")
	}

	got, err := store.GetRequest(session.ID, req.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if got == nil || got.ID != req.ID {
		t.Fatalf("unexpected request: %+v", got)
	}
}

func TestListRequests_UsesActiveSessionWhenSessionIDBlank(t *testing.T) {
	store := mustNewTestStorage(t)

	active := &Session{Name: "active"}
	mustCreateSession(t, store, active)
	reqA := &Request{Method: "GET", Host: "example.com", Path: "/a", URL: "http://example.com/a"}
	mustSaveRequest(t, store, reqA)

	other := &Session{Name: "other"}
	mustCreateSession(t, store, other)
	mustSetActiveSession(t, store, other.ID)
	reqB := &Request{Method: "GET", Host: "example.org", Path: "/b", URL: "http://example.org/b"}
	mustSaveRequest(t, store, reqB)

	requests, err := store.ListRequests(RequestListOptions{})
	if err != nil {
		t.Fatalf("ListRequests failed: %v", err)
	}
	if len(requests) != 1 || requests[0].ID != reqB.ID {
		t.Fatalf("expected only active-session request %q, got %+v", reqB.ID, requests)
	}
}

func TestImportAll_PreservesActiveSessionSelection(t *testing.T) {
	store := mustNewTestStorage(t)

	first := &Session{Name: "first"}
	second := &Session{Name: "second"}
	mustCreateSession(t, store, first)
	mustCreateSession(t, store, second)
	mustSetActiveSession(t, store, second.ID)

	mustSaveRequest(t, store, &Request{
		ID:        "req_export_1",
		SessionID: second.ID,
		Method:    "GET",
		Host:      "api.example.com",
		Path:      "/export-check",
	})

	exported, err := store.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	clone, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage clone failed: %v", err)
	}
	if err := clone.ImportAll(exported); err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	active, err := clone.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != second.ID {
		t.Fatalf("expected imported active session %q, got %+v", second.ID, active)
	}

	var out exportData
	if err := json.Unmarshal(exported, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if out.ActiveSession != second.ID {
		t.Fatalf("exported ActiveSession = %q, want %q", out.ActiveSession, second.ID)
	}
	hasNestedRequests := false
	for _, session := range out.Sessions {
		if session != nil && len(session.Requests) > 0 {
			hasNestedRequests = true
			break
		}
	}
	if !hasNestedRequests {
		t.Fatalf("expected export payload to preserve nested requests, got %+v", out.Sessions)
	}
}

func TestImportAll_AssignsNewestSessionActiveWhenMissingActiveSelection(t *testing.T) {
	store := mustNewTestStorage(t)

	payload := exportData{
		Sessions: []*exportSession{
			{ID: "sess_old", Name: "old", CreatedAt: mustParseTestTime(t, "2026-05-22T10:00:00Z")},
			{ID: "sess_new", Name: "new", CreatedAt: mustParseTestTime(t, "2026-05-22T11:00:00Z")},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := store.ImportAll(data); err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != "sess_new" {
		t.Fatalf("expected newest imported session active, got %+v", active)
	}
}

func TestImportAll_FallsBackFromStaleActiveSessionToEmbeddedActive(t *testing.T) {
	store := mustNewTestStorage(t)

	payload := exportData{
		ActiveSession: "stale-session",
		Sessions: []*exportSession{
			{ID: "sess_old", Name: "old", CreatedAt: mustParseTestTime(t, "2026-05-22T10:00:00Z"), IsActive: true},
			{ID: "sess_new", Name: "new", CreatedAt: mustParseTestTime(t, "2026-05-22T11:00:00Z")},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := store.ImportAll(data); err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	active, err := store.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != "sess_old" {
		t.Fatalf("expected embedded active session, got %+v", active)
	}
}

func TestCloneRequest_DeepCopiesTLSSlices(t *testing.T) {
	original := &Request{
		TLSServerCertificates: []TLSCertificate{{
			DNSNames:              []string{"dns-1"},
			EmailAddresses:        []string{"mail-1"},
			IPAddresses:           []string{"10.0.0.1"},
			OCSPServers:           []string{"ocsp-1"},
			IssuingCertificateURL: []string{"issuer-1"},
			Extensions:            []TLSExtension{{ID: 1, Name: "ext-1", Value: "value-1"}},
		}},
		TLSServerExtensions: []TLSExtension{{ID: 2, Name: "server-ext", Value: "server-value"}},
	}

	cloned := cloneRequest(original)
	if cloned == nil {
		t.Fatal("expected clone")
	}

	original.TLSServerCertificates[0].DNSNames[0] = "dns-mutated"
	original.TLSServerCertificates[0].EmailAddresses[0] = "mail-mutated"
	original.TLSServerCertificates[0].IPAddresses[0] = "10.0.0.2"
	original.TLSServerCertificates[0].OCSPServers[0] = "ocsp-mutated"
	original.TLSServerCertificates[0].IssuingCertificateURL[0] = "issuer-mutated"
	original.TLSServerCertificates[0].Extensions[0].Value = "value-mutated"
	original.TLSServerExtensions[0].Value = "server-mutated"

	if cloned.TLSServerCertificates[0].DNSNames[0] != "dns-1" {
		t.Fatalf("DNSNames aliased: %+v", cloned.TLSServerCertificates[0].DNSNames)
	}
	if cloned.TLSServerCertificates[0].EmailAddresses[0] != "mail-1" {
		t.Fatalf("EmailAddresses aliased: %+v", cloned.TLSServerCertificates[0].EmailAddresses)
	}
	if cloned.TLSServerCertificates[0].IPAddresses[0] != "10.0.0.1" {
		t.Fatalf("IPAddresses aliased: %+v", cloned.TLSServerCertificates[0].IPAddresses)
	}
	if cloned.TLSServerCertificates[0].OCSPServers[0] != "ocsp-1" {
		t.Fatalf("OCSPServers aliased: %+v", cloned.TLSServerCertificates[0].OCSPServers)
	}
	if cloned.TLSServerCertificates[0].IssuingCertificateURL[0] != "issuer-1" {
		t.Fatalf("IssuingCertificateURL aliased: %+v", cloned.TLSServerCertificates[0].IssuingCertificateURL)
	}
	if cloned.TLSServerCertificates[0].Extensions[0].Value != "value-1" {
		t.Fatalf("certificate extensions aliased: %+v", cloned.TLSServerCertificates[0].Extensions)
	}
	if cloned.TLSServerExtensions[0].Value != "server-value" {
		t.Fatalf("server extensions aliased: %+v", cloned.TLSServerExtensions)
	}
}

func TestSaveAndListChatMessages_RoundTripOrdered(t *testing.T) {
	store := mustNewTestStorage(t)
	session := &Session{Name: "chat"}
	mustCreateSession(t, store, session)

	first := &ChatMessage{SessionID: session.ID, Role: "user", Content: "hello", CreatedAt: mustParseTestTime(t, "2026-05-26T10:00:00Z")}
	second := &ChatMessage{SessionID: session.ID, Role: "assistant", Content: "world", CreatedAt: mustParseTestTime(t, "2026-05-26T10:00:01Z")}
	if err := store.SaveChatMessage(first); err != nil {
		t.Fatalf("SaveChatMessage first failed: %v", err)
	}
	if err := store.SaveChatMessage(second); err != nil {
		t.Fatalf("SaveChatMessage second failed: %v", err)
	}

	messages, err := store.ListChatMessages(session.ID)
	if err != nil {
		t.Fatalf("ListChatMessages failed: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Content != "hello" || messages[1].Content != "world" {
		t.Fatalf("unexpected messages order: %+v", messages)
	}
}

func TestDeleteSession_CascadesChatMessages(t *testing.T) {
	store := mustNewTestStorage(t)
	session := &Session{Name: "chat-delete"}
	mustCreateSession(t, store, session)
	if err := store.SaveChatMessage(&ChatMessage{SessionID: session.ID, Role: "user", Content: "to-delete"}); err != nil {
		t.Fatalf("SaveChatMessage failed: %v", err)
	}

	if err := store.DeleteSession(session.ID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	messages, err := store.ListChatMessages(session.ID)
	if err != nil {
		t.Fatalf("ListChatMessages after delete failed: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 chat messages after session delete, got %d", len(messages))
	}
}

func TestExportImportAll_PreservesChatMessages(t *testing.T) {
	store := mustNewTestStorage(t)
	session := &Session{Name: "chat-export"}
	mustCreateSession(t, store, session)
	if err := store.SaveChatMessage(&ChatMessage{SessionID: session.ID, Role: "user", Content: "persist me", CreatedAt: mustParseTestTime(t, "2026-05-26T10:00:00Z")}); err != nil {
		t.Fatalf("SaveChatMessage failed: %v", err)
	}

	exported, err := store.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	clone, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	if err := clone.ImportAll(exported); err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	messages, err := clone.ListChatMessages(session.ID)
	if err != nil {
		t.Fatalf("ListChatMessages failed: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "persist me" {
		t.Fatalf("unexpected imported chat messages: %+v", messages)
	}
}

func mustNewTestStorage(t *testing.T) *Storage {
	t.Helper()
	store, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return store
}

func mustCreateSession(t *testing.T, store *Storage, session *Session) {
	t.Helper()
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
}

func mustSetActiveSession(t *testing.T, store *Storage, sessionID string) {
	t.Helper()
	if err := store.SetActiveSession(sessionID); err != nil {
		t.Fatalf("SetActiveSession failed: %v", err)
	}
}

func mustSaveRequest(t *testing.T, store *Storage, req *Request) {
	t.Helper()
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest failed: %v", err)
	}
}

func mustParseTestTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse failed: %v", err)
	}
	return parsed
}
