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
	"github.com/michael/hacker-news/internal/hnapi"
	"github.com/michael/hacker-news/internal/reader"
	"github.com/michael/hacker-news/internal/store"
)

func TestServiceRefreshAndReadFlow(t *testing.T) {
	temp := t.TempDir()
	stateStore, err := store.NewStateStore(filepath.Join(temp, "state.json"))
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	storiesCache := cache.NewStoriesCache(filepath.Join(temp, "stories.json"))
	contentCache, err := cache.NewContentCache(filepath.Join(temp, "content.json"))
	if err != nil {
		t.Fatalf("NewContentCache error: %v", err)
	}

	hnClient := hnapi.New(2 * time.Second)
	hnClient.SetHTTPClientForTest(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/v0/newstories.json":
				return jsonResponse(`[1]`), nil
			case "/v0/item/1.json":
				return jsonResponse(`{"id":1,"type":"story","title":"X","url":"https://example.com/post","by":"ab","score":1,"descendants":0,"time":1730000000}`), nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	})
	hnClient.SetBaseURL("https://example.com/v0")
	fetcher := reader.NewFetcher(2*time.Second, 24*time.Hour, contentCache)
	fetcherHTTP := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://example.com/post" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
			return htmlResponse(`<!doctype html><html><body><h1>Post</h1><p>Hello</p></body></html>`), nil
		}),
	}
	fetcher.SetHTTPClientForTest(fetcherHTTP)
	preloader := NewPreloader(fetcher, 1, 4)
	service := NewService(hnClient, storiesCache, stateStore, fetcher, preloader)

	stories, err := service.RefreshStories(context.Background(), 10, 2)
	if err != nil {
		t.Fatalf("RefreshStories error: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected one story, got %d", len(stories))
	}

	cached, ok, err := service.LoadCachedStories(1 * time.Hour)
	if err != nil || !ok || len(cached) != 1 {
		t.Fatalf("expected cached stories, ok=%v err=%v len=%d", ok, err, len(cached))
	}

	md, err := service.FetchArticleMarkdown(context.Background(), stories[0])
	if err != nil {
		t.Fatalf("FetchArticleMarkdown error: %v", err)
	}
	if md == "" {
		t.Fatalf("expected markdown")
	}
	if !service.IsRead(stories[0].ID) {
		t.Fatalf("expected story marked as read")
	}

	enabled, err := service.ToggleReadLater(stories[0].ID)
	if err != nil || !enabled || !service.IsReadLater(stories[0].ID) {
		t.Fatalf("expected read later enabled, enabled=%v err=%v", enabled, err)
	}

	service.Preload(context.Background(), []domain.Article{stories[0]}, 1)
	time.Sleep(100 * time.Millisecond)
}

func TestServiceFetchArticleMarkdownWithoutURL(t *testing.T) {
	temp := t.TempDir()
	stateStore, err := store.NewStateStore(filepath.Join(temp, "state.json"))
	if err != nil {
		t.Fatalf("NewStateStore error: %v", err)
	}
	storiesCache := cache.NewStoriesCache(filepath.Join(temp, "stories.json"))
	contentCache, err := cache.NewContentCache(filepath.Join(temp, "content.json"))
	if err != nil {
		t.Fatalf("NewContentCache error: %v", err)
	}

	hnClient := hnapi.New(2 * time.Second)
	fetcher := reader.NewFetcher(2*time.Second, 24*time.Hour, contentCache)
	preloader := NewPreloader(fetcher, 1, 4)
	service := NewService(hnClient, storiesCache, stateStore, fetcher, preloader)

	md, err := service.FetchArticleMarkdown(context.Background(), domain.Article{ID: 10, Title: "Ask HN"})
	if err != nil {
		t.Fatalf("FetchArticleMarkdown error: %v", err)
	}
	if md == "" || !strings.Contains(md, "No external URL") {
		t.Fatalf("expected no-url markdown placeholder, got: %q", md)
	}
	if service.IsRead(10) {
		t.Fatalf("expected no-url story not to be auto-marked read")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func htmlResponse(body string) *http.Response {
	resp := jsonResponse(body)
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	return resp
}
