package bindings

import (
	"testing"

	"github.com/packetmind/packetmind/internal/agent"
	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/storage"
)

func TestListModels_UsesProviderDisplayName(t *testing.T) {
	config.DefaultModelsStore = config.NewModelsStore(t.TempDir(), &config.ModelsConfig{
		Providers: []config.ProviderConfig{
			{ID: "ark", Name: "Ark", APIType: "openai-compatible", BaseURL: "https://ark.example/v1", Models: map[string]config.ModelConfig{
				"glm-test": {Name: "GLM Test", Limit: &config.ModelLimit{Output: 4096}},
			}},
		},
		DefaultModel:   "glm-test",
		ActiveProvider: "ark",
		ActiveModel:    "glm-test",
	})

	resp := NewAgentAPI().ListModels()
	if resp.Code != 0 {
		t.Fatalf("ListModels code = %d, want 0", resp.Code)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("ListModels data type = %T, want map[string]interface{}", resp.Data)
	}
	groups, ok := data["grouped"].([]map[string]interface{})
	if !ok || len(groups) != 1 {
		t.Fatalf("ListModels grouped = %#v, want one group", data["grouped"])
	}
	if groups[0]["provider"] != "ark" || groups[0]["provider_name"] != "Ark" {
		t.Fatalf("unexpected provider group: %#v", groups[0])
	}
}

func TestGetChatHistory_ReturnsEmptyStorageHistory(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	config.DefaultModelsStore = config.NewModelsStore("./configs", &config.ModelsConfig{})

	api := NewAgentAPI()
	resp := api.GetChatHistory("sess_missing")
	if resp.Code != 0 {
		t.Fatalf("GetChatHistory code = %d, want 0", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("GetChatHistory data type = %T, want map[string]interface{}", resp.Data)
	}
	messages, ok := data["messages"].([]ChatMessageDTO)
	if !ok {
		t.Fatalf("GetChatHistory messages type = %T, want []ChatMessageDTO", data["messages"])
	}
	if len(messages) != 0 {
		t.Fatalf("GetChatHistory messages len = %d, want 0", len(messages))
	}
}

func TestGetChatHistory_ReadsFromStorage(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	config.DefaultModelsStore = config.NewModelsStore("./configs", &config.ModelsConfig{})

	sess := &storage.Session{Name: "chat"}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: sess.ID, Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("SaveChatMessage failed: %v", err)
	}

	api := NewAgentAPI()
	resp := api.GetChatHistory(sess.ID)
	if resp.Code != 0 {
		t.Fatalf("GetChatHistory code = %d, want 0", resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("GetChatHistory data type = %T, want map[string]interface{}", resp.Data)
	}
	messages, ok := data["messages"].([]ChatMessageDTO)
	if !ok {
		t.Fatalf("GetChatHistory messages type = %T, want []ChatMessageDTO", data["messages"])
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("unexpected messages: %+v", messages)
	}
}

func TestClearSessionMemory_DeletesStorageBackedChatHistory(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store

	sess := &storage.Session{Name: "chat-clear"}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: sess.ID, Role: "user", Content: "bye"}); err != nil {
		t.Fatalf("SaveChatMessage failed: %v", err)
	}

	api := NewAgentAPI()
	resp := api.ClearSessionMemory(sess.ID)
	if resp.Code != 0 {
		t.Fatalf("ClearSessionMemory code = %d, want 0", resp.Code)
	}

	messages, err := store.ListChatMessages(sess.ID)
	if err != nil {
		t.Fatalf("ListChatMessages failed: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected storage-backed chat history cleared, got %+v", messages)
	}
	if resp.Message != "session chat history cleared" {
		t.Fatalf("unexpected ClearSessionMemory message: %q", resp.Message)
	}
}

func TestPersistMessages_SavesUserAndAssistant(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store

	sess := &storage.Session{Name: "chat-record"}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	api := NewAgentAPI()
	api.persistUserMessage(sess.ID, "question")
	api.persistAssistantMessage(sess.ID, &agent.AgentResult{SessionID: sess.ID, FinalAnswer: "answer"})

	stored, err := store.ListChatMessages(sess.ID)
	if err != nil {
		t.Fatalf("ListChatMessages failed: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored chat messages, got %d", len(stored))
	}
	if stored[0].Role != "user" || stored[1].Role != "assistant" {
		t.Fatalf("unexpected stored roles: %+v", stored)
	}
	if stored[0].Content != "question" || stored[1].Content != "answer" {
		t.Fatalf("unexpected stored content: %+v", stored)
	}
}

func TestGetSessionContext_UsesStorageBackedStats(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store
	config.DefaultModelsStore = config.NewModelsStore("./configs", &config.ModelsConfig{})

	sess := &storage.Session{Name: "chat-context"}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: sess.ID, Role: "user", Content: "hello world"}); err != nil {
		t.Fatalf("SaveChatMessage user failed: %v", err)
	}
	if err := store.SaveChatMessage(&storage.ChatMessage{SessionID: sess.ID, Role: "assistant", Content: "hi there"}); err != nil {
		t.Fatalf("SaveChatMessage assistant failed: %v", err)
	}

	api := NewAgentAPI()
	stats := api.GetSessionContext(sess.ID)
	if stats.Code != 0 {
		t.Fatalf("GetSessionContext code = %d, want 0", stats.Code)
	}
	if !stats.HasHistory {
		t.Fatal("expected HasHistory to be true")
	}
	if stats.MessageCount != 2 {
		t.Fatalf("MessageCount = %d, want 2", stats.MessageCount)
	}
	if stats.EstimatedTokens <= 0 {
		t.Fatalf("EstimatedTokens = %d, want > 0", stats.EstimatedTokens)
	}
}

func TestRegisterSessionDeleteHook_AllowsInternalCleanupWiring(t *testing.T) {
	store, err := storage.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	storage.Default = store

	api := NewSessionAPI()
	called := ""
	RegisterSessionDeleteHook(api, func(sessionID string) {
		called = sessionID
	})

	sess := &storage.Session{Name: "to-delete"}
	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	resp := api.DeleteSession(sess.ID)
	if resp.Code != 0 {
		t.Fatalf("DeleteSession code = %d, want 0", resp.Code)
	}
	if called != sess.ID {
		t.Fatalf("delete hook called with %q, want %q", called, sess.ID)
	}
}
