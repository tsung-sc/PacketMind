package openai

import (
	"context"
	"fmt"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
	"github.com/packetmind/packetmind/internal/agent/llmcore"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

func (c *Client) Stream(ctx context.Context, input []*llmtypes.LLMMessage, opts ...llmtypes.LLMOption) (*llmtypes.LLMStreamReader, error) {
	if c == nil || c.oc == nil {
		return nil, fmt.Errorf("%s client is nil", c.providerID())
	}
	options := llmtypes.GetLLMOptions(opts...)
	params := buildProviderPayload(defaultModelID(c.modelID, options), input, collectProviderModelTools(options.Tools), options)
	params.StreamOptions = openaisdk.ChatCompletionStreamOptionsParam{IncludeUsage: openaisdk.Bool(true)}
	stream, err := c.newStreamingChat(ctx, params, c.providerName())
	if err != nil {
		return nil, err
	}
	return consumeStream(stream, c.config.ExtractReasoningInStream), nil
}

func (c *Client) newStreamingChat(ctx context.Context, params openaisdk.ChatCompletionNewParams, providerName string, opts ...option.RequestOption) (*ssestream.Stream[openaisdk.ChatCompletionChunk], error) {
	return newStreamingChat(ctx, c.oc, params, providerName, opts...)
}

func newStreamingChat(ctx context.Context, client *openaisdk.Client, params openaisdk.ChatCompletionNewParams, providerName string, opts ...option.RequestOption) (*ssestream.Stream[openaisdk.ChatCompletionChunk], error) {
	return llmcore.RetryProviderCall(ctx, providerName, "stream", func() (*ssestream.Stream[openaisdk.ChatCompletionChunk], error) {
		stream := client.Chat.Completions.NewStreaming(ctx, params, opts...)
		if stream.Err() != nil {
			return nil, wrapOpenAIError(stream.Err(), providerName, "stream")
		}
		return stream, nil
	}, llmcore.RetryNotifierFromContext(ctx))
}

func consumeStream(stream *ssestream.Stream[openaisdk.ChatCompletionChunk], includeReasoning bool) *llmtypes.LLMStreamReader {
	sr, sw := llmtypes.NewLLMStreamReader(100)
	go func() {
		defer sw.Close()
		defer stream.Close()
		for stream.Next() {
			chunk := stream.Current()
			usage := toLLMTokenUsage(chunk.Usage)
			if len(chunk.Choices) == 0 {
				if usage != nil {
					sw.Send(&llmtypes.LLMMessage{Role: llmtypes.RoleAssistant, ResponseMeta: &llmtypes.ResponseMeta{Usage: usage}})
				}
				continue
			}
			choice := chunk.Choices[0]
			msg := &llmtypes.LLMMessage{Role: llmtypes.RoleAssistant, Content: choice.Delta.Content}
			if includeReasoning {
				msg.ReasoningContent = extraFieldString(choice.Delta.JSON.ExtraFields, "reasoning_content")
			}
			if len(choice.Delta.ToolCalls) > 0 {
				msg.ToolCalls = toLLMDeltaToolCalls(choice.Delta.ToolCalls)
			}
			if choice.FinishReason != "" && choice.FinishReason != "null" || usage != nil {
				msg.ResponseMeta = &llmtypes.ResponseMeta{FinishReason: choice.FinishReason, Usage: usage}
			}
			if msg.Content != "" || msg.ReasoningContent != "" || len(msg.ToolCalls) > 0 || msg.ResponseMeta != nil {
				sw.Send(msg)
			}
		}
		if err := stream.Err(); err != nil {
			sw.SendError(err)
		}
	}()
	return sr
}

func toLLMDeltaToolCalls(toolCalls []openaisdk.ChatCompletionChunkChoiceDeltaToolCall) []llmtypes.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	out := make([]llmtypes.ToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		out = append(out, llmtypes.ToolCall{
			Index: int(toolCall.Index),
			ID:    toolCall.ID,
			Type:  toolCall.Type,
			Function: llmtypes.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})
	}
	return out
}
