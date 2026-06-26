package builtin

import (
	"fmt"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

func newFindPriorResponseSourcesHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		beforeRequestID, err := tools.GetRequiredStringArg(args, "before_request_id")
		if err != nil {
			return nil, err
		}
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		occurrences, err := store.FindPriorResponseSources(sessionID, beforeRequestID, value, tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to find prior response sources: %w", err)
		}

		requestIDs := make([]string, 0, len(occurrences))
		for _, occurrence := range occurrences {
			requestIDs = append(requestIDs, occurrence.RequestID)
		}

		content := mustMarshalJSON(map[string]any{
			"ok":                true,
			"before_request_id": beforeRequestID,
			"value":             value,
			"count":             len(occurrences),
			"results":           occurrences,
		})
		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Found %d prior response source candidates (value=%s)", len(occurrences), value),
			RequestIDs: requestIDs,
		}, nil
	}
}

func newFindLaterRequestUsagesHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		afterRequestID, err := tools.GetRequiredStringArg(args, "after_request_id")
		if err != nil {
			return nil, err
		}
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		occurrences, err := store.FindLaterRequestUsages(sessionID, afterRequestID, value, tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to find later request usages: %w", err)
		}

		requestIDs := make([]string, 0, len(occurrences))
		for _, occurrence := range occurrences {
			requestIDs = append(requestIDs, occurrence.RequestID)
		}

		content := mustMarshalJSON(map[string]any{
			"ok":               true,
			"after_request_id": afterRequestID,
			"value":            value,
			"count":            len(occurrences),
			"results":          occurrences,
		})
		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Found %d later request reuse candidates (value=%s)", len(occurrences), value),
			RequestIDs: requestIDs,
		}, nil
	}
}

func newTraceValueFlowHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		requestID, err := tools.GetRequiredStringArg(args, "request_id")
		if err != nil {
			return nil, err
		}
		fieldName, err := tools.GetRequiredStringArg(args, "field_name")
		if err != nil {
			return nil, err
		}
		location, err := tools.GetRequiredStringArg(args, "location")
		if err != nil {
			return nil, err
		}
		limit := tools.GetIntArg(args, "limit", defaultSearchLimit)

		chain, err := store.TraceValueFlow(sessionID, requestID, fieldName, storage.ParamLocation(strings.TrimSpace(location)), tools.NormalizeLimit(limit))
		if err != nil {
			return nil, fmt.Errorf("failed to trace value flow: %w", err)
		}

		requestIDs := make([]string, 0, len(chain.Links)+1)
		requestIDs = append(requestIDs, requestID)
		for _, link := range chain.Links {
			requestIDs = append(requestIDs, link.SourceRequestID)
		}

		content := mustMarshalJSON(map[string]any{
			"ok":         true,
			"request_id": requestID,
			"field_name": fieldName,
			"location":   location,
			"provenance": chain,
		})
		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Completed provenance trace for field %s, %d candidate links", fieldName, len(chain.Links)),
			RequestIDs: requestIDs,
		}, nil
	}
}
