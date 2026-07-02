package updater

import (
	"fmt"
	"strings"
	"time"
)

func shouldUpdate(currentVersion string, latestVersion string) bool {
	currentVersion = strings.TrimSpace(currentVersion)
	latestVersion = strings.TrimSpace(latestVersion)
	if latestVersion == "" {
		return false
	}
	if currentVersion == "" {
		return normalizeVersionLabel(latestVersion) != ""
	}
	if strings.EqualFold(currentVersion, "dev") {
		return true
	}
	if semverVersionPattern.MatchString(currentVersion) && semverVersionPattern.MatchString(latestVersion) {
		return compareSemver(latestVersion, currentVersion) > 0
	}
	return normalizeVersionLabel(latestVersion) != normalizeVersionLabel(currentVersion)
}

func compareSemver(a, b string) int {
	ap := semverParts(a)
	bp := semverParts(b)
	for i := 0; i < 3; i++ {
		if ap[i] > bp[i] {
			return 1
		}
		if ap[i] < bp[i] {
			return -1
		}
	}
	return 0
}

func semverParts(version string) [3]int {
	version = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(version), "v"), "V")
	base := strings.SplitN(version, "-", 2)[0]
	base = strings.SplitN(base, "+", 2)[0]
	parts := strings.Split(base, ".")
	var out [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		_, _ = fmt.Sscanf(parts[i], "%d", &out[i])
	}
	return out
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
