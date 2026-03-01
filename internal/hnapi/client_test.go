package hnapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

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

func TestFetchNewStoryIDsLimit(t *testing.T) {
	c := New(2 * time.Second)
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v0/newstories.json" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return jsonResponse(`[10,9,8,7]`), nil
		}),
	}
	c.baseURL = "https://mock.local/v0"

	ids, err := c.FetchNewStoryIDs(context.Background(), 2)
	if err != nil {
		t.Fatalf("FetchNewStoryIDs error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 9 {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestFetchStories(t *testing.T) {
	c := New(2 * time.Second)
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/v0/newstories.json":
				return jsonResponse(`[1,2,3]`), nil
			case "/v0/item/1.json":
				return jsonResponse(`{"id":1,"type":"story","title":"A","url":"https://a.dev","by":"alice","score":10,"descendants":2,"time":1730000000}`), nil
			case "/v0/item/2.json":
				return jsonResponse(`{"id":2,"type":"comment"}`), nil
			case "/v0/item/3.json":
				return jsonResponse(`{"id":3,"type":"story","title":"B","url":"https://b.dev","by":"bob","score":5,"descendants":1,"time":1730000100}`), nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
					Header:     make(http.Header),
				}, nil
			}
		}),
	}
	c.baseURL = "https://mock.local/v0"

	stories, err := c.FetchStories(context.Background(), 10, 3)
	if err != nil {
		t.Fatalf("FetchStories error: %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(stories))
	}
	if stories[0].ID != 3 || stories[1].ID != 1 {
		t.Fatalf("stories not sorted by time desc: %+v", stories)
	}
}
