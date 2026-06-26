package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	agenttools "github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/storage"
)

type stubLLMClient struct{}

func (s *stubLLMClient) Stream(_ context.Context, _ []*llmtypes.LLMMessage, _ ...llmtypes.LLMOption) (*llmtypes.LLMStreamReader, error) {
	sr, sw := llmtypes.NewLLMStreamReader(1)
	go func() {
		defer sw.Close()
		sw.Send(&llmtypes.LLMMessage{Role: llmtypes.RoleAssistant, Content: "stub"})
	}()
	return sr, nil
}

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()

	store, err := storage.NewStorage("")
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	return store
}

func mustCreateSession(t *testing.T, store *storage.Storage, session *storage.Session) {
	t.Helper()
	if err := store.CreateSession(session); err != nil {
		t.Fatalf("CreateSession %s failed: %v", session.ID, err)
	}
}

func mustSaveRequest(t *testing.T, store *storage.Storage, req *storage.Request) {
	t.Helper()
	if err := store.SaveRequest(req); err != nil {
		t.Fatalf("SaveRequest %s failed: %v", req.ID, err)
	}
}

func joinMessageContents(messages []*llmtypes.LLMMessage) string {
	var b strings.Builder
	for _, msg := range messages {
		b.WriteString(msg.Content)
	}
	return b.String()
}

func TestExecuteTool_TraceValueFlow(t *testing.T) {
	store := newTestStorage(t)

	mustCreateSession(t, store, &storage.Session{ID: "sess_1", Name: "Test"})

	base := time.Now().Add(-1 * time.Minute)
	mustSaveRequest(t, store, &storage.Request{
		ID:              "req_a",
		CreatedAt:       base,
		Method:          "GET",
		Host:            "api.example.com",
		Path:            "/login",
		StatusCode:      200,
		RespContentType: "application/json",
		RespBody:        []byte(`{"token":"abc123"}`),
	})
	mustSaveRequest(t, store, &storage.Request{
		ID:        "req_b",
		CreatedAt: base.Add(3 * time.Second),
		Method:    "POST",
		Host:      "api.example.com",
		Path:      "/profile",
		Headers:   storage.Headers{"Authorization": {"Bearer abc123"}},
	})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), "trace_value_flow", `{"request_id":"req_b","field_name":"Authorization","location":"header","limit":5}`, "sess_1")
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if !strings.Contains(result.Content, "req_a") {
		t.Fatalf("expected result to mention req_a, got %s", result.Content)
	}
	if !strings.Contains(result.Summary, "Authorization") {
		t.Fatalf("expected summary to mention target field, got %s", result.Summary)
	}
}

func TestExecuteTool_SearchAllFields(t *testing.T) {
	store := newTestStorage(t)

	mustCreateSession(t, store, &storage.Session{ID: "sess_search", Name: "Test"})

	base := time.Now().Add(-2 * time.Minute)
	mustSaveRequest(t, store, &storage.Request{
		ID:          "req_search_1",
		CreatedAt:   base,
		Method:      "POST",
		Host:        "api.example.com",
		Path:        "/login",
		QueryString: "source=mobile&region=us",
		Headers: storage.Headers{
			"Authorization": {"Bearer token-abc-123"},
		},
		Cookies: storage.Cookies{
			"session_token": "token-abc-123",
		},
		ContentType: "application/json",
		Body:        []byte(`{"token":"token-abc-123"}`),
		RespHeaders: storage.Headers{
			"X-Trace-Token": {"token-abc-123"},
		},
		RespContentType: "application/json",
		RespBody:        []byte(`{"echo":"token-abc-123"}`),
	})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolSearchAllFields, `{"session_id":"sess_search","value":"token-abc-123","limit":20}`, "")
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	if !strings.Contains(result.Summary, "across all request fields") {
		t.Fatalf("expected summary to mention all fields, got: %s", result.Summary)
	}
	if len(result.RequestIDs) == 0 || result.RequestIDs[0] != "req_search_1" {
		t.Fatalf("expected request_ids to include req_search_1, got: %#v", result.RequestIDs)
	}

	if !strings.Contains(result.Content, `"field":"headers.Authorization"`) {
		t.Fatalf("expected header match in content, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"field":"body"`) {
		t.Fatalf("expected body match in content, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"field":"response_body"`) {
		t.Fatalf("expected response body match in content, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"field":"cookies.session_token"`) {
		t.Fatalf("expected cookie match in content, got: %s", result.Content)
	}
}

type captureLLMClient struct {
	lastInput []*llmtypes.LLMMessage
}

func (c *captureLLMClient) Stream(ctx context.Context, input []*llmtypes.LLMMessage, opts ...llmtypes.LLMOption) (*llmtypes.LLMStreamReader, error) {
	c.lastInput = input
	sr, sw := llmtypes.NewLLMStreamReader(1)
	go func() {
		defer sw.Close()
		sw.Send(&llmtypes.LLMMessage{Role: llmtypes.RoleAssistant, Content: "captured"})
	}()
	return sr, nil
}

func TestExecuteTool_GetRequest_UsesDefaultSessionInsteadOfActiveSession(t *testing.T) {
	store := newTestStorage(t)

	active := &storage.Session{ID: "sess_active", Name: "Active"}
	mustCreateSession(t, store, active)
	view := &storage.Session{ID: "sess_view", Name: "View"}
	mustCreateSession(t, store, view)

	mustSaveRequest(t, store, &storage.Request{ID: "req_active", Method: "GET", Host: "active.example.com", Path: "/a"})
	mustSaveRequest(t, store, &storage.Request{ID: "req_view", SessionID: view.ID, Method: "GET", Host: "view.example.com", Path: "/v"})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolGetRequest, `{"request_id":"req_view"}`, view.ID)
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if !strings.Contains(result.Content, `"session_id":"sess_view"`) {
		t.Fatalf("expected request snapshot session_id sess_view, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"host":"view.example.com"`) {
		t.Fatalf("expected request snapshot host from viewed session, got: %s", result.Content)
	}
}

func TestExecuteTool_DiffRequests_UsesDefaultSessionInsteadOfActiveSession(t *testing.T) {
	store := newTestStorage(t)

	active := &storage.Session{ID: "sess_active", Name: "Active"}
	mustCreateSession(t, store, active)
	view := &storage.Session{ID: "sess_view", Name: "View"}
	mustCreateSession(t, store, view)

	mustSaveRequest(t, store, &storage.Request{ID: "req_a", SessionID: view.ID, Method: "GET", Host: "api.example.com", Path: "/a"})
	mustSaveRequest(t, store, &storage.Request{ID: "req_b", SessionID: view.ID, Method: "POST", Host: "api.example.com", Path: "/b"})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolDiffRequests, `{"request_id_a":"req_a","request_id_b":"req_b"}`, view.ID)
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if !strings.Contains(result.Summary, "Compared req_a and req_b") {
		t.Fatalf("unexpected diff summary: %s", result.Summary)
	}
}

func TestExecuteTool_GetRequest_UsesExplicitSessionArgument(t *testing.T) {
	store := newTestStorage(t)

	mustCreateSession(t, store, &storage.Session{ID: "sess_default", Name: "Default"})
	view := &storage.Session{ID: "sess_explicit", Name: "Explicit"}
	mustCreateSession(t, store, view)

	mustSaveRequest(t, store, &storage.Request{ID: "req_view", SessionID: view.ID, Method: "GET", Host: "view.example.com", Path: "/v"})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolGetRequest, `{"session_id":"sess_explicit","request_id":"req_view"}`, "sess_default")
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if !strings.Contains(result.Content, `"session_id":"sess_explicit"`) {
		t.Fatalf("expected explicit session_id in request snapshot, got: %s", result.Content)
	}
}

func TestExecuteTool_DiffRequests_UsesExplicitSessionArgument(t *testing.T) {
	store := newTestStorage(t)

	mustCreateSession(t, store, &storage.Session{ID: "sess_default", Name: "Default"})
	view := &storage.Session{ID: "sess_explicit", Name: "Explicit"}
	mustCreateSession(t, store, view)

	mustSaveRequest(t, store, &storage.Request{ID: "req_a", SessionID: view.ID, Method: "GET", Host: "api.example.com", Path: "/a"})
	mustSaveRequest(t, store, &storage.Request{ID: "req_b", SessionID: view.ID, Method: "POST", Host: "api.example.com", Path: "/b"})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolDiffRequests, `{"session_id":"sess_explicit","request_id_a":"req_a","request_id_b":"req_b"}`, "sess_default")
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if !strings.Contains(result.Summary, "Compared req_a and req_b") {
		t.Fatalf("unexpected diff summary: %s", result.Summary)
	}
}

func TestExecuteTool_SearchByHeader_UsesRequestSessionInsteadOfActiveSession(t *testing.T) {
	store := newTestStorage(t)

	active := &storage.Session{ID: "sess_active", Name: "Active"}
	mustCreateSession(t, store, active)
	view := &storage.Session{ID: "sess_view", Name: "View"}
	mustCreateSession(t, store, view)

	mustSaveRequest(t, store, &storage.Request{
		ID:        "req_view",
		SessionID: view.ID,
		Method:    "GET",
		Host:      "api.example.com",
		Path:      "/v",
		Headers:   storage.Headers{"Authorization": {"Bearer sess-view-token"}},
	})

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(store)
	result, err := agent.mcpMgr.Execute(context.Background(), agenttools.ToolSearchRequestsByHeader, `{"session_id":"sess_view","header_name":"Authorization","header_value":"sess-view-token"}`, "")
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	var payload struct {
		Results []struct {
			SessionID string `json:"session_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(result.Content), &payload); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(payload.Results) == 0 {
		t.Fatalf("expected at least one result, got: %s", result.Content)
	}
	if payload.Results[0].SessionID != view.ID {
		t.Fatalf("expected result session_id %q, got %q", view.ID, payload.Results[0].SessionID)
	}
}

func TestAgentAnalyze_TargetSnapshotUsesAnalysisSessionInsteadOfActiveSession(t *testing.T) {
	store := newTestStorage(t)

	active := &storage.Session{ID: "sess_active", Name: "Active"}
	mustCreateSession(t, store, active)
	view := &storage.Session{ID: "sess_view", Name: "View"}
	mustCreateSession(t, store, view)
	mustSaveRequest(t, store, &storage.Request{ID: "req_view", SessionID: view.ID, Method: "GET", Host: "api.example.com", Path: "/target"})

	llm := &captureLLMClient{}
	agent := NewAgent(llm)
	agent.SetStore(store)
	_, err := agent.Analyze(context.Background(), &AgentRequest{
		RequestID: "req_view",
		SessionID: view.ID,
		Query:     "inspect target",
		Model:     "mock-model",
	}, nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	joined := joinMessageContents(llm.lastInput)
	if !strings.Contains(joined, `"session_id":"sess_view"`) {
		t.Fatalf("expected initial prompt snapshot to contain viewed session id, got: %s", joined)
	}
	if strings.Contains(joined, `"session_id":"sess_active"`) {
		t.Fatalf("expected initial prompt snapshot not to leak active session id, got: %s", joined)
	}
}

func TestAgentAnalyze_TargetSnapshotUsesRequestSessionWhenSessionOmitted(t *testing.T) {
	store := newTestStorage(t)

	active := &storage.Session{ID: "sess_active", Name: "Active"}
	mustCreateSession(t, store, active)
	view := &storage.Session{ID: "sess_view", Name: "View"}
	mustCreateSession(t, store, view)
	mustSaveRequest(t, store, &storage.Request{ID: "req_view", SessionID: view.ID, Method: "GET", Host: "api.example.com", Path: "/target"})

	llm := &captureLLMClient{}
	agent := NewAgent(llm)
	agent.SetStore(store)
	result, err := agent.Analyze(context.Background(), &AgentRequest{
		RequestID: "req_view",
		Query:     "inspect target",
		Model:     "mock-model",
	}, nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if result.SessionID != view.ID {
		t.Fatalf("expected result session_id %q, got %q", view.ID, result.SessionID)
	}

	joined := joinMessageContents(llm.lastInput)
	if !strings.Contains(joined, `"session_id":"sess_view"`) {
		t.Fatalf("expected initial prompt snapshot to contain request session id, got: %s", joined)
	}
	if strings.Contains(joined, `"session_id":"sess_active"`) {
		t.Fatalf("expected initial prompt snapshot not to leak active session id, got: %s", joined)
	}
}

func TestAgentAnalyze_UsesSingleADKRuntime(t *testing.T) {
	newTestStorage(t)

	agent := NewAgent(&stubLLMClient{})
	agent.SetStore(storage.Default)
	req := &AgentRequest{
		SessionID: "sess_runtime",
		Query:     "Summarize the current capture context",
		Model:     "mock-model",
	}

	var events []AgentEvent
	result, err := agent.Analyze(context.Background(), req, func(event AgentEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if result == nil {
		t.Fatal("Analyze returned nil result")
	}
	if result.FinalAnswer == "" {
		t.Fatal("Analyze returned empty final answer")
	}
	if result.StoppedEarly {
		t.Fatalf("expected normal completion, got stopped early with reason %q", result.StopReason)
	}
	if result.DepthUsed == 0 {
		t.Fatalf("expected non-zero depth_used, got %d", result.DepthUsed)
	}
	if len(events) < 1 {
		t.Fatalf("expected at least final event, got %d", len(events))
	}
	if events[0].Type == "start" {
		t.Fatalf("first event type = start, want runtime event without synthetic start")
	}
	if events[len(events)-1].Type != "final" {
		t.Fatalf("last event type = %q, want final", events[len(events)-1].Type)
	}
	if events[len(events)-1].Content != result.FinalAnswer {
		t.Fatalf("final event content mismatch: got %q want %q", events[len(events)-1].Content, result.FinalAnswer)
	}
}

func TestAgentAnalyze_LoadsStorageBackedChatHistoryBeforePrompt(t *testing.T) {
	store := newTestStorage(t)

	session := &storage.Session{ID: "sess_history", Name: "History"}
	mustCreateSession(t, store, session)
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: session.ID, Role: "user", Content: "previous user question"}); err != nil {
		t.Fatalf("SaveChatMessage user failed: %v", err)
	}
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: session.ID, Role: "assistant", Content: "previous assistant answer"}); err != nil {
		t.Fatalf("SaveChatMessage assistant failed: %v", err)
	}

	llm := &captureLLMClient{}
	agent := NewAgent(llm)
	agent.SetStore(store)
	_, err := agent.Analyze(context.Background(), &AgentRequest{
		SessionID: session.ID,
		Query:     "new question",
		Model:     "mock-model",
	}, nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(llm.lastInput) < 4 {
		t.Fatalf("expected system prompt + 2 history messages + current prompt, got %d messages", len(llm.lastInput))
	}
	if llm.lastInput[1].Role != llmtypes.RoleUser || llm.lastInput[1].Content != "previous user question" {
		t.Fatalf("unexpected first history message: %+v", llm.lastInput[1])
	}
	if llm.lastInput[2].Role != llmtypes.RoleAssistant || llm.lastInput[2].Content != "previous assistant answer" {
		t.Fatalf("unexpected second history message: %+v", llm.lastInput[2])
	}
	if !strings.Contains(llm.lastInput[len(llm.lastInput)-1].Content, "new question") {
		t.Fatalf("expected latest prompt to contain current question, got %q", llm.lastInput[len(llm.lastInput)-1].Content)
	}
}

func TestAgentAnalyze_IgnoresPrivilegedStoredChatRoles(t *testing.T) {
	store := newTestStorage(t)

	session := &storage.Session{ID: "sess_history_roles", Name: "History Roles"}
	mustCreateSession(t, store, session)
	for _, msg := range []*storage.ChatMessage{
		{SessionID: session.ID, Role: "user", Content: "previous user question"},
		{SessionID: session.ID, Role: "system", Content: "system override"},
		{SessionID: session.ID, Role: "tool", Content: "tool output"},
		{SessionID: session.ID, Role: "assistant", Content: "previous assistant answer"},
	} {
		if err := store.SaveChatMessage(msg); err != nil {
			t.Fatalf("SaveChatMessage %q failed: %v", msg.Role, err)
		}
	}

	llm := &captureLLMClient{}
	agent := NewAgent(llm)
	agent.SetStore(store)
	_, err := agent.Analyze(context.Background(), &AgentRequest{
		SessionID: session.ID,
		Query:     "new question",
		Model:     "mock-model",
	}, nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	joined := joinMessageContents(llm.lastInput)
	if strings.Contains(joined, "system override") {
		t.Fatalf("expected persisted system history to be ignored, got: %s", joined)
	}
	if strings.Contains(joined, "tool output") {
		t.Fatalf("expected persisted tool history to be ignored, got: %s", joined)
	}
	if !strings.Contains(joined, "previous user question") {
		t.Fatalf("expected user history to remain, got: %s", joined)
	}
	if !strings.Contains(joined, "previous assistant answer") {
		t.Fatalf("expected assistant history to remain, got: %s", joined)
	}
}

func TestAgentAnalyze_PreservesAllChatHistory(t *testing.T) {
	store := newTestStorage(t)

	session := &storage.Session{ID: "sess_history_full", Name: "History Full"}
	mustCreateSession(t, store, session)
	for i := 1; i <= 15; i++ {
		if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: session.ID, Role: "user", Content: fmt.Sprintf("history-%02d", i)}); err != nil {
			t.Fatalf("SaveChatMessage %d failed: %v", i, err)
		}
	}

	llm := &captureLLMClient{}
	agent := NewAgent(llm)
	agent.SetStore(store)
	_, err := agent.Analyze(context.Background(), &AgentRequest{
		SessionID: session.ID,
		Query:     "new question",
		Model:     "mock-model",
	}, nil)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if got := len(llm.lastInput); got != 17 {
		t.Fatalf("expected system prompt + 15 history messages + current prompt, got %d", got)
	}
	if llm.lastInput[1].Content != "history-01" {
		t.Fatalf("expected oldest history message history-01, got %q", llm.lastInput[1].Content)
	}
	if llm.lastInput[15].Content != "history-15" {
		t.Fatalf("expected newest history message history-15, got %q", llm.lastInput[15].Content)
	}
	if !strings.Contains(llm.lastInput[len(llm.lastInput)-1].Content, "new question") {
		t.Fatalf("expected latest prompt to contain current question, got %q", llm.lastInput[len(llm.lastInput)-1].Content)
	}
}
