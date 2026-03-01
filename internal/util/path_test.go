package util

import "testing"

func TestAppDataDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", home)
	dir, err := AppDataDir()
	if err != nil {
		t.Fatalf("AppDataDir error: %v", err)
	}
	if dir == "" {
		t.Fatalf("expected non-empty dir")
	}
}
