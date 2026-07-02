package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func (u *Updater) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	client := u.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.latestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "PacketMind-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github release request failed: %s", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func releaseVersion(release *githubRelease) string {
	if release == nil {
		return ""
	}
	if tag := strings.TrimSpace(release.TagName); tag != "" {
		return tag
	}
	return strings.TrimSpace(release.Name)
}

func selectInstallerAsset(assets []githubAsset, goos, goarch string) *githubAsset {
	patterns := installerPatterns(goos, goarch)
	for _, pattern := range patterns {
		for i := range assets {
			if pattern(strings.ToLower(assets[i].Name)) {
				return &assets[i]
			}
		}
	}
	return nil
}

func installerPatterns(goos, goarch string) []func(string) bool {
	suffix := "_" + goos + "_" + goarch
	switch goos {
	case "windows":
		return []func(string) bool{
			func(name string) bool {
				return strings.Contains(name, "installer") && strings.Contains(name, suffix) && strings.HasSuffix(name, ".exe")
			},
			func(name string) bool {
				return strings.Contains(name, "setup") && strings.Contains(name, suffix) && strings.HasSuffix(name, ".exe")
			},
		}
	case "darwin":
		return []func(string) bool{
			func(name string) bool { return strings.Contains(name, suffix) && strings.HasSuffix(name, ".dmg") },
		}
	case "linux":
		return []func(string) bool{
			func(name string) bool { return strings.Contains(name, suffix) && strings.HasSuffix(name, ".tar.gz") },
			func(name string) bool { return strings.Contains(name, suffix) && strings.HasSuffix(name, ".appimage") },
		}
	default:
		return nil
	}
}

func runtimeGOOS() string {
	return runtime.GOOS
}

func runtimeGOARCH() string {
	return runtime.GOARCH
}
