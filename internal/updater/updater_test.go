package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewUpdater(t *testing.T) {
	u := NewUpdater("1.2.3")
	if u == nil {
		t.Fatal("expected updater instance")
	}
	if u.currentVersion != "1.2.3" {
		t.Fatalf("expected current version to be stored, got %q", u.currentVersion)
	}
	if u.httpClient == nil {
		t.Fatal("expected http client")
	}
}

func TestUpdaterCurrentVersion(t *testing.T) {
	u := NewUpdater("v1.0.0")
	if got := u.CurrentVersion(); got != "v1.0.0" {
		t.Fatalf("expected version %q, got %q", "v1.0.0", got)
	}
}

func TestUpdaterCheckForUpdateWithInstallerAsset(t *testing.T) {
	publishedAt := time.Date(2026, 4, 12, 8, 30, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(githubRelease{
			TagName:     "v1.1.0",
			Body:        "bug fixes",
			PublishedAt: publishedAt,
			Assets: []githubAsset{{
				Name:               "PacketMindInstaller_v1.1.0_windows_amd64.exe",
				BrowserDownloadURL: "https://example.com/installer.exe",
				Size:               2048,
			}},
		})
	}))
	defer server.Close()

	u := NewUpdater("v1.0.0")
	u.latestURL = server.URL
	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if !info.HasUpdate {
		t.Fatalf("expected HasUpdate=true, got %+v", info)
	}
	if info.CurrentVersion != "v1.0.0" || info.LatestVersion != "v1.1.0" {
		t.Fatalf("unexpected versions: %+v", info)
	}
	if info.ReleaseNotes != "bug fixes" {
		t.Fatalf("expected release notes, got %+v", info)
	}
	if info.PublishedAt != publishedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected published_at: %q", info.PublishedAt)
	}
	if info.AssetName != "PacketMindInstaller_v1.1.0_windows_amd64.exe" || info.AssetSize != 2048 {
		t.Fatalf("unexpected asset metadata: %+v", info)
	}
}

func TestUpdaterCheckForUpdateReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	u := NewUpdater("1.0.0")
	u.latestURL = server.URL
	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShouldUpdateSkipsAlreadyLatest(t *testing.T) {
	if shouldUpdate("v1.0.0", "v1.0.0") {
		t.Fatal("expected no update for same version")
	}
}

func TestShouldUpdateWithNonSemverCurrentVersion(t *testing.T) {
	if !shouldUpdate("dev", "v1.1.0") {
		t.Fatal("expected dev to update to release")
	}
}

func TestSelectInstallerAssetPrefersWindowsInstaller(t *testing.T) {
	assets := []githubAsset{
		{Name: "PacketMind_v1.1.0_windows_amd64.zip"},
		{Name: "PacketMindInstaller_v1.1.0_windows_amd64.exe"},
	}
	asset := selectInstallerAsset(assets, "windows", "amd64")
	if asset == nil || asset.Name != "PacketMindInstaller_v1.1.0_windows_amd64.exe" {
		t.Fatalf("unexpected asset: %+v", asset)
	}
}

func TestProgressReaderReportsProgress(t *testing.T) {
	src := bytes.NewReader([]byte("abcdef"))
	var updates [][2]int64
	reader := newProgressReader(src, 6, func(downloaded, total int64) {
		updates = append(updates, [2]int64{downloaded, total})
	})
	buf := make([]byte, 2)
	for {
		_, err := reader.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}
	if len(updates) < 2 {
		t.Fatalf("expected progress callbacks, got %d", len(updates))
	}
	if updates[0] != [2]int64{0, 6} {
		t.Fatalf("expected initial progress event, got %#v", updates[0])
	}
	if last := updates[len(updates)-1]; last != [2]int64{6, 6} {
		t.Fatalf("expected final progress event, got %#v", last)
	}
}
