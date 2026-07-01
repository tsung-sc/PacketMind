package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/packetmind/packetmind/internal/agent"
	"github.com/packetmind/packetmind/internal/agent/mcp"
	"github.com/packetmind/packetmind/internal/api/bindings"
	"github.com/packetmind/packetmind/internal/appctx"
	"github.com/packetmind/packetmind/internal/config"
	"github.com/packetmind/packetmind/internal/proxy"
	"github.com/packetmind/packetmind/internal/storage"
	"github.com/packetmind/packetmind/internal/version"
)

//go:embed all:gui/dist
var assets embed.FS

type App struct {
	sessionAPI *bindings.SessionAPI
	requestAPI *bindings.RequestAPI
	agentAPI   *bindings.AgentAPI
	configAPI  *bindings.ConfigAPI
	proxyAPI   *bindings.ProxyAPI
	updaterAPI *bindings.UpdaterAPI
}

func NewApp() *App {
	return &App{
		sessionAPI: bindings.NewSessionAPI(),
		requestAPI: bindings.NewRequestAPI(),
		agentAPI:   bindings.NewAgentAPI(),
		configAPI:  bindings.NewConfigAPI(),
		proxyAPI:   bindings.NewProxyAPI(),
		updaterAPI: bindings.NewUpdaterAPI(version.Version),
	}
}

func resolveConfigDir() string {
	cliConfigDir := flag.String("config-dir", "", "Directory containing PacketMind config files")
	flag.Parse()

	configDir := strings.TrimSpace(*cliConfigDir)
	if configDir == "" {
		configDir = strings.TrimSpace(os.Getenv("PACKETMIND_CONFIG_DIR"))
	}
	if configDir == "" {
		configDir = "./configs"
	}
	if abs, err := filepath.Abs(configDir); err == nil {
		return abs
	}
	return configDir
}

func main() {
	fmt.Println("PacketMind Wails bootstrap")
	fmt.Printf("[PacketMind] version %s (build %s, commit %s)\n", version.Version, version.BuildTime, version.Commit)

	configDir := resolveConfigDir()
	fmt.Printf("[PacketMind] config dir: %s\n", configDir)

	modelsStore, err := config.LoadModelsStore(configDir)
	if err != nil {
		log.Fatalf("Warning: Failed to load models config: %v", err)
	}
	config.DefaultModelsStore = modelsStore
	appSettingsStore, err := config.LoadAppSettingsStore(configDir)
	if err != nil {
		log.Printf("Warning: Failed to load packetmind settings: %v, using defaults", err)
		appSettingsStore = config.NewAppSettingsStore(configDir, config.DefaultPacketMindSettings())
	}
	config.DefaultSettingsStore = appSettingsStore
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	store, err := storage.NewStorage()
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	storage.Default = store

	defaultSession := &storage.Session{Name: "Session 1"}
	if err := store.CreateSession(defaultSession); err != nil {
		log.Fatalf("Failed to create default session: %v", err)
	}

	prox, err := proxy.New(appSettingsStore.Snapshot())
	if err != nil {
		log.Fatalf("Failed to initialize proxy: %v", err)
	}
	proxy.Default = prox

	app := NewApp()

	prox.SetOnRequest(func(req *storage.Request) {
		sid := activeSessionID()
		app.emitRequest(bindings.ToRequestEvent(req, sid))
	})
	prox.SetOnComplete(func(req *storage.Request) {
		sid := activeSessionID()
		app.emitRequestComplete(bindings.ToRequestEvent(req, sid))
	})

	app.requestAPI.SetOnRequest(func(req *storage.Request) {
		sid := activeSessionID()
		app.emitRequest(bindings.ToRequestEvent(req, sid))
	})
	app.requestAPI.SetOnComplete(func(req *storage.Request) {
		sid := activeSessionID()
		app.emitRequestComplete(bindings.ToRequestEvent(req, sid))
	})

	go func() {
		if err := prox.Start(context.Background()); err != nil {
			log.Printf("Failed to start proxy: %v", err)
		}
	}()

	err = wails.Run(&options.App{
		Title:     "PacketMind",
		Width:     1400,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 700,
		Frameless: true,
		OnStartup: app.startup,
		OnShutdown: func(ctx context.Context) {
			_ = ctx
			if proxy.Default != nil {
				_ = proxy.Default.Stop()
			}
			if storage.Default != nil {
				_ = storage.Default.Close()
			}
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []any{
			app.sessionAPI,
			app.requestAPI,
			app.agentAPI,
			app.configAPI,
			app.proxyAPI,
			app.updaterAPI,
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            true,
				UseToolbar:                 false,
			},
			Appearance: mac.NSAppearanceNameAqua,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (a *App) startup(ctx context.Context) {
	appctx.Set(ctx)
	a.applyScreenRelativeWindowSize()

	mcpManager := initMCPManager(ctx, config.DefaultSettingsStore.Snapshot().MCP)
	if mcpManager != nil {
		agent.DefaultMCPManager = mcpManager
	}
}

func initMCPManager(ctx context.Context, mcpCfg config.MCPSettings) *mcp.Manager {
	enabled := make([]config.MCPServerConfig, 0)
	for _, s := range mcpCfg.Servers {
		if s.Enabled && s.Command != "" && s.Name != "" {
			enabled = append(enabled, s)
		}
	}
	if len(enabled) == 0 {
		return nil
	}

	manager := mcp.NewManager()
	for _, srv := range enabled {
		client, err := mcp.NewStdioClient(ctx, srv.Command, srv.Args, srv.Env)
		if err != nil {
			log.Printf("[MCP] Failed to connect %q: %v", srv.Name, err)
			continue
		}
		if err := manager.AddAdapter(srv.Name, client); err != nil {
			log.Printf("[MCP] Failed to register adapter %q: %v", srv.Name, err)
			continue
		}
		log.Printf("[MCP] Connected: %s", srv.Name)
	}

	if total, errs := manager.RegisterAll(ctx); total > 0 {
		log.Printf("[MCP] Registered %d tools from %d servers", total, manager.AdapterCount())
	} else if len(errs) > 0 {
		for name, e := range errs {
			log.Printf("[MCP] Tool discovery failed for %q: %v", name, e)
		}
	}

	if manager.AdapterCount() == 0 {
		return nil
	}
	return manager
}

func (a *App) applyScreenRelativeWindowSize() {
	if appctx.Ctx == nil {
		return
	}

	screens, err := runtime.ScreenGetAll(appctx.Ctx)
	if err != nil || len(screens) == 0 {
		log.Printf("[Wails] Screen info unavailable, using fallback centered window sizing: %v", err)
		runtime.WindowSetSize(appctx.Ctx, 1620, 980)
		runtime.WindowCenter(appctx.Ctx)
		return
	}

	var primaryScreen *runtime.Screen
	for i := range screens {
		if screens[i].IsPrimary {
			primaryScreen = &screens[i]
			break
		}
	}
	if primaryScreen == nil {
		primaryScreen = &screens[0]
	}

	screenW := primaryScreen.Size.Width
	screenH := primaryScreen.Size.Height

	const widthRatio = 0.90
	const heightRatio = 0.86
	minWidth := 1024
	minHeight := 700

	winW := int(float64(screenW) * widthRatio)
	winH := int(float64(screenH) * heightRatio)

	if winW < minWidth {
		winW = minWidth
	}
	if winH < minHeight {
		winH = minHeight
	}

	winX := (screenW - winW) / 2
	winY := (screenH - winH) / 2

	runtime.WindowSetPosition(appctx.Ctx, winX, winY)
	runtime.WindowSetSize(appctx.Ctx, winW, winH)

	log.Printf("[Wails] Window positioned: %dx%d at (%d,%d) on screen %dx%d",
		winW, winH, winX, winY, screenW, screenH)
}

func (a *App) emitRequest(req *bindings.RequestEventDTO) {
	if appctx.Ctx == nil || req == nil {
		return
	}
	runtime.EventsEmit(appctx.Ctx, "request:new", req)
}

func (a *App) emitRequestComplete(req *bindings.RequestEventDTO) {
	if appctx.Ctx == nil || req == nil {
		return
	}
	runtime.EventsEmit(appctx.Ctx, "request:complete", req)
}

func activeSessionID() string {
	if storage.Default == nil {
		return ""
	}
	sess, err := storage.Default.GetActiveSession()
	if err != nil || sess == nil {
		return ""
	}
	return sess.ID
}
