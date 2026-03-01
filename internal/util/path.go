package util

import (
	"os"
	"path/filepath"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func AppDataDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(base, "hnx")
	if err := EnsureDir(path); err != nil {
		return "", err
	}
	return path, nil
}
