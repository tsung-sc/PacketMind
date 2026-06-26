package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gocmd "github.com/go-cmd/cmd"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/agent/tools"
)

const (
	bashDefaultTimeout = 60 * time.Second
	bashMaxTimeout     = 300 * time.Second
	bashWorkspaceDir   = "data/agent-workspace"
)

type BashResult struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

func newBashHandler() tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		_ = sessionID
		command, err := tools.GetRequiredStringArg(args, "command")
		if err != nil {
			return nil, err
		}
		timeoutSec := tools.GetIntArg(args, "timeout", int(bashDefaultTimeout.Seconds()))
		workdir := tools.GetStringArg(args, "workdir", bashWorkspaceDir)

		result, err := ExecuteBash(context.Background(), command, workdir, time.Duration(timeoutSec)*time.Second)
		if err != nil {
			return &agentruntime.ToolExecutionResult{
				Content: mustMarshalJSON(map[string]any{"ok": false, "error": err.Error()}),
				Summary: fmt.Sprintf("bash execution failed: %s", err),
			}, nil
		}

		payload := map[string]any{"ok": true, "exit_code": result.ExitCode, "stdout": result.Stdout}
		if result.Stderr != "" {
			payload["stderr"] = result.Stderr
		}
		if result.TimedOut {
			payload["timed_out"] = true
			payload["ok"] = false
		}

		return &agentruntime.ToolExecutionResult{
			Content: mustMarshalJSON(payload),
			Summary: formatBashSummary(result),
		}, nil
	}
}

func ExecuteBash(ctx context.Context, command, workdir string, timeout time.Duration) (*BashResult, error) {
	if timeout <= 0 {
		timeout = bashDefaultTimeout
	}
	if timeout > bashMaxTimeout {
		timeout = bashMaxTimeout
	}
	if workdir == "" {
		workdir = bashWorkspaceDir
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return nil, fmt.Errorf("workspace dir: %w", err)
	}

	shell, args := detectShell(command)
	if shell == "" {
		return nil, fmt.Errorf("no shell found on this system")
	}

	c := gocmd.NewCmdOptions(gocmd.Options{
		Buffered:  true,
		Streaming: false,
	}, shell, args...)
	c.Dir = workdir
	c.Env = append(os.Environ(), "PACKETMIND=1")

	statusChan := c.Start()

	timedOut := false
	select {
	case <-ctx.Done():
		_ = c.Stop()
		timedOut = ctx.Err() == context.DeadlineExceeded
	case <-time.After(timeout):
		_ = c.Stop()
		timedOut = true
	case <-statusChan:
	}

	status := c.Status()

	stdoutStr := strings.Join(status.Stdout, "\n")
	stderrStr := strings.Join(status.Stderr, "\n")

	exitCode := status.Exit
	if exitCode == -1 && !timedOut && status.Error != nil {
		return nil, fmt.Errorf("bash execution failed: %w", status.Error)
	}

	return &BashResult{
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
		ExitCode: exitCode,
		TimedOut: timedOut,
	}, nil
}

func detectShell(command string) (string, []string) {
	if shell := os.Getenv("SHELL"); shell != "" {
		base := filepath.Base(shell)
		if base != "fish" && base != "nu" {
			return shell, []string{"-c", command}
		}
	}

	switch runtime.GOOS {
	case "windows":
		if p, err := exec.LookPath("cmd"); err == nil {
			return p, []string{"/c", "chcp 65001 >nul 2>&1 & " + command}
		}
		for _, c := range []string{"pwsh", "powershell"} {
			if p, err := exec.LookPath(c); err == nil {
				utf8Setup := `$OutputEncoding = [System.Text.Encoding]::UTF8; [Console]::OutputEncoding = [System.Text.Encoding]::UTF8; `
				return p, []string{"-NoProfile", "-NonInteractive", "-Command", utf8Setup + command}
			}
		}
	default:
		for _, c := range []string{"bash", "sh", "zsh"} {
			if _, err := exec.LookPath(c); err == nil {
				return c, []string{"-c", command}
			}
		}
	}
	return "", nil
}

func formatBashSummary(r *BashResult) string {	var parts []string
	if r.TimedOut {
		parts = append(parts, "bash timed out")
	} else {
		parts = append(parts, fmt.Sprintf("bash exited %d", r.ExitCode))
	}
	if r.Stdout != "" {
		preview := r.Stdout
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		parts = append(parts, preview)
	}
	if r.Stderr != "" {
		preview := r.Stderr
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		parts = append(parts, "stderr: "+preview)
	}
	return strings.Join(parts, "\n")
}

