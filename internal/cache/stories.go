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

type storiesFile struct {
	FetchedAt time.Time        `json:"fetched_at"`
	Articles  []domain.Article `json:"articles"`
}

type StoriesCache struct {
	mu   sync.RWMutex
	path string
}

func NewStoriesCache(path string) *StoriesCache {
	return &StoriesCache{path: path}
}

func (c *StoriesCache) LoadFresh(maxAge time.Duration) ([]domain.Article, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b, err := os.ReadFile(c.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read stories cache: %w", err)
	}
	var sf storiesFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, false, fmt.Errorf("decode stories cache: %w", err)
	}
	if time.Since(sf.FetchedAt) > maxAge {
		return nil, false, nil
	}
	return sf.Articles, true, nil
}

func (c *StoriesCache) Save(articles []domain.Article) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(c.path), 0o750); err != nil {
		return fmt.Errorf("mkdir stories cache dir: %w", err)
	}
	b, err := json.MarshalIndent(storiesFile{FetchedAt: time.Now(), Articles: articles}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode stories cache: %w", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write stories tmp: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("rename stories cache: %w", err)
	}
	return nil
}
