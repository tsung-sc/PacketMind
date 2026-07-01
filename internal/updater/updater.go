package updater

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

const GitHubSlug = "tsung-sc/PacketMind"

var semverVersionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// UpdateInfo describes an available update.
type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseNotes   string `json:"release_notes"`
	PublishedAt    string `json:"published_at"`
	DownloadURL    string `json:"download_url"`
	AssetSize      int64  `json:"asset_size"`
	HasUpdate      bool   `json:"has_update"`
}

// ProgressCallback is called during download with progress info.
type ProgressCallback func(downloaded int64, total int64)

// Updater handles checking for and applying application updates.
type Updater struct {
	currentVersion string
	progressCb     ProgressCallback
	newClient      updateClientFactory
}

type updateClientFactory func(progressCb ProgressCallback) (updateClient, error)

type updateClient interface {
	DetectLatest(ctx context.Context) (*releaseInfo, bool, error)
	UpdateTo(ctx context.Context, rel *releaseInfo) error
}

type releaseInfo struct {
	raw          *selfupdate.Release
	version      string
	releaseNotes string
	publishedAt  time.Time
	downloadURL  string
	assetSize    int64
	greaterThan  func(string) bool
}

// NewUpdater creates a new Updater with the given current version.
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: strings.TrimSpace(currentVersion),
		newClient:      newSelfUpdateClient,
	}
}

// CurrentVersion returns the current application version.
func (u *Updater) CurrentVersion() string {
	return u.currentVersion
}

// SetProgressCallback sets the callback for download progress.
func (u *Updater) SetProgressCallback(cb ProgressCallback) {
	u.progressCb = cb
}

// CheckForUpdate checks GitHub Releases for a newer version.
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	client, err := u.client()
	if err != nil {
		return nil, err
	}

	release, found, err := client.DetectLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect latest release: %w", err)
	}

	info := &UpdateInfo{CurrentVersion: u.currentVersion}
	if !found || release == nil {
		return info, nil
	}

	info.LatestVersion = release.version
	info.ReleaseNotes = release.releaseNotes
	info.PublishedAt = formatPublishedAt(release.publishedAt)
	info.DownloadURL = release.downloadURL
	info.AssetSize = release.assetSize
	info.HasUpdate = shouldUpdate(u.currentVersion, release)

	return info, nil
}

// PerformUpdate downloads and applies the latest available update.
func (u *Updater) PerformUpdate(ctx context.Context) error {
	client, err := u.client()
	if err != nil {
		return err
	}

	release, found, err := client.DetectLatest(ctx)
	if err != nil {
		return fmt.Errorf("detect latest release: %w", err)
	}
	if !found || release == nil {
		return fmt.Errorf("no compatible release found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if !shouldUpdate(u.currentVersion, release) {
		return nil
	}

	if err := client.UpdateTo(ctx, release); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}

	return nil
}

func (u *Updater) client() (updateClient, error) {
	factory := u.newClient
	if factory == nil {
		factory = newSelfUpdateClient
	}

	client, err := factory(u.progressCb)
	if err != nil {
		return nil, fmt.Errorf("create update client: %w", err)
	}

	return client, nil
}

func newSelfUpdateClient(progressCb ProgressCallback) (updateClient, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("create github source: %w", err)
	}

	var wrappedSource selfupdate.Source = source
	if progressCb != nil {
		wrappedSource = &progressSource{
			base: source,
			cb:   progressCb,
		}
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:        wrappedSource,
		Validator:     &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
		UniversalArch: "all",
	})
	if err != nil {
		return nil, fmt.Errorf("create updater: %w", err)
	}

	return &selfUpdateClient{
		updater:    updater,
		repository: selfupdate.ParseSlug(GitHubSlug),
	}, nil
}

type selfUpdateClient struct {
	updater    *selfupdate.Updater
	repository selfupdate.Repository
}

func (c *selfUpdateClient) DetectLatest(ctx context.Context) (*releaseInfo, bool, error) {
	release, found, err := c.updater.DetectLatest(ctx, c.repository)
	if err != nil || !found || release == nil {
		return nil, found, err
	}

	return &releaseInfo{
		raw:          release,
		version:      release.Version(),
		releaseNotes: release.ReleaseNotes,
		publishedAt:  release.PublishedAt,
		downloadURL:  release.AssetURL,
		assetSize:    int64(release.AssetByteSize),
		greaterThan:  release.GreaterThan,
	}, true, nil
}

func (c *selfUpdateClient) UpdateTo(ctx context.Context, rel *releaseInfo) error {
	if rel == nil || rel.raw == nil {
		return selfupdate.ErrInvalidRelease
	}

	exePath, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	return c.updater.UpdateTo(ctx, rel.raw, exePath)
}

type progressSource struct {
	base selfupdate.Source
	cb   ProgressCallback
}

func (s *progressSource) ListReleases(ctx context.Context, repository selfupdate.Repository) ([]selfupdate.SourceRelease, error) {
	return s.base.ListReleases(ctx, repository)
}

func (s *progressSource) DownloadReleaseAsset(ctx context.Context, rel *selfupdate.Release, assetID int64) (io.ReadCloser, error) {
	rc, err := s.base.DownloadReleaseAsset(ctx, rel, assetID)
	if err != nil {
		return nil, err
	}
	if s.cb == nil || rel == nil || assetID != rel.AssetID {
		return rc, nil
	}

	return newProgressReader(rc, int64(rel.AssetByteSize), s.cb), nil
}

type progressReader struct {
	reader     io.ReadCloser
	total      int64
	downloaded int64
	cb         ProgressCallback
}

func newProgressReader(reader io.ReadCloser, total int64, cb ProgressCallback) *progressReader {
	pr := &progressReader{
		reader: reader,
		total:  total,
		cb:     cb,
	}
	if cb != nil {
		cb(0, total)
	}
	return pr
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		if r.cb != nil {
			r.cb(r.downloaded, r.total)
		}
	}
	return n, err
}

func (r *progressReader) Close() error {
	if r.reader == nil {
		return nil
	}
	return r.reader.Close()
}

func shouldUpdate(currentVersion string, rel *releaseInfo) bool {
	if rel == nil {
		return false
	}

	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion == "" {
		return normalizeVersionLabel(rel.version) != ""
	}

	if semverVersionPattern.MatchString(currentVersion) {
		if greater, ok := rel.isGreaterThan(currentVersion); ok {
			return greater
		}
	}

	return normalizeVersionLabel(rel.version) != normalizeVersionLabel(currentVersion)
}

func (r *releaseInfo) isGreaterThan(currentVersion string) (greater bool, ok bool) {
	if r == nil || r.greaterThan == nil {
		return false, false
	}

	defer func() {
		if recover() != nil {
			greater = false
			ok = false
		}
	}()

	return r.greaterThan(currentVersion), true
}

func normalizeVersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if len(version) > 1 && (version[0] == 'v' || version[0] == 'V') && version[1] >= '0' && version[1] <= '9' {
		return version[1:]
	}
	return version
}

func formatPublishedAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

var _ selfupdate.Source = (*progressSource)(nil)
