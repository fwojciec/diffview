package fs

import (
	"os"
	"path/filepath"
)

// DefaultCacheDir returns the default cache directory for diffstory.
// Uses XDG_CACHE_HOME if set, otherwise falls back to ~/.cache/diffstory,
// or system temp directory if home is unavailable.
func DefaultCacheDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "diffstory")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "diffstory")
	}
	return filepath.Join(home, ".cache", "diffstory")
}
