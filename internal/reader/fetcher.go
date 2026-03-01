package reader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	htmlmd "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/michael/hacker-news/internal/cache"
	"github.com/michael/hacker-news/internal/domain"
)

type Fetcher struct {
	httpClient *http.Client
	cache      *cache.ContentCache
	maxAge     time.Duration
}

func NewFetcher(httpTimeout, cacheMaxAge time.Duration, contentCache *cache.ContentCache) *Fetcher {
	return &Fetcher{
		httpClient: &http.Client{Timeout: httpTimeout},
		cache:      contentCache,
		maxAge:     cacheMaxAge,
	}
}

// SetHTTPClientForTest allows deterministic transport injection in tests.
func (f *Fetcher) SetHTTPClientForTest(client *http.Client) {
	if client != nil {
		f.httpClient = client
	}
}

func (f *Fetcher) FetchMarkdown(ctx context.Context, url string) (string, error) {
	if entry, ok := f.cache.Get(url, f.maxAge); ok {
		return entry.Markdown, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build article request: %w", err)
	}
	req.Header.Set("User-Agent", "hnx/1.0 (+https://news.ycombinator.com)")

	res, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch article: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("article status: %s", res.Status)
	}

	htmlBody, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read article body: %w", err)
	}

	md, err := htmlmd.ConvertString(string(htmlBody))
	if err != nil {
		return "", fmt.Errorf("convert article html to markdown: %w", err)
	}
	if md == "" {
		md = "_No readable content extracted from this page._"
	}

	_ = f.cache.Set(domain.ArticleContent{URL: url, Markdown: md, FetchedAt: time.Now()})
	return md, nil
}
