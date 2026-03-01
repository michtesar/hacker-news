package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/michael/hacker-news/internal/domain"
)

type ContentCache struct {
	mu      sync.RWMutex
	path    string
	entries map[string]domain.ArticleContent
}

func NewContentCache(path string) (*ContentCache, error) {
	c := &ContentCache{path: path, entries: map[string]domain.ArticleContent{}}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ContentCache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	b, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read content cache: %w", err)
	}
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, &c.entries); err != nil {
		return fmt.Errorf("decode content cache: %w", err)
	}
	return nil
}

func (c *ContentCache) Get(url string, maxAge time.Duration) (domain.ArticleContent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[url]
	if !ok {
		return domain.ArticleContent{}, false
	}
	if time.Since(entry.FetchedAt) > maxAge {
		return domain.ArticleContent{}, false
	}
	return entry, true
}

func (c *ContentCache) Set(entry domain.ArticleContent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[entry.URL] = entry
	return c.saveLocked()
}

func (c *ContentCache) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("mkdir content cache dir: %w", err)
	}
	b, err := json.Marshal(c.entries)
	if err != nil {
		return fmt.Errorf("encode content cache: %w", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write content cache tmp: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("rename content cache: %w", err)
	}
	return nil
}
