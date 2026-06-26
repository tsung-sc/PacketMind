package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
)

type batchToolResult struct {
	Tool       string   `json:"tool"`
	OK         bool     `json:"ok"`
	Summary    string   `json:"summary,omitempty"`
	Content    string   `json:"content,omitempty"`
	RequestIDs []string `json:"request_ids,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func newBatchExecuteHandler(exec tools.ToolExecutor) tools.BuiltinHandler {
	return func(args map[string]any, defaultSessionID string) (*agentruntime.ToolExecutionResult, error) {
		calls := tools.ParseBatchCallsArg(args, "calls")
		if len(calls) == 0 {
			return nil, fmt.Errorf("calls is required")
		}

		results := make([]batchToolResult, len(calls))
		requestIDSet := make(map[string]struct{})
		var mu sync.Mutex
		var wg sync.WaitGroup

		for idx, call := range calls {
			wg.Add(1)
			go func(i int, c tools.BatchToolCall) {
				defer wg.Done()
				if c.Tool == "" {
					results[i] = batchToolResult{Tool: c.Tool, OK: false, Error: "tool name is required"}
					return
				}

				execResult, err := exec.Execute(context.Background(), c.Tool, c.Args, defaultSessionID)
				if err != nil {
					results[i] = batchToolResult{Tool: c.Tool, OK: false, Error: err.Error()}
					return
				}

				results[i] = batchToolResult{Tool: c.Tool, OK: true, Summary: execResult.Summary, Content: execResult.Content, RequestIDs: append([]string(nil), execResult.RequestIDs...)}
				mu.Lock()
				for _, requestID := range execResult.RequestIDs {
					if requestID != "" {
						requestIDSet[requestID] = struct{}{}
					}
				}
				mu.Unlock()
			}(idx, call)
		}

		wg.Wait()

		okCount := 0
		failCount := 0
		for _, result := range results {
			if result.OK {
				okCount++
			} else {
				failCount++
			}
		}

		requestIDs := make([]string, 0, len(requestIDSet))
		for id := range requestIDSet {
			requestIDs = append(requestIDs, id)
		}

		content := mustMarshalJSON(map[string]any{"ok": failCount == 0, "total": len(calls), "success": okCount, "failed": failCount, "results": results, "request_ids": requestIDs})
		return &agentruntime.ToolExecutionResult{Content: content, Summary: fmt.Sprintf("Batch executed %d calls (%d success, %d failed)", len(calls), okCount, failCount), RequestIDs: requestIDs}, nil
	}
}
