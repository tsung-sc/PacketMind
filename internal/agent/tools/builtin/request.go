package builtin

import (
	"fmt"
	"strings"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/storage"
)

func newGetRequestHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		requestID, err := tools.GetRequiredStringArg(args, "request_id")
		if err != nil {
			return nil, err
		}
		includeFullBody := tools.GetBoolArg(args, "include_full_body", false)

		req, err := store.GetRequest(sessionID, requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to get request %s: %w", requestID, err)
		}

		snapshotSessionID := strings.TrimSpace(req.SessionID)
		if snapshotSessionID == "" {
			snapshotSessionID = strings.TrimSpace(sessionID)
		}
		snapshot := makeRequestSnapshot(req, snapshotSessionID)
		if includeFullBody {
			snapshot.BodyPreview = fullBodyText(req.Body, req.ContentType, req.Headers)
			snapshot.ResponseBodyPreview = fullBodyText(req.RespBody, req.RespContentType, req.RespHeaders)
		}

		content := mustMarshalJSON(map[string]any{
			"ok":      true,
			"request": snapshot,
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Fetched request %s (%s %s)", req.ID, req.Method, req.Path),
			RequestIDs: []string{req.ID},
		}, nil
	}
}
