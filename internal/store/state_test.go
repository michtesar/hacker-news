package store

import (
	"path/filepath"
	"testing"
)

func TestStateStoreReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}

	if s.IsRead(101) {
		t.Fatalf("expected unread")
	}
	if err := s.MarkRead(101); err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}
	if !s.IsRead(101) {
		t.Fatalf("expected read")
	}

	enabled, err := s.ToggleReadLater(101)
	if err != nil {
		t.Fatalf("ToggleReadLater error: %v", err)
	}
	if !enabled || !s.IsReadLater(101) {
		t.Fatalf("expected read later enabled")
	}

	enabled, err = s.ToggleReadLater(101)
	if err != nil {
		t.Fatalf("ToggleReadLater second error: %v", err)
	}
	if enabled || s.IsReadLater(101) {
		t.Fatalf("expected read later disabled")
	}

	s2, err := NewStateStore(path)
	if err != nil {
		t.Fatalf("reload state store error: %v", err)
	}
	if !s2.IsRead(101) {
		t.Fatalf("expected persisted read state")
	}
}
