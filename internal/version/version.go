package version

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

const repoSlug = "ripnet/shellodex"

type versionCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"` // stored without "v" prefix
}

// CheckForUpdate returns the latest version string (e.g. "v1.0.2") if one is
// available and newer than current. Returns "" for dev builds or if up-to-date.
// Safe to call from a goroutine — network access happens only when the cache is
// older than 24 hours.
func CheckForUpdate(current string) string {
	if current == "dev" || current == "unknown" {
		return ""
	}

	cur := strings.TrimPrefix(current, "v")
	cache := readCache()

	if time.Since(cache.CheckedAt) < 24*time.Hour && cache.Latest != "" {
		if semverNewer(cache.Latest, cur) {
			return "v" + cache.Latest
		}
		return ""
	}

	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(repoSlug))
	if err != nil || !found {
		return ""
	}

	latestV := latest.Version()
	writeCache(versionCache{CheckedAt: time.Now(), Latest: latestV})

	if semverNewer(latestV, cur) {
		return "v" + latestV
	}
	return ""
}

// SelfUpdate downloads and replaces the running binary with the latest release.
// Returns the new version string on success.
func SelfUpdate(current string) (string, error) {
	cur := strings.TrimPrefix(current, "v")
	rel, err := selfupdate.UpdateSelf(context.Background(), cur, selfupdate.ParseSlug(repoSlug))
	if err != nil {
		return "", err
	}
	return "v" + rel.Version(), nil
}

// semverNewer reports whether a is strictly greater than b.
// Both should be in "X.Y.Z" format (no "v" prefix).
func semverNewer(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}

func cacheFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "shellodex", "version_check.json")
}

func readCache() versionCache {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		return versionCache{}
	}
	var c versionCache
	_ = json.Unmarshal(data, &c)
	return c
}

func writeCache(c versionCache) {
	f := cacheFile()
	if err := os.MkdirAll(filepath.Dir(f), 0755); err != nil {
		return
	}
	data, _ := json.Marshal(c)
	_ = os.WriteFile(f, data, 0644)
}
