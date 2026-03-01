package app

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/michael/hacker-news/internal/cache"
	"github.com/michael/hacker-news/internal/domain"
	"github.com/michael/hacker-news/internal/reader"
)

func TestPreloaderWarmsContentCache(t *testing.T) {
	contentCache, err := cache.NewContentCache(filepath.Join(t.TempDir(), "content.json"))
	if err != nil {
		t.Fatalf("NewContentCache error: %v", err)
	}

	fetcher := reader.NewFetcher(2*time.Second, 24*time.Hour, contentCache)
	fetcher.SetHTTPClientForTest(&http.Client{
		Transport: preloaderRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("<html><body><h1>Warm</h1></body></html>")),
				Header:     make(http.Header),
			}, nil
		}),
	})

	p := NewPreloader(fetcher, 2, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.Start(ctx)

	p.EnqueueStories([]domain.Article{
		{ID: 1, URL: "https://example.com/a"},
		{ID: 2, URL: "https://example.com/b"},
		{ID: 3},
	}, 3)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, okA := contentCache.Get("https://example.com/a", 1*time.Hour)
		_, okB := contentCache.Get("https://example.com/b", 1*time.Hour)
		if okA && okB {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected preloader to warm both cache entries before timeout")
}

type preloaderRoundTripFunc func(*http.Request) (*http.Response, error)

func (f preloaderRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
