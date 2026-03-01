package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/michael/hacker-news/internal/domain"
)

func TestContentCacheGetSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "content.json")
	c, err := NewContentCache(path)
	if err != nil {
		t.Fatalf("NewContentCache error: %v", err)
	}

	entry := domain.ArticleContent{
		URL:       "https://example.com",
		Markdown:  "# hello",
		FetchedAt: time.Now(),
	}
	if err := c.Set(entry); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	got, ok := c.Get(entry.URL, 1*time.Hour)
	if !ok || got.Markdown != "# hello" {
		t.Fatalf("unexpected cache value: ok=%v got=%+v", ok, got)
	}

	_, ok = c.Get(entry.URL, -1*time.Second)
	if ok {
		t.Fatalf("expected stale entry")
	}
}
