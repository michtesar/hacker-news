package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/michael/hacker-news/internal/domain"
)

func TestStoriesCacheFreshness(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stories.json")
	c := NewStoriesCache(path)

	articles := []domain.Article{{ID: 1, Title: "hello"}}
	if err := c.Save(articles); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	got, ok, err := c.LoadFresh(1 * time.Hour)
	if err != nil {
		t.Fatalf("LoadFresh error: %v", err)
	}
	if !ok || len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unexpected fresh cache: ok=%v got=%+v", ok, got)
	}

	_, ok, err = c.LoadFresh(-1 * time.Second)
	if err != nil {
		t.Fatalf("LoadFresh expired error: %v", err)
	}
	if ok {
		t.Fatalf("expected stale cache")
	}
}
