package bindings

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/packetmind/packetmind/internal/appctx"
	"github.com/packetmind/packetmind/internal/updater"
)

// UpdaterAPI provides auto-update bindings for the frontend.
type UpdaterAPI struct {
	updater *updater.Updater
}

// NewUpdaterAPI creates an UpdaterAPI with the given current version.
func NewUpdaterAPI(currentVersion string) *UpdaterAPI {
	u := &UpdaterAPI{
		updater: updater.NewUpdater(currentVersion),
	}
	u.updater.SetProgressCallback(func(downloaded, total int64) {
		if appctx.Ctx != nil && total > 0 {
			runtime.EventsEmit(appctx.Ctx, "update:progress", map[string]any{
				"downloaded": downloaded,
				"total":      total,
				"percent":    float64(downloaded) / float64(total) * 100,
			})
		}
	})
	return u
}

// GetVersion returns the current application version.
func (a *UpdaterAPI) GetVersion() SessionResponse {
	return SessionResponse{
		Code: 0,
		Data: map[string]any{
			"version": a.updater.CurrentVersion(),
		},
	}
}

// CheckForUpdate checks GitHub Releases for a newer version.
func (a *UpdaterAPI) CheckForUpdate() SessionResponse {
	if appctx.Ctx == nil {
		return SessionResponse{Code: 50003, Message: "context not initialized"}
	}
	info, err := a.updater.CheckForUpdate(appctx.Ctx)
	if err != nil {
		return SessionResponse{Code: 50001, Message: fmt.Sprintf("check update failed: %v", err)}
	}
	return SessionResponse{Code: 0, Data: info}
}

func (a *UpdaterAPI) DownloadUpdate() SessionResponse {
	if appctx.Ctx == nil {
		return SessionResponse{Code: 50003, Message: "context not initialized"}
	}
	info, err := a.updater.DownloadInstaller(appctx.Ctx)
	if err != nil {
		return SessionResponse{Code: 50001, Message: fmt.Sprintf("download update failed: %v", err)}
	}
	if info == nil {
		return SessionResponse{Code: 40004, Message: "no update available"}
	}
	if appctx.Ctx != nil {
		runtime.EventsEmit(appctx.Ctx, "update:done", map[string]any{
			"success": true,
			"path":    info.Path,
			"name":    info.Name,
			"version": info.Version,
		})
	}
	return SessionResponse{Code: 0, Message: "Update installer downloaded", Data: info}
}

func (a *UpdaterAPI) OpenDownloadedUpdate(path string) SessionResponse {
	if err := updater.OpenPath(path); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}
	return SessionResponse{Code: 0}
}

func (a *UpdaterAPI) ShowDownloadedUpdate(path string) SessionResponse {
	if err := updater.ShowInFolder(path); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}
	return SessionResponse{Code: 0}
}

// PerformUpdate downloads and applies the latest update.
func (a *UpdaterAPI) PerformUpdate() SessionResponse {
	return a.DownloadUpdate()
}
