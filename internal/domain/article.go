package domain

import "time"

// Article is a Hacker News story with UI metadata.
type Article struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	By           string    `json:"by"`
	Score        int       `json:"score"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
}

func (a Article) HasURL() bool {
	return a.URL != ""
}
