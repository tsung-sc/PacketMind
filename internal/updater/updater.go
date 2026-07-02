package updater

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const GitHubSlug = "tsung-sc/PacketMind"

var semverVersionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseNotes   string `json:"release_notes"`
	PublishedAt    string `json:"published_at"`
	DownloadURL    string `json:"download_url"`
	AssetName      string `json:"asset_name"`
	AssetSize      int64  `json:"asset_size"`
	HasUpdate      bool   `json:"has_update"`
}

type DownloadedUpdateInfo struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
}

type ProgressCallback func(downloaded int64, total int64)

type Updater struct {
	currentVersion string
	progressCb     ProgressCallback
	httpClient     *http.Client
	latestURL      string
}

func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: strings.TrimSpace(currentVersion),
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
		},
		latestURL: fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubSlug),
	}
}

func (u *Updater) CurrentVersion() string {
	return u.currentVersion
}

func (u *Updater) SetProgressCallback(cb ProgressCallback) {
	u.progressCb = cb
}

func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	release, asset, err := u.latestInstaller(ctx)
	if err != nil {
		return nil, err
	}
	info := &UpdateInfo{CurrentVersion: u.currentVersion}
	if release == nil || asset == nil {
		return info, nil
	}
	version := releaseVersion(release)
	info.LatestVersion = version
	info.ReleaseNotes = release.Body
	info.PublishedAt = formatPublishedAt(release.PublishedAt)
	info.DownloadURL = asset.BrowserDownloadURL
	info.AssetName = asset.Name
	info.AssetSize = asset.Size
	info.HasUpdate = shouldUpdate(u.currentVersion, version)
	return info, nil
}

func (u *Updater) DownloadInstaller(ctx context.Context) (*DownloadedUpdateInfo, error) {
	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil || !info.HasUpdate || strings.TrimSpace(info.DownloadURL) == "" {
		return nil, nil
	}
	dir, err := updateDownloadDir()
	if err != nil {
		return nil, err
	}
	target, err := u.downloadToDir(ctx, info.DownloadURL, info.AssetName, dir, info.AssetSize)
	if err != nil {
		return nil, err
	}
	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		return nil, err
	}
	if err := verifyInstallerChecksum(ctx, u.httpClient, release.Assets, info.AssetName, target); err != nil {
		return nil, err
	}
	return &DownloadedUpdateInfo{Version: info.LatestVersion, Path: target, Name: info.AssetName, Size: info.AssetSize}, nil
}

func (u *Updater) latestInstaller(ctx context.Context) (*githubRelease, *githubAsset, error) {
	release, err := u.fetchLatestRelease(ctx)
	if err != nil || release == nil {
		return release, nil, err
	}
	asset := selectInstallerAsset(release.Assets, runtimeGOOS(), runtimeGOARCH())
	return release, asset, nil
}
