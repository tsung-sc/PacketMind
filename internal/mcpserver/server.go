package mcpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/mark3labs/mcp-go/server"
	"github.com/packetmind/packetmind/internal/agent/mcp"
	"github.com/packetmind/packetmind/internal/storage"
)

var (
	mu      sync.Mutex
	httpSrv *http.Server
	running bool
)

func StartSSEServer(store *storage.Storage, host string, port int) error {
	mu.Lock()
	defer mu.Unlock()

	if running {
		return fmt.Errorf("MCP server already running")
	}

	s := mcp.NewBuiltinServer(store, nil)
	sseServer := server.NewSSEServer(s)

	addr := fmt.Sprintf("%s:%d", host, port)
	httpSrv = &http.Server{
		Addr:    addr,
		Handler: sseServer,
	}
	running = true

	log.Printf("[MCP Server] SSE listening on http://%s/sse", addr)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[MCP Server] error: %v", err)
		}
		mu.Lock()
		running = false
		httpSrv = nil
		mu.Unlock()
	}()

	return nil
}

func StopSSEServer() error {
	mu.Lock()
	defer mu.Unlock()

	if !running || httpSrv == nil {
		return nil
	}

	log.Printf("[MCP Server] stopping SSE server")
	return httpSrv.Shutdown(context.Background())
}

func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return running
}
