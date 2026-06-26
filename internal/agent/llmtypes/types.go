package llmtypes

import (
	"context"
	"io"
	"sort"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Index    int          `json:"index,omitempty"`
	Function FunctionCall `json:"function"`
}

type ChatMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ResponseMeta struct {
	Usage        *TokenUsage `json:"usage,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type LLMMessage struct {
	Role             Role              `json:"role"`
	Content          string            `json:"content,omitempty"`
	ToolCalls        []ToolCall        `json:"tool_calls,omitempty"`
	ToolCallID       string            `json:"tool_call_id,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	MultiContent     []ChatMessagePart `json:"multi_content,omitempty"`
	ResponseMeta     *ResponseMeta     `json:"response_meta,omitempty"`
	Extra            map[string]any    `json:"extra,omitempty"`
	Name             string            `json:"name,omitempty"`
}

type DataType string

const (
	TypeString DataType = "string"
	TypeArray  DataType = "array"
	TypeObject DataType = "object"
)

type ToolParameter struct {
	Type      DataType                   `json:"type"`
	Desc      string                     `json:"description,omitempty"`
	Enum      []string                   `json:"enum,omitempty"`
	Required  bool                       `json:"required"`
	ElemInfo  *ToolParameter             `json:"items,omitempty"`
	SubParams map[string]*ToolParameter  `json:"properties,omitempty"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Desc        string         `json:"description"`
	ParamsOneOf *ToolParams    `json:"params,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

type ToolParams struct {
	Params map[string]*ToolParameter `json:"params"`
}

func NewToolParams(params map[string]*ToolParameter) *ToolParams {
	if len(params) == 0 {
		return nil
	}
	return &ToolParams{Params: params}
}

func (p *ToolParams) ToJSONSchema() (map[string]any, error) {
	if p == nil || len(p.Params) == 0 {
		return nil, nil
	}

	properties := make(map[string]any, len(p.Params))
	required := make([]string, 0, len(p.Params))
	keys := make([]string, 0, len(p.Params))
	for key := range p.Params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		param := p.Params[key]
		if param == nil {
			continue
		}
		properties[key] = param.toJSONSchema()
		if param.Required {
			required = append(required, key)
		}
	}

	result := map[string]any{
		"type":       string(TypeObject),
		"properties": properties,
	}
	if len(required) > 0 {
		result["required"] = required
	}
	return result, nil
}

func (p *ToolParameter) toJSONSchema() map[string]any {

	result := map[string]any{
		"type": string(p.Type),
	}
	if p.Desc != "" {
		result["description"] = p.Desc
	}
	if len(p.Enum) > 0 {
		result["enum"] = append([]string(nil), p.Enum...)
	}
	if p.ElemInfo != nil {
		result["items"] = p.ElemInfo.toJSONSchema()
	}
	if len(p.SubParams) > 0 {
		properties := make(map[string]any, len(p.SubParams))
		required := make([]string, 0, len(p.SubParams))
		keys := make([]string, 0, len(p.SubParams))
		for key := range p.SubParams {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child := p.SubParams[key]
			if child == nil {
				continue
			}
			properties[key] = child.toJSONSchema()
			if child.Required {
				required = append(required, key)
			}
		}
		result["properties"] = properties
		if len(required) > 0 {
			result["required"] = required
		}
	}
	return result
}

type TokenUsage struct {
	PromptTokens       int `json:"prompt_tokens"`
	CompletionTokens   int `json:"completion_tokens"`
	TotalTokens        int `json:"total_tokens"`
	CachedPromptTokens int `json:"cached_prompt_tokens,omitempty"`
	ReasoningTokens    int `json:"reasoning_tokens,omitempty"`

	PromptTokenDetails PromptTokenDetails     `json:"prompt_token_details,omitempty"`
	CompletionTokensDetails CompletionTokenDetails `json:"completion_tokens_details,omitempty"`
}

type PromptTokenDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type CompletionTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type LLMClient interface {
	Stream(ctx context.Context, input []*LLMMessage, opts ...LLMOption) (*LLMStreamReader, error)
}

type LLMOptions struct {
	Model       *string
	MaxTokens   *int
	Temperature *float64
	Tools       []*ToolDefinition
}

type LLMOption func(*LLMOptions)

func WithModel(model string) LLMOption {
	return func(o *LLMOptions) { o.Model = &model }
}

func WithTools(tools []*ToolDefinition) LLMOption {
	return func(o *LLMOptions) { o.Tools = tools }
}

func GetLLMOptions(opts ...LLMOption) *LLMOptions {
	o := &LLMOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

type LLMStreamReader struct {
	ch  chan *LLMStreamChunk
	err error
}

type LLMStreamChunk struct {
	Message *LLMMessage
	Err     error
}

func NewLLMStreamReader(buf int) (*LLMStreamReader, *LLMStreamWriter) {
	ch := make(chan *LLMStreamChunk, buf)
	return &LLMStreamReader{ch: ch}, &LLMStreamWriter{ch: ch}
}

func (r *LLMStreamReader) Next() (*LLMMessage, error) {
	if r == nil || r.ch == nil {
		return nil, io.EOF
	}
	chunk, ok := <-r.ch
	if !ok {
		if r.err != nil {
			return nil, r.err
		}
		return nil, io.EOF
	}
	if chunk == nil {
		return nil, nil
	}
	if chunk.Err != nil {
		return nil, chunk.Err
	}
	return chunk.Message, nil
}

func (r *LLMStreamReader) Recv() (*LLMMessage, error) {
	return r.Next()
}

func (r *LLMStreamReader) Close() error {
	return nil
}

// CollectStream reads all chunks from a stream reader and assembles them into
// a single complete LLMMessage. This is used by runtimes that need full
// assistant turns rather than incremental deltas.
func CollectStream(reader *LLMStreamReader) (*LLMMessage, error) {
	return CollectStreamWithTextDelta(reader, nil)
}

func CollectStreamWithTextDelta(reader *LLMStreamReader, onDelta func(string)) (*LLMMessage, error) {
	if reader == nil {
		return nil, io.EOF
	}

	assembled := &LLMMessage{Role: RoleAssistant}
	toolCallsByIndex := make(map[int]*ToolCall)
	sawChunk := false

	for {
		msg, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if msg == nil {
			continue
		}

		sawChunk = true
		if msg.Role != "" {
			assembled.Role = msg.Role
		}
		if delta := msg.Content; delta != "" {
			assembled.Content += delta
			if onDelta != nil && len(msg.ToolCalls) == 0 {
				onDelta(delta)
			}
		}
		assembled.ReasoningContent += msg.ReasoningContent
		if len(msg.MultiContent) > 0 {
			assembled.MultiContent = append(assembled.MultiContent, msg.MultiContent...)
		}
		if msg.ToolCallID != "" {
			assembled.ToolCallID = msg.ToolCallID
		}
		if msg.ToolName != "" {
			assembled.ToolName = msg.ToolName
		}
		if msg.Name != "" {
			assembled.Name = msg.Name
		}
		if len(msg.Extra) > 0 {
			assembled.Extra = msg.Extra
		}
		mergeToolCallsByIndex(toolCallsByIndex, msg.ToolCalls)
		if responseMetaHasData(msg.ResponseMeta) {
			assembled.ResponseMeta = msg.ResponseMeta
		}
	}

	if !sawChunk {
		return nil, nil
	}
	assembled.ToolCalls = flattenToolCallsByIndex(toolCallsByIndex)
	return assembled, nil
}

type LLMStreamWriter struct {
	ch chan *LLMStreamChunk
}

func (w *LLMStreamWriter) Send(msg *LLMMessage) {
	if w == nil || w.ch == nil || msg == nil {
		return
	}
	w.ch <- &LLMStreamChunk{Message: msg}
}

func (w *LLMStreamWriter) SendError(err error) {
	if w == nil || w.ch == nil {
		return
	}
	w.ch <- &LLMStreamChunk{Err: err}
}

func (w *LLMStreamWriter) Close() {
	if w == nil || w.ch == nil {
		return
	}
	close(w.ch)
	w.ch = nil
}

func CloneMessages(messages []*LLMMessage) []*LLMMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*LLMMessage, len(messages))
	copy(out, messages)
	return out
}

func CloneMessage(msg *LLMMessage) *LLMMessage {
	if msg == nil {
		return nil
	}
	cloned := *msg
	if len(msg.ToolCalls) > 0 {
		cloned.ToolCalls = append([]ToolCall(nil), msg.ToolCalls...)
	}
	if len(msg.MultiContent) > 0 {
		cloned.MultiContent = append([]ChatMessagePart(nil), msg.MultiContent...)
	}
	if msg.ResponseMeta != nil {
		meta := *msg.ResponseMeta
		if msg.ResponseMeta.Usage != nil {
			usage := *msg.ResponseMeta.Usage
			meta.Usage = &usage
		}
		cloned.ResponseMeta = &meta
	}
	return &cloned
}

func mergeToolCallsByIndex(dest map[int]*ToolCall, toolCalls []ToolCall) {
	for i, toolCall := range toolCalls {
		idx := toolCall.Index
		if idx < 0 {
			idx = i
		}
		existing, ok := dest[idx]
		if !ok {
			cloned := toolCall
			dest[idx] = &cloned
			continue
		}
		if toolCall.ID != "" {
			existing.ID = toolCall.ID
		}
		if toolCall.Type != "" {
			existing.Type = toolCall.Type
		}
		existing.Index = idx
		existing.Function.Name += toolCall.Function.Name
		existing.Function.Arguments += toolCall.Function.Arguments
	}
}

func flattenToolCallsByIndex(toolCallsByIndex map[int]*ToolCall) []ToolCall {
	if len(toolCallsByIndex) == 0 {
		return nil
	}
	indexes := make([]int, 0, len(toolCallsByIndex))
	for idx := range toolCallsByIndex {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	out := make([]ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		if toolCall := toolCallsByIndex[idx]; toolCall != nil {
			out = append(out, *toolCall)
		}
	}
	return out
}

func responseMetaHasData(meta *ResponseMeta) bool {
	if meta == nil {
		return false
	}
	if meta.Usage != nil {
		return true
	}
	return meta.FinishReason != ""
}
