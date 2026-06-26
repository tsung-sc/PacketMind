package openai

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/respjson"
	"github.com/openai/openai-go/shared"
	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

type openAIToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

func defaultModelID(defaultModel string, options *llmtypes.LLMOptions) string {
	modelID := strings.TrimSpace(defaultModel)
	if options != nil && options.Model != nil && strings.TrimSpace(*options.Model) != "" {
		modelID = strings.TrimSpace(*options.Model)
	}
	return modelID
}

func buildProviderPayload(modelID string, messages []*llmtypes.LLMMessage, tools []*llmtypes.ToolDefinition, options *llmtypes.LLMOptions) openaisdk.ChatCompletionNewParams {
	params := openaisdk.ChatCompletionNewParams{
		Model:    modelID,
		Messages: make([]openaisdk.ChatCompletionMessageParamUnion, 0, len(messages)),
	}
	if options != nil {
		if options.MaxTokens != nil && *options.MaxTokens > 0 {
			params.MaxCompletionTokens = openaisdk.Int(int64(*options.MaxTokens))
		}
		if options.Temperature != nil && *options.Temperature != 0 {
			params.Temperature = openaisdk.Float(*options.Temperature)
		}
	}
	if len(tools) > 0 {
		openAITools, err := schemaToolsToOpenAIParams(tools)
		if err == nil && len(openAITools) > 0 {
			params.Tools = make([]openaisdk.ChatCompletionToolParam, 0, len(openAITools))
			params.ToolChoice = openaisdk.ChatCompletionToolChoiceOptionUnionParam{OfAuto: openaisdk.String(string(openaisdk.ChatCompletionToolChoiceOptionAutoAuto))}
			for _, tool := range openAITools {
				params.Tools = append(params.Tools, openaisdk.ChatCompletionToolParam{
					Type: "function",
					Function: shared.FunctionDefinitionParam{
						Name:        tool.Name,
						Description: openaisdk.String(tool.Description),
						Parameters:  tool.Parameters,
					},
				})
			}
		}
	}
	for _, msg := range messages {
		params.Messages = append(params.Messages, toOpenAIMessage(msg))
	}
	return params
}

func toOpenAIMessage(msg *llmtypes.LLMMessage) openaisdk.ChatCompletionMessageParamUnion {
	if msg == nil {
		return openaisdk.UserMessage("")
	}
	switch msg.Role {
	case llmtypes.RoleSystem:
		message := openaisdk.ChatCompletionSystemMessageParam{Role: "system"}
		message.Content.OfString = openaisdk.String(msg.Content)
		if msg.Name != "" {
			message.Name = openaisdk.String(msg.Name)
		}
		return openaisdk.ChatCompletionMessageParamUnion{OfSystem: &message}
	case llmtypes.RoleAssistant:
		message := openaisdk.ChatCompletionAssistantMessageParam{Role: "assistant"}
		if msg.Content != "" {
			message.Content.OfString = openaisdk.String(msg.Content)
		}
		if msg.Name != "" {
			message.Name = openaisdk.String(msg.Name)
		}
		if len(msg.ToolCalls) > 0 {
			message.ToolCalls = make([]openaisdk.ChatCompletionMessageToolCallParam, 0, len(msg.ToolCalls))
			for _, toolCall := range msg.ToolCalls {
				message.ToolCalls = append(message.ToolCalls, openaisdk.ChatCompletionMessageToolCallParam{
					ID:   toolCall.ID,
					Type: "function",
					Function: openaisdk.ChatCompletionMessageToolCallFunctionParam{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
				})
			}
		}
		return openaisdk.ChatCompletionMessageParamUnion{OfAssistant: &message}
	case llmtypes.RoleTool:
		message := openaisdk.ChatCompletionToolMessageParam{Role: "tool", ToolCallID: msg.ToolCallID}
		message.Content.OfString = openaisdk.String(msg.Content)
		return openaisdk.ChatCompletionMessageParamUnion{OfTool: &message}
	default:
		message := openaisdk.ChatCompletionUserMessageParam{Role: "user"}
		message.Content.OfString = openaisdk.String(msg.Content)
		if msg.Name != "" {
			message.Name = openaisdk.String(msg.Name)
		}
		return openaisdk.ChatCompletionMessageParamUnion{OfUser: &message}
	}
}

func toLLMMessage(completion openaisdk.ChatCompletion) *llmtypes.LLMMessage {
	msg := &llmtypes.LLMMessage{
		Role: llmtypes.RoleAssistant,
		Extra: map[string]any{
			"model": completion.Model,
		},
		ResponseMeta: &llmtypes.ResponseMeta{
			Usage: toLLMTokenUsage(completion.Usage),
		},
	}
	if len(completion.Choices) == 0 {
		if msg.ResponseMeta.Usage == nil {
			msg.ResponseMeta = nil
		}
		return msg
	}
	choice := completion.Choices[0]
	msg.ResponseMeta.FinishReason = choice.FinishReason
	msg.Content = contentWithReasoning(choice.Message.Content, extraFieldString(choice.Message.JSON.ExtraFields, "reasoning_content"))
	msg.ToolCalls = toLLMToolCalls(choice.Message.ToolCalls)
	if msg.ResponseMeta.FinishReason == "" && msg.ResponseMeta.Usage == nil {
		msg.ResponseMeta = nil
	}
	if len(msg.Extra) == 0 {
		msg.Extra = nil
	}
	return msg
}

func toLLMToolCalls(toolCalls []openaisdk.ChatCompletionMessageToolCall) []llmtypes.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	out := make([]llmtypes.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		out = append(out, llmtypes.ToolCall{
			ID:   toolCall.ID,
			Type: string(toolCall.Type),
			Function: llmtypes.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})
	}
	return out
}

func toLLMTokenUsage(usage openaisdk.CompletionUsage) *llmtypes.TokenUsage {
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 && usage.PromptTokensDetails.CachedTokens == 0 {
		return nil
	}
	return &llmtypes.TokenUsage{
		PromptTokens:       int(usage.PromptTokens),
		CompletionTokens:   int(usage.CompletionTokens),
		TotalTokens:        int(usage.TotalTokens),
		CachedPromptTokens: int(usage.PromptTokensDetails.CachedTokens),
		PromptTokenDetails: llmtypes.PromptTokenDetails{CachedTokens: int(usage.PromptTokensDetails.CachedTokens)},
	}
}

func schemaToolsToOpenAIParams(tools []*llmtypes.ToolDefinition) ([]openAIToolDefinition, error) {
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]openAIToolDefinition, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		params := toolJSONSchemaFromExtra(tool)
		if params == nil && tool.ParamsOneOf != nil {
			jsonSchema, err := tool.ParamsOneOf.ToJSONSchema()
			if err != nil {
				return nil, err
			}
			if jsonSchema != nil {
				raw, err := json.Marshal(jsonSchema)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(raw, &params); err != nil {
					return nil, err
				}
			}
		}
		out = append(out, openAIToolDefinition{Name: tool.Name, Description: tool.Desc, Parameters: params})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func collectProviderModelTools(extra []*llmtypes.ToolDefinition) []*llmtypes.ToolDefinition {
	return cloneToolDefinitions(extra)
}

func cloneToolDefinitions(tools []*llmtypes.ToolDefinition) []*llmtypes.ToolDefinition {
	if len(tools) == 0 {
		return nil
	}
	out := make([]*llmtypes.ToolDefinition, len(tools))
	copy(out, tools)
	return out
}

func toolJSONSchemaFromExtra(tool *llmtypes.ToolDefinition) map[string]any {
	if tool == nil || len(tool.Extra) == 0 {
		return nil
	}
	raw, _ := tool.Extra["json_schema"].(map[string]any)
	return cloneJSONObject(raw)
}

func cloneJSONObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var clone map[string]any
	if err := json.Unmarshal(raw, &clone); err != nil {
		return nil
	}
	return clone
}

func decodeMessageContent(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var builder strings.Builder
		for _, part := range parts {
			if part.Type == "text" {
				builder.WriteString(part.Text)
			}
		}
		return builder.String()
	}
	return trimmed
}

func contentWithReasoning(content, reasoning string) string {
	reasoning = strings.TrimSpace(reasoning)
	content = strings.TrimSpace(content)
	if reasoning == "" {
		return content
	}
	if content == "" {
		return reasoning
	}
	return reasoning + "\n\n" + content
}

func extraFieldString(fields map[string]respjson.Field, key string) string {
	if len(fields) == 0 {
		return ""
	}
	value, ok := fields[key]
	if !ok {
		return ""
	}
	return decodeMessageContent(json.RawMessage(value.Raw()))
}

func wrapOpenAIError(err error, providerName, operation string) error {
	if err == nil {
		return nil
	}
	var apiErr *openaisdk.Error
	if !errors.As(err, &apiErr) || apiErr == nil {
		return llmcore.WrapError(err, llmcore.CategoryFatal, providerName+" "+operation+" failed")
	}
	if apiErr.StatusCode != 0 {
		body := ""
		status := ""
		if apiErr.Response != nil {
			body = strings.TrimSpace(string(apiErr.DumpResponse(true)))
			status = apiErr.Response.Status
		}
		return &llmcore.ProviderStatusError{MessagePrefix: providerName + " " + operation, StatusCode: apiErr.StatusCode, Status: status, Body: body}
	}
	if apiErr.Code != "" || apiErr.Message != "" {
		return &llmcore.ProviderAPIError{MessagePrefix: providerName + " " + operation, Code: apiErr.Code, Message: apiErr.Message}
	}
	return llmcore.WrapError(err, llmcore.CategoryFatal, providerName+" "+operation+" failed")
}
