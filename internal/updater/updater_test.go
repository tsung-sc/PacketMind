package updater

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	if u.newClient == nil {
		t.Fatal("expected default client factory to be configured")
	}
}

func TestUpdaterCurrentVersion(t *testing.T) {
	u := NewUpdater("v1.0.0")
	if got := u.CurrentVersion(); got != "v1.0.0" {
		t.Fatalf("expected version %q, got %q", "v1.0.0", got)
	}
}

func TestUpdaterCheckForUpdateWithMock(t *testing.T) {
	publishedAt := time.Date(2026, 4, 12, 8, 30, 0, 0, time.UTC)
	client := &fakeUpdateClient{
		release: &releaseInfo{
			version:      "1.1.0",
			releaseNotes: "bug fixes",
			publishedAt:  publishedAt,
			downloadURL:  "https://example.com/packetmind.zip",
			assetSize:    2048,
			greaterThan: func(current string) bool {
				return current == "1.0.0"
			},
		},
		found: true,
	}

	u := NewUpdater("1.0.0")
	u.newClient = func(ProgressCallback) (updateClient, error) {
		return client, nil
	}

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}
	if !info.HasUpdate {
		t.Fatalf("expected HasUpdate=true, got %+v", info)
	}
	if info.CurrentVersion != "1.0.0" || info.LatestVersion != "1.1.0" {
		t.Fatalf("unexpected versions: %+v", info)
	}
	if info.ReleaseNotes != "bug fixes" {
		t.Fatalf("expected release notes to be propagated, got %+v", info)
	}
	if info.PublishedAt != publishedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected published_at: %q", info.PublishedAt)
	}
	if info.DownloadURL != "https://example.com/packetmind.zip" || info.AssetSize != 2048 {
		t.Fatalf("unexpected download metadata: %+v", info)
	}
}

func TestUpdaterCheckForUpdateReturnsError(t *testing.T) {
	expectedErr := errors.New("boom")
	u := NewUpdater("1.0.0")
	u.newClient = func(ProgressCallback) (updateClient, error) {
		return &fakeUpdateClient{detectErr: expectedErr}, nil
	}

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error %v, got %v", expectedErr, err)
	}
}

func TestUpdaterPerformUpdateSkipsWhenAlreadyLatest(t *testing.T) {
	client := &fakeUpdateClient{
		release: &releaseInfo{
			version: "1.0.0",
			greaterThan: func(string) bool {
				return false
			},
		},
		found: true,
	}

	u := NewUpdater("1.0.0")
	u.newClient = func(ProgressCallback) (updateClient, error) {
		return client, nil
	}

	if err := u.PerformUpdate(context.Background()); err != nil {
		t.Fatalf("PerformUpdate failed: %v", err)
	}
	if client.updateCalls != 0 {
		t.Fatalf("expected update to be skipped, got %d calls", client.updateCalls)
	}
}

func TestUpdaterPerformUpdateWithNonSemverCurrentVersion(t *testing.T) {
	client := &fakeUpdateClient{
		release: &releaseInfo{version: "1.1.0"},
		found:   true,
	}

	u := NewUpdater("dev")
	u.newClient = func(ProgressCallback) (updateClient, error) {
		return client, nil
	}

	if err := u.PerformUpdate(context.Background()); err != nil {
		t.Fatalf("PerformUpdate failed: %v", err)
	}
	if client.updateCalls != 1 {
		t.Fatalf("expected update to be applied once, got %d", client.updateCalls)
	}
}

func TestProgressReaderReportsProgress(t *testing.T) {
	src := &trackingReadCloser{reader: bytes.NewReader([]byte("abcdef"))}
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

	if err := reader.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !src.closed {
		t.Fatal("expected underlying reader to be closed")
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

type fakeUpdateClient struct {
	release     *releaseInfo
	found       bool
	detectErr   error
	updateErr   error
	updateCalls int
}

func (c *fakeUpdateClient) DetectLatest(context.Context) (*releaseInfo, bool, error) {
	return c.release, c.found, c.detectErr
}

func (c *fakeUpdateClient) UpdateTo(context.Context, *releaseInfo) error {
	c.updateCalls++
	return c.updateErr
}

type trackingReadCloser struct {
	reader *bytes.Reader
	closed bool
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
