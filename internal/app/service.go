package app

import (
	"context"
	"time"

	"github.com/michael/hacker-news/internal/cache"
	"github.com/michael/hacker-news/internal/domain"
	"github.com/michael/hacker-news/internal/hnapi"
	"github.com/michael/hacker-news/internal/reader"
	"github.com/michael/hacker-news/internal/store"
)

type Service struct {
	hnClient     *hnapi.Client
	storiesCache *cache.StoriesCache
	state        *store.StateStore
	fetcher      *reader.Fetcher
	preloader    *Preloader
}

func NewService(
	hnClient *hnapi.Client,
	storiesCache *cache.StoriesCache,
	state *store.StateStore,
	fetcher *reader.Fetcher,
	preloader *Preloader,
) *Service {
	return &Service{
		hnClient:     hnClient,
		storiesCache: storiesCache,
		state:        state,
		fetcher:      fetcher,
		preloader:    preloader,
	}
}

func (s *Service) LoadCachedStories(maxAge time.Duration) ([]domain.Article, bool, error) {
	return s.storiesCache.LoadFresh(maxAge)
}

func (s *Service) RefreshStories(ctx context.Context, limit, workers int) ([]domain.Article, error) {
	stories, err := s.hnClient.FetchStories(ctx, limit, workers)
	if err != nil {
		return nil, err
	}
	_ = s.storiesCache.Save(stories)
	return stories, nil
}

func (s *Service) FetchArticleMarkdown(ctx context.Context, article domain.Article) (string, error) {
	if !article.HasURL() {
		return "_No external URL available for this story._", nil
	}
	md, err := s.fetcher.FetchMarkdown(ctx, article.URL)
	if err != nil {
		return "", err
	}
	_ = s.state.MarkRead(article.ID)
	return md, nil
}

func (s *Service) IsRead(id int) bool {
	return s.state.IsRead(id)
}

func (s *Service) ToggleReadLater(id int) (bool, error) {
	return s.state.ToggleReadLater(id)
}

func (s *Service) IsReadLater(id int) bool {
	return s.state.IsReadLater(id)
}

func (s *Service) Preload(ctx context.Context, stories []domain.Article, topN int) {
	s.preloader.Start(ctx)
	s.preloader.EnqueueStories(stories, topN)
}
