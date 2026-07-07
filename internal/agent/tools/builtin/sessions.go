package builtin

import (
	"fmt"
	"time"

	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/storage"
)

func newListSessionsHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		sessions, err := store.ListSessions()
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}

		items := make([]map[string]any, 0, len(sessions))
		for _, s := range sessions {
			items = append(items, map[string]any{
				"id":         s.ID,
				"name":       s.Name,
				"is_active":  s.IsActive,
				"created_at": s.CreatedAt.Format(time.RFC3339),
			})
		}

		content := mustMarshalJSON(map[string]any{
			"ok":       true,
			"sessions": items,
			"count":    len(items),
		})

		return &agentruntime.ToolExecutionResult{
			Content: content,
			Summary: fmt.Sprintf("Found %d sessions", len(items)),
		}, nil
	}
}
