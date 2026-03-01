package reader

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/michael/hacker-news/internal/cache"
)

func TestFetcherFetchesAndCachesMarkdown(t *testing.T) {
	hits := 0

	contentCache, err := cache.NewContentCache(filepath.Join(t.TempDir(), "content.json"))
	if err != nil {
		t.Fatalf("NewContentCache error: %v", err)
	}

	f := NewFetcher(2*time.Second, 24*time.Hour, contentCache)
	f.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			hits++
			if hits > 1 {
				return nil, errors.New("network should not be used after cache is warm")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`<!doctype html><html><body><h1>Title</h1><p>Body text</p></body></html>`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	md1, err := f.FetchMarkdown(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("first FetchMarkdown error: %v", err)
	}
	if md1 == "" {
		t.Fatalf("expected markdown output")
	}

	md2, err := f.FetchMarkdown(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("second FetchMarkdown should hit cache: %v", err)
	}
	if md2 != md1 {
		t.Fatalf("expected cached markdown")
	}
	if hits != 1 {
		t.Fatalf("expected one HTTP hit, got %d", hits)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
