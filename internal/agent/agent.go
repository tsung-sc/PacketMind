package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/packetmind/packetmind/internal/agent/llmtypes"
	"github.com/packetmind/packetmind/internal/agent/mcp"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/agent/tools/builtin"
	"github.com/packetmind/packetmind/internal/storage"
)

type Agent struct {
	llm    llmtypes.LLMClient
	store  *storage.Storage
	mcpMgr *mcp.Manager
}

// NewAgent creates a new request exploration agent.
func NewAgent(llm llmtypes.LLMClient) *Agent {
	return &Agent{
		llm: llm,
	}
}

// NewAgentFromProvider creates an Agent backed by the selected provider client.
func NewAgentFromProvider(provider, apiKey, baseURL string) (*Agent, error) {
	factory, err := GetProviderFactory(provider)
	if err != nil {
		return nil, err
	}
	return factory.NewAgent(apiKey, baseURL)
}

func (a *Agent) SetMCPManager(manager *mcp.Manager) {
	if manager == nil || a.mcpMgr == nil {
		return
	}
	a.mcpMgr.MergeFrom(manager)
	ctx := context.Background()
	a.mcpMgr.RegisterAll(ctx)
}

func (a *Agent) SetStore(store *storage.Storage) {
	a.store = store
	if a.mcpMgr != nil {
		return
	}

	a.mcpMgr = mcp.NewManager()

	builtinSrv := mcp.NewBuiltinServer(store, a.mcpMgr)
	ctx := context.Background()
	builtinClient, err := mcp.NewInProcessClient(ctx, builtinSrv)
	if err != nil {
		fmt.Printf("[Agent] Failed to create builtin MCP client: %v\n", err)
		return
	}

	if err := a.mcpMgr.AddAdapterWithPrefix("builtin", "", builtinClient); err != nil {
		fmt.Printf("[Agent] Failed to add builtin adapter: %v\n", err)
		return
	}

	total, errs := a.mcpMgr.RegisterAll(ctx)
	if len(errs) > 0 {
		for name, e := range errs {
			fmt.Printf("[Agent] Builtin tool %s failed: %v\n", name, e)
		}
	}
	fmt.Printf("[Agent] Registered %d builtin tools\n", total)
}

func (a *Agent) Analyze(ctx context.Context, req *AgentRequest, onEvent AgentEventHandler) (*AgentResult, error) {
	if req == nil {
		return nil, fmt.Errorf("agent request is nil")
	}
	if a == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if a.store == nil {
		return nil, fmt.Errorf("agent storage is nil")
	}
	if a.llm == nil {
		return nil, fmt.Errorf("agent llm client is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, fmt.Errorf("model is required")
	}

	hasTarget := strings.TrimSpace(req.RequestID) != ""
	var targetReq *storage.Request
	sessionID := strings.TrimSpace(req.SessionID)
	if hasTarget {
		var err error
		if sessionID == "" {
			targetReq, err = a.store.GetRequestByID(req.RequestID)
			if err != nil {
				return nil, fmt.Errorf("failed to get target request: %w", err)
			}
			sessionID = strings.TrimSpace(targetReq.SessionID)
		} else {
			targetReq, err = a.store.GetRequest(sessionID, req.RequestID)
			if err != nil {
				return nil, fmt.Errorf("failed to get target request: %w", err)
			}
		}
		if sessionID == "" && targetReq != nil {
			sessionID = strings.TrimSpace(targetReq.SessionID)
		}
	} else if sessionID == "" {
		if sess, err := a.store.GetActiveSession(); err == nil && sess != nil {
			sessionID = sess.ID
		}
	}

	result := &AgentResult{
		RequestID:  req.RequestID,
		SessionID:  sessionID,
		Model:      req.Model,
		TraceChain: make([]AgentTraceStep, 0, 32),
	}

	var accumulatedUsage *llmtypes.TokenUsage

	builder := NewSystemPromptBuilder().WithRole(DefaultAgentRole())
	systemPrompt := builder.Build()

	initialPrompt := a.buildInitialPrompt(req, targetReq, sessionID)

	runtimeInput, historyErr := a.loadChatHistoryMessages(sessionID)
	if historyErr != nil {
		return nil, fmt.Errorf("failed to load chat history: %w", historyErr)
	}
	runtimeInput = append(runtimeInput, &llmtypes.LLMMessage{Role: llmtypes.RoleUser, Content: initialPrompt})

	runtimeHandler := func(event AgentEvent) {
		switch event.Type {
		case "thought":
			a.appendTrace(result, AgentTraceStep{
				Depth:     event.Depth,
				Type:      "thought",
				Content:   event.Content,
				CreatedAt: event.CreatedAt,
			})
		case "action":
			a.appendTrace(result, AgentTraceStep{
				Depth:         event.Depth,
				Type:          "action",
				ToolName:      event.ToolName,
				ToolArguments: event.Arguments,
				CreatedAt:     event.CreatedAt,
			})
		case "observation":
			a.appendTrace(result, AgentTraceStep{
				Depth:      event.Depth,
				Type:       "observation",
				Content:    event.Content,
				ToolName:   event.ToolName,
				ToolResult: event.Result,
				RequestIDs: append([]string(nil), event.RequestIDs...),
				CreatedAt:  event.CreatedAt,
			})
		}
		if event.Type != "final" {
			a.emit(onEvent, event)
		}
	}

	var (
		runtimeResult *agentruntime.RuntimeResult
		err           error
	)

	runner := a.tryNewRunner(req.Model, sessionID, req.InterventionProvider)
	if runner == nil {
		return nil, fmt.Errorf("failed to create runtime runner")
	}
	runner.SetSystemPrompt(systemPrompt)
	runtimeResult, err = runner.Run(ctx, runtimeInput, runtimeHandler)
	if err != nil {
		return nil, err
	}

	accumulatedUsage = agentruntime.AddTokenUsage(accumulatedUsage, runtimeResult.TokenUsage)
	result.TokenUsage = agentruntime.CloneTokenUsage(accumulatedUsage)

	if runtimeResult.StoppedEarly {
		message := "Agent stopped early"
		if runtimeResult.StopReason == "empty_final_answer" {
			message = "Model did not return a final answer"
		}
		return a.stopEarly(result, runtimeResult.Iterations, runtimeResult.ToolCalls, onEvent, runtimeResult.StopReason, message)
	}

	result.FinalAnswer = runtimeResult.FinalAnswer
	result.DepthUsed = runtimeResult.Iterations
	result.ToolCalls = runtimeResult.ToolCalls

	a.appendTrace(result, AgentTraceStep{
		Depth:     runtimeResult.Iterations,
		Type:      "final",
		Content:   runtimeResult.FinalAnswer,
		CreatedAt: time.Now(),
	})
	a.emit(onEvent, AgentEvent{
		Depth:     runtimeResult.Iterations,
		Type:      "final",
		Content:   runtimeResult.FinalAnswer,
		ToolCalls: runtimeResult.ToolCalls,
		CreatedAt: time.Now(),
	})

	return result, nil
}

func (a *Agent) tryNewRunner(modelID, sessionID string, intervention agentruntime.InterventionFunc) *agentruntime.Runner {
	if a == nil || a.llm == nil || a.mcpMgr == nil {
		return nil
	}
	return agentruntime.NewRunner(
		a.llm,
		a.mcpMgr.Schemas(),
		agentruntime.WithModel(modelID),
		agentruntime.WithSessionID(sessionID),
		agentruntime.WithExecuteTool(a.mcpMgr.SafeExecute),
		agentruntime.WithIntervention(intervention),
	)
}

func (a *Agent) buildInitialPrompt(req *AgentRequest, target *storage.Request, sessionID string) string {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		query = "Trace key parameter origins, propagation chains, and verifiable clues for the current request."
	}

	if target == nil {
		if strings.TrimSpace(sessionID) == "" {
			return "Analysis target: " + query
		}
		return fmt.Sprintf("Analysis target: %s\n\nNo target request is set. Current session_id: %s", query, sessionID)
	}

	snapshotSessionID := sessionID
	if strings.TrimSpace(target.SessionID) != "" {
		snapshotSessionID = target.SessionID
	}

	return fmt.Sprintf("Analysis target: %s\n\nStarting context:\n- request_id: %s\n- session_id: %s\n- host: %s\n- path: %s\n\nInitial request snapshot (JSON):\n%s", query, target.ID, sessionID, target.Host, target.Path, builtin.MustMarshalJSON(builtin.MakeRequestSnapshot(target, snapshotSessionID)))
}

func (a *Agent) loadChatHistoryMessages(sessionID string) ([]*llmtypes.LLMMessage, error) {
	if a == nil || a.store == nil {
		return nil, nil
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, nil
	}

	history, err := a.store.ListChatMessages(sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]*llmtypes.LLMMessage, 0, len(history))
	for _, msg := range history {
		if msg == nil {
			continue
		}
		role, ok := storageChatRoleToLLMRole(msg.Role)
		if !ok {
			continue
		}
		messages = append(messages, &llmtypes.LLMMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
	return messages, nil
}

func storageChatRoleToLLMRole(role string) (llmtypes.Role, bool) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case string(llmtypes.RoleUser):
		return llmtypes.RoleUser, true
	case string(llmtypes.RoleAssistant):
		return llmtypes.RoleAssistant, true
	default:
		return "", false
	}
}

func (a *Agent) stopEarly(result *AgentResult, depth, toolCalls int, onEvent AgentEventHandler, reason, message string) (*AgentResult, error) {
	finalAnswer := a.buildFallbackAnswer(result.TraceChain, message)
	result.FinalAnswer = finalAnswer
	result.DepthUsed = depth
	result.ToolCalls = toolCalls
	result.StoppedEarly = true
	result.StopReason = reason

	a.appendTrace(result, AgentTraceStep{
		Depth:     depth,
		Type:      "warning",
		Content:   message,
		CreatedAt: time.Now(),
	})
	a.appendTrace(result, AgentTraceStep{
		Depth:     depth,
		Type:      "final",
		Content:   finalAnswer,
		CreatedAt: time.Now(),
	})

	a.emit(onEvent, AgentEvent{
		Depth:     depth,
		Type:      "warning",
		Content:   message,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
	})
	a.emit(onEvent, AgentEvent{
		Depth:     depth,
		Type:      "final",
		Content:   finalAnswer,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
	})

	return result, nil
}

func (a *Agent) appendTrace(result *AgentResult, step AgentTraceStep) {
	result.TraceChain = append(result.TraceChain, step)
}

func (a *Agent) emit(onEvent AgentEventHandler, event AgentEvent) {
	if onEvent != nil {
		onEvent(event)
	}
}

func (a *Agent) buildFallbackAnswer(trace []AgentTraceStep, reason string) string {
	evidence := make([]string, 0, 4)
	requestIDs := make(map[string]struct{})
	for i := len(trace) - 1; i >= 0 && len(evidence) < 3; i-- {
		step := trace[i]
		if step.Type == "observation" && strings.TrimSpace(step.Content) != "" {
			evidence = append(evidence, "- "+step.Content)
		}
		for _, requestID := range step.RequestIDs {
			if requestID != "" {
				requestIDs[requestID] = struct{}{}
			}
		}
	}

	idList := make([]string, 0, len(requestIDs))
	for requestID := range requestIDs {
		idList = append(idList, requestID)
	}
	sort.Strings(idList)

	if len(evidence) == 0 {
		evidence = append(evidence, "- Insufficient evidence collected so far; manual confirmation needed")
	}

	relatedRequests := "-"
	if len(idList) > 0 {
		relatedRequests = strings.Join(idList, ", ")
	}

	return ExecutePrompt("fallback_answer", map[string]any{
		"Reason":          reason,
		"Evidence":        strings.Join(evidence, "\n"),
		"RelatedRequests": relatedRequests,
	})
}
