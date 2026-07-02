package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var downloadHTTPClient = &http.Client{
	Timeout:   0,
	Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
}

func (u *Updater) downloadToDir(ctx context.Context, url, assetName, dir string, total int64) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, filepath.Base(assetName))
	if err := u.downloadFile(ctx, url, target, total); err != nil {
		return "", err
	}
	return target, nil
}

func (u *Updater) downloadFile(ctx context.Context, url, target string, total int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if total <= 0 {
		total = resp.ContentLength
	}
	reader := io.Reader(resp.Body)
	if u.progressCb != nil {
		reader = newProgressReader(resp.Body, total, u.progressCb)
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func verifyInstallerChecksum(ctx context.Context, client *http.Client, assets []githubAsset, assetName, target string) error {
	checksumsURL := ""
	for _, asset := range assets {
		if strings.EqualFold(asset.Name, "checksums.txt") {
			checksumsURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumsURL == "" {
		return nil
	}
	expected, err := downloadChecksum(ctx, client, checksumsURL, assetName)
	if err != nil || expected == "" {
		return err
	}
	actual, err := fileSHA256(target)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}
	return nil
}

func downloadChecksum(ctx context.Context, client *http.Client, url, assetName string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && filepath.Base(fields[1]) == filepath.Base(assetName) {
			return fields[0], nil
		}
	}
	return "", scanner.Err()
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func updateDownloadDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base, err = os.UserConfigDir()
		if err != nil || strings.TrimSpace(base) == "" {
			return "", fmt.Errorf("resolve user cache dir: %w", err)
		}
	}
	return filepath.Join(base, "PacketMind", "updates"), nil
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	cb         ProgressCallback
}

func newProgressReader(reader io.Reader, total int64, cb ProgressCallback) *progressReader {
	pr := &progressReader{reader: reader, total: total, cb: cb}
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
