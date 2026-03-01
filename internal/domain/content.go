package domain

import "time"

// ArticleContent stores rendered Markdown for an article URL.
type ArticleContent struct {
	URL       string    `json:"url"`
	Markdown  string    `json:"markdown"`
	FetchedAt time.Time `json:"fetched_at"`
}
