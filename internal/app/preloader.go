package app

import (
	"context"
	"sync"

	"github.com/michael/hacker-news/internal/domain"
	"github.com/michael/hacker-news/internal/reader"
)

type Preloader struct {
	fetcher *reader.Fetcher
	workers int
	jobs    chan string
	once    sync.Once
}

func NewPreloader(fetcher *reader.Fetcher, workers int, queueSize int) *Preloader {
	if workers < 1 {
		workers = 1
	}
	if queueSize < workers {
		queueSize = workers * 2
	}
	return &Preloader{
		fetcher: fetcher,
		workers: workers,
		jobs:    make(chan string, queueSize),
	}
}

func (p *Preloader) Start(ctx context.Context) {
	p.once.Do(func() {
		for i := 0; i < p.workers; i++ {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case url := <-p.jobs:
						if url == "" {
							continue
						}
						_, _ = p.fetcher.FetchMarkdown(ctx, url)
					}
				}
			}()
		}
	})
}

func (p *Preloader) EnqueueStories(stories []domain.Article, n int) {
	if n <= 0 || n > len(stories) {
		n = len(stories)
	}
	for _, story := range stories[:n] {
		if !story.HasURL() {
			continue
		}
		select {
		case p.jobs <- story.URL:
		default:
		}
	}
}
