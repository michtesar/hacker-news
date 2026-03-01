package hnapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/michael/hacker-news/internal/domain"
)

const defaultBaseURL = "https://hacker-news.firebaseio.com/v0"

type itemResponse struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	By          string `json:"by"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Time        int64  `json:"time"`
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func New(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    defaultBaseURL,
	}
}

func (c *Client) SetBaseURL(url string) {
	if url != "" {
		c.baseURL = url
	}
}

// SetHTTPClientForTest allows deterministic transport injection in tests.
func (c *Client) SetHTTPClientForTest(client *http.Client) {
	if client != nil {
		c.httpClient = client
	}
}

func (c *Client) FetchNewStoryIDs(ctx context.Context, limit int) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/newstories.json", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch new stories: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("newstories status: %s", res.Status)
	}

	var ids []int
	if err := json.NewDecoder(res.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("decode new stories: %w", err)
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (c *Client) FetchItem(ctx context.Context, id int) (domain.Article, error) {
	url := fmt.Sprintf("%s/item/%d.json", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Article{}, fmt.Errorf("build item request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Article{}, fmt.Errorf("fetch item %d: %w", id, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return domain.Article{}, fmt.Errorf("item %d status: %s", id, res.Status)
	}

	var ir itemResponse
	if err := json.NewDecoder(res.Body).Decode(&ir); err != nil {
		return domain.Article{}, fmt.Errorf("decode item %d: %w", id, err)
	}
	if ir.ID == 0 || ir.Type != "story" {
		return domain.Article{}, fmt.Errorf("item %d is not a story", id)
	}

	return domain.Article{
		ID:           ir.ID,
		Title:        ir.Title,
		URL:          ir.URL,
		By:           ir.By,
		Score:        ir.Score,
		CommentCount: ir.Descendants,
		CreatedAt:    time.Unix(ir.Time, 0),
	}, nil
}

func (c *Client) FetchStories(ctx context.Context, limit, workers int) ([]domain.Article, error) {
	ids, err := c.FetchNewStoryIDs(ctx, limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if workers < 1 {
		workers = 1
	}

	type result struct {
		article domain.Article
		err     error
	}

	jobs := make(chan int)
	results := make(chan result, len(ids))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				article, err := c.FetchItem(ctx, id)
				if err != nil {
					results <- result{err: err}
					continue
				}
				results <- result{article: article}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, id := range ids {
			select {
			case <-ctx.Done():
				return
			case jobs <- id:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	stories := make([]domain.Article, 0, len(ids))
	for r := range results {
		if r.err != nil {
			continue
		}
		stories = append(stories, r.article)
	}
	if len(stories) == 0 {
		return nil, fmt.Errorf("no stories fetched successfully")
	}

	sort.Slice(stories, func(i, j int) bool {
		return stories[i].CreatedAt.After(stories[j].CreatedAt)
	})
	return stories, nil
}
