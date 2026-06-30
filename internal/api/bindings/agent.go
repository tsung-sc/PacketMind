package bindings

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/packetmind/packetmind/internal/agent"
	"github.com/packetmind/packetmind/internal/agent/llmcore"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/appctx"
	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/storage"
)

// analysisContext tracks an in-flight analysis with its session association.
type analysisIntervention struct {
	ID      string
	Content string
}

type analysisContext struct {
	cancel    context.CancelFunc
	sessionID string
	mu        sync.Mutex
	steers    []analysisIntervention
}

// AgentAPI 提供 AI 分析相关的前端绑定。
type AgentAPI struct {
	activeCtxs   map[string]*analysisContext
	activeCtxsMu sync.RWMutex
}

// AnalyzeRequest AI 分析请求。
type AnalyzeRequest struct {
	RequestID string `json:"request_id"`
	SessionID string `json:"session_id"`
	Query     string `json:"query"`
	ModelID   string `json:"model_id"`
}

type InterventionRequest struct {
	AnalysisID string `json:"analysis_id"`
	SessionID  string `json:"session_id"`
	MessageID  string `json:"message_id"`
	Message    string `json:"message"`
	Mode       string `json:"mode"`
}

// AnalyzeResponse AI 分析响应。
type AnalyzeResponse struct {
	Code       int    `json:"code"`
	Message    string `json:"message,omitempty"`
	AnalysisID string `json:"analysis_id,omitempty"`
	ModelID    string `json:"model_id,omitempty"`
}

// NewAgentAPI 创建 AI 绑定。
func NewAgentAPI() *AgentAPI {
	return &AgentAPI{
		activeCtxs: make(map[string]*analysisContext),
	}
}

// Analyze 异步执行 AI 分析。
func (a *AgentAPI) Analyze(req AnalyzeRequest) AnalyzeResponse {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		settings := config.DefaultModelsStore.GetSettings()
		modelID = settings.Model
	}

	settings := config.DefaultModelsStore.GetSettings()
	providerID := strings.TrimSpace(settings.Provider)
	apiType := strings.TrimSpace(config.DefaultModelsStore.APIType(providerID))
	apiKey := config.DefaultModelsStore.APIKey(providerID)
	baseURL := config.DefaultModelsStore.BaseURL(providerID)

	analysisID := "ana_" + uuid.New().String()

	go a.runAgentAnalysis(analysisID, &req, modelID, apiType, apiKey, baseURL)

	return AnalyzeResponse{
		Code:       0,
		AnalysisID: analysisID,
		ModelID:    modelID,
	}
}
func (a *AgentAPI) runAgentAnalysis(analysisID string, req *AnalyzeRequest, modelID, apiType, apiKey, baseURL string) {
	sessionID := strings.TrimSpace(req.SessionID)

	defer func() {
		if r := recover(); r != nil {
			a.emitError(analysisID, sessionID, fmt.Errorf("Agent analysis panic: %v", r))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	a.registerContext(analysisID, cancel, sessionID)
	defer a.unregisterContext(analysisID)
	defer cancel()

	baseAgent, err := agent.NewAgentFromProvider(apiType, apiKey, baseURL)
	if err != nil {
		a.emitError(analysisID, sessionID, fmt.Errorf("Failed to create agent: %w", err))
		return
	}
	baseAgent.SetStore(storage.Default)
	if agent.DefaultMCPManager != nil {
		baseAgent.SetMCPManager(agent.DefaultMCPManager)
	}

	agentReq := &agent.AgentRequest{
		RequestID: req.RequestID,
		SessionID: sessionID,
		Query:     strings.TrimSpace(req.Query),
		Model:     modelID,
		InterventionProvider: func() []agentruntime.Intervention {
			return a.takeSteers(analysisID)
		},
	}

	a.persistUserMessage(sessionID, agentReq.Query)

	result, err := baseAgent.Analyze(ctx, agentReq, func(event agent.AgentEvent) {
		a.emitAgentEvent(analysisID, sessionID, event)
	})
	if err != nil {
		a.emitError(analysisID, sessionID, err)
		return
	}

	a.persistAssistantMessage(sessionID, result)
	a.emitAgentResult(analysisID, sessionID, result)
}

// SessionContextStats holds storage-backed chat context statistics for a
// capture session, suitable for frontend modal display.
type SessionContextStats struct {
	Code int `json:"code"`

	SessionID         string  `json:"session_id"`
	HasHistory        bool    `json:"has_history"`
	MessageCount      int     `json:"message_count"`
	EstimatedTokens   int     `json:"estimated_tokens"`
	ActiveModel       string  `json:"active_model"`
	ActiveProvider    string  `json:"active_provider"`
	ActiveMaxTokens   int     `json:"active_max_tokens"`
	ActiveTemperature float64 `json:"active_temperature"`
	AvailableModels   int     `json:"available_models"`
}

// GetSessionContext returns storage-backed session-context statistics.
// This is an exported Wails binding suitable for frontend modal display.
func (a *AgentAPI) GetSessionContext(sessionID string) SessionContextStats {
	sessionID = strings.TrimSpace(sessionID)

	stats := SessionContextStats{
		Code:      0,
		SessionID: sessionID,
	}

	if sessionID != "" && storage.Default != nil {
		messages, err := storage.Default.ListChatMessages(sessionID)
		if err == nil {
			stats.MessageCount = len(messages)
			stats.HasHistory = len(messages) > 0
			for _, msg := range messages {
				if msg != nil {
					stats.EstimatedTokens += estimateChatMessageTokens(msg.Content)
				}
			}
		}
	}

	settings := config.DefaultModelsStore.GetSettings()
	stats.ActiveModel = settings.Model
	stats.ActiveProvider = settings.Provider
	stats.ActiveMaxTokens = settings.MaxTokens
	stats.ActiveTemperature = settings.Temperature
	stats.AvailableModels = len(config.DefaultModelsStore.RuntimeModels())

	return stats
}

// ChatMessageDTO is a frontend-facing DTO for a single chat message.
// It avoids exposing time.Time directly to Wails bindings.
type ChatMessageDTO struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// GetChatHistory returns the persisted agent chat history for a capture session.
// The frontend calls this when switching sessions to restore the conversation.
func (a *AgentAPI) GetChatHistory(sessionID string) SessionResponse {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionResponse{Code: 40002, Message: "session_id is required"}
	}

	messages, err := storage.Default.ListChatMessages(sessionID)
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	items := make([]ChatMessageDTO, 0, len(messages))
	for _, m := range messages {
		items = append(items, ChatMessageDTO{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	return SessionResponse{Code: 0, Data: map[string]interface{}{"messages": items}}
}

// ClearSessionMemory clears persisted agent chat history for a session.
func (a *AgentAPI) ClearSessionMemory(sessionID string) SessionResponse {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionResponse{Code: 40002, Message: "session_id is required"}
	}

	if err := storage.Default.DeleteChatMessages(sessionID); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "session chat history cleared"}
}

// CancelAnalysis 取消分析。
func (a *AgentAPI) SendIntervention(req InterventionRequest) SessionResponse {
	analysisID := strings.TrimSpace(req.AnalysisID)
	message := strings.TrimSpace(req.Message)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "steer"
	}
	if analysisID == "" || message == "" {
		return SessionResponse{Code: 40002, Message: "analysis_id and message are required"}
	}
	if mode == "abort" {
		return a.CancelAnalysis(analysisID)
	}
	if mode != "steer" && mode != "queue" {
		return SessionResponse{Code: 40002, Message: "mode must be steer, queue, or abort"}
	}
	if mode == "queue" {
		return SessionResponse{Code: 40901, Message: "queue mode is handled by the frontend queue"}
	}

	a.activeCtxsMu.RLock()
	ac, exists := a.activeCtxs[analysisID]
	a.activeCtxsMu.RUnlock()
	if !exists {
		return SessionResponse{Code: 40001, Message: "Analysis not found or already completed"}
	}
	messageID := strings.TrimSpace(req.MessageID)
	if messageID == "" {
		messageID = "steer_" + uuid.New().String()
	}
	ac.mu.Lock()
	ac.steers = append(ac.steers, analysisIntervention{ID: messageID, Content: message})
	ac.mu.Unlock()
	if appctx.Ctx != nil {
		runtime.EventsEmit(appctx.Ctx, "agent:analysis", map[string]interface{}{
			"analysis_id": analysisID,
			"session_id":  ac.sessionID,
			"agent_event": map[string]interface{}{
				"type":       "thought",
				"content":    "User steering received. I will apply it at the next safe step.",
				"created_at": time.Now(),
			},
		})
	}
	return SessionResponse{Code: 0, Message: "Steering instruction queued for next step"}
}

// CancelAnalysis 取消分析。
func (a *AgentAPI) CancelAnalysis(analysisID string) SessionResponse {
	a.activeCtxsMu.Lock()
	ac, exists := a.activeCtxs[analysisID]
	if exists {
		delete(a.activeCtxs, analysisID)
	}
	a.activeCtxsMu.Unlock()

	if !exists {
		return SessionResponse{Code: 40001, Message: "Analysis not found or already completed"}
	}

	ac.cancel()

	if appctx.Ctx != nil {
		runtime.EventsEmit(appctx.Ctx, "agent:analysis", map[string]interface{}{
			"analysis_id": analysisID,
			"session_id":  ac.sessionID,
			"cancelled":   true,
			"done":        true,
		})
	}

	return SessionResponse{Code: 0, Message: "Analysis cancelled"}
}

// modelItem is a frontend-friendly model object constructed from the config map.
type modelItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens"`
}

// ListModels 列出当前模型配置（按 provider 分组）。
func (a *AgentAPI) ListModels() SessionResponse {
	modelsCfg := config.DefaultModelsStore.Snapshot()
	settings := config.DefaultModelsStore.GetSettings()

	var allModels []modelItem
	grouped := make([]map[string]interface{}, 0, len(modelsCfg.Providers))

	for _, provider := range modelsCfg.Providers {
		providerModels := make([]modelItem, 0, len(provider.Models))
		for id, m := range provider.Models {
			mi := modelItem{ID: id, Name: m.Name}
			if m.Limit != nil && m.Limit.Output > 0 {
				mi.MaxTokens = m.Limit.Output
			}
			providerModels = append(providerModels, mi)
			allModels = append(allModels, mi)
		}

		providerName := provider.Name
		providerIcon := ""
		if desc, ok := agent.GetProviderDescriptor(provider.APIType); ok {
			providerIcon = desc.Icon
			if providerName == "" {
				providerName = desc.Name
			}
		}
		if providerName == "" {
			providerName = provider.ID
		}

		grouped = append(grouped, map[string]interface{}{
			"provider":      provider.ID,
			"provider_name": providerName,
			"provider_icon": providerIcon,
			"models":        providerModels,
		})
	}

	return SessionResponse{
		Code: 0,
		Data: map[string]interface{}{
			"models":          allModels,
			"default_model":   modelsCfg.DefaultModel,
			"active_model":    settings.Model,
			"active_provider": settings.Provider,
			"grouped":         grouped,
		},
	}
}

// ListProviders returns all available providers with their configuration status.
func (a *AgentAPI) ListProviders() SessionResponse {
	providers := config.DefaultModelsStore.GetProviders()
	return SessionResponse{
		Code: 0,
		Data: map[string]interface{}{
			"providers": providers,
		},
	}
}

func (a *AgentAPI) emitAgentEvent(analysisID, sessionID string, event agent.AgentEvent) {
	if appctx.Ctx == nil {
		return
	}

	agentEvent := map[string]interface{}{
		"depth":           event.Depth,
		"type":            event.Type,
		"content":         event.Content,
		"tool_name":       event.ToolName,
		"tool_call_id":    event.ToolCallID,
		"arguments":       event.Arguments,
		"result":          event.Result,
		"request_ids":     event.RequestIDs,
		"tool_calls":      event.ToolCalls,
		"created_at":      event.CreatedAt.Format(time.RFC3339),
		"specialist_name": event.SpecialistName,
		"error_category":  event.ErrorCategory,
		"error_tool_name": event.ErrorToolName,
		"error_timeout":   event.ErrorTimeout,
		"error_recovered": event.ErrorRecovered,
		"retry_attempt":   event.RetryAttempt,
		"retry_max":       event.RetryMax,
		"intervention_id": event.InterventionID,
	}

	runtime.EventsEmit(appctx.Ctx, "agent:analysis", map[string]interface{}{
		"analysis_id": analysisID,
		"session_id":  sessionID,
		"agent_event": agentEvent,
		"done":        false,
	})
}

func (a *AgentAPI) emitAgentResult(analysisID, sessionID string, result *agent.AgentResult) {
	if appctx.Ctx == nil {
		return
	}
	if result == nil {
		a.emitError(analysisID, sessionID, fmt.Errorf("invalid agent result"))
		return
	}

	agentResult := map[string]interface{}{
		"final_answer":  result.FinalAnswer,
		"depth_used":    result.DepthUsed,
		"tool_calls":    result.ToolCalls,
		"stopped_early": result.StoppedEarly,
		"stop_reason":   result.StopReason,
	}
	if result.TokenUsage != nil {
		agentResult["token_usage"] = map[string]interface{}{
			"input_tokens":                result.TokenUsage.PromptTokens,
			"output_tokens":               result.TokenUsage.CompletionTokens,
			"cache_creation_input_tokens": result.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			"cache_read_input_tokens":     result.TokenUsage.PromptTokenDetails.CachedTokens,
			"total_tokens":                result.TokenUsage.TotalTokens,
		}
		agentResult["tokens_used"] = result.TokenUsage.TotalTokens
	} else {
		agentResult["tokens_used"] = 0
	}

	runtime.EventsEmit(appctx.Ctx, "agent:analysis", map[string]interface{}{
		"analysis_id":  analysisID,
		"session_id":   sessionID,
		"content":      result.FinalAnswer,
		"done":         true,
		"agent_result": agentResult,
	})
}

func (a *AgentAPI) persistUserMessage(sessionID string, query string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || storage.Default == nil {
		return
	}
	if q := strings.TrimSpace(query); q != "" {
		_ = storage.Default.SaveChatMessage(&storage.ChatMessage{
			SessionID: sessionID,
			Role:      "user",
			Content:   q,
		})
	}
}

func (a *AgentAPI) persistAssistantMessage(sessionID string, result *agent.AgentResult) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" && result != nil {
		sessionID = strings.TrimSpace(result.SessionID)
	}
	if sessionID == "" || storage.Default == nil || result == nil {
		return
	}
	if answer := strings.TrimSpace(result.FinalAnswer); answer != "" {
		_ = storage.Default.SaveChatMessage(&storage.ChatMessage{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   answer,
		})
	}
}

func estimateChatMessageTokens(content string) int {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0
	}
	return (len(content) + 3) / 4
}

func (a *AgentAPI) emitError(analysisID, sessionID string, err error) {
	if appctx.Ctx == nil {
		return
	}
	if err == nil {
		err = fmt.Errorf("unknown error")
	}

	errCategory := ""
	errToolName := ""
	errTimeout := ""
	errRecovered := false
	if agentErr := llmcore.AsAgentError(err); agentErr != nil {
		errCategory = string(agentErr.Category)
		errToolName = agentErr.ToolName
		if agentErr.Timeout > 0 {
			errTimeout = agentErr.Timeout.String()
		}
		errRecovered = agentErr.Recovered
	}

	runtime.EventsEmit(appctx.Ctx, "agent:analysis", map[string]interface{}{
		"analysis_id":     analysisID,
		"session_id":      sessionID,
		"error":           err.Error(),
		"error_category":  errCategory,
		"error_tool_name": errToolName,
		"error_timeout":   errTimeout,
		"error_recovered": errRecovered,
		"done":            true,
	})
}

func (a *AgentAPI) registerContext(analysisID string, cancel context.CancelFunc, sessionID string) {
	a.activeCtxsMu.Lock()
	a.activeCtxs[analysisID] = &analysisContext{cancel: cancel, sessionID: sessionID}
	a.activeCtxsMu.Unlock()
}

func (a *AgentAPI) takeSteers(analysisID string) []agentruntime.Intervention {
	a.activeCtxsMu.RLock()
	ac := a.activeCtxs[analysisID]
	a.activeCtxsMu.RUnlock()
	if ac == nil {
		return nil
	}
	ac.mu.Lock()
	defer ac.mu.Unlock()
	if len(ac.steers) == 0 {
		return nil
	}
	items := make([]agentruntime.Intervention, 0, len(ac.steers))
	for _, steer := range ac.steers {
		items = append(items, agentruntime.Intervention{ID: steer.ID, Content: steer.Content})
	}
	ac.steers = nil
	return items
}

func (a *AgentAPI) unregisterContext(analysisID string) {
	a.activeCtxsMu.Lock()
	delete(a.activeCtxs, analysisID)
	a.activeCtxsMu.Unlock()
}
