package runtime

import "github.com/packetmind/packetmind/internal/agent/llmtypes"

func AddTokenUsage(base, delta *llmtypes.TokenUsage) *llmtypes.TokenUsage {
	if delta == nil {
		return CloneTokenUsage(base)
	}
	if base == nil {
		return CloneTokenUsage(delta)
	}

	return &llmtypes.TokenUsage{
		PromptTokens:       base.PromptTokens + delta.PromptTokens,
		CompletionTokens:   base.CompletionTokens + delta.CompletionTokens,
		TotalTokens:        base.TotalTokens + delta.TotalTokens,
		CachedPromptTokens: base.CachedPromptTokens + delta.CachedPromptTokens,
		ReasoningTokens:    base.ReasoningTokens + delta.ReasoningTokens,
		PromptTokenDetails: llmtypes.PromptTokenDetails{
			CachedTokens: base.PromptTokenDetails.CachedTokens + delta.PromptTokenDetails.CachedTokens,
		},
		CompletionTokensDetails: llmtypes.CompletionTokenDetails{
			ReasoningTokens: base.CompletionTokensDetails.ReasoningTokens + delta.CompletionTokensDetails.ReasoningTokens,
		},
	}
}

func CloneTokenUsage(usage *llmtypes.TokenUsage) *llmtypes.TokenUsage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	return &cloned
}

func messageUsage(msg *llmtypes.LLMMessage) *llmtypes.TokenUsage {
	if msg == nil || msg.ResponseMeta == nil {
		return nil
	}
	return msg.ResponseMeta.Usage
}
