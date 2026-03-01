package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/michael/hacker-news/internal/domain"
)

type fakeService struct {
	articles      []domain.Article
	read          map[int]bool
	readLater     map[int]bool
	articleBodies map[int]string
	preloadCalls  int
}

func (f *fakeService) LoadCachedStories(maxAge time.Duration) ([]domain.Article, bool, error) {
	return f.articles, true, nil
}

func (f *fakeService) RefreshStories(ctx context.Context, limit, workers int) ([]domain.Article, error) {
	return f.articles, nil
}

func (f *fakeService) FetchArticleMarkdown(ctx context.Context, article domain.Article) (string, error) {
	f.read[article.ID] = true
	if v, ok := f.articleBodies[article.ID]; ok {
		return v, nil
	}
	return "body", nil
}

func (f *fakeService) IsRead(id int) bool {
	return f.read[id]
}

func (f *fakeService) ToggleReadLater(id int) (bool, error) {
	next := !f.readLater[id]
	f.readLater[id] = next
	if !next {
		delete(f.readLater, id)
	}
	return next, nil
}

func (f *fakeService) IsReadLater(id int) bool {
	return f.readLater[id]
}

func (f *fakeService) Preload(ctx context.Context, stories []domain.Article, topN int) {
	f.preloadCalls++
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestModelApplyUnreadFilter(t *testing.T) {
	svc := &fakeService{
		read:      map[int]bool{2: true},
		readLater: map[int]bool{},
	}
	m := New(context.Background(), svc).(*model)
	m.articles = []domain.Article{{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}}
	m.height = 30
	m.applyFilters()
	if len(m.visible) != 2 {
		t.Fatalf("expected 2 visible, got %d", len(m.visible))
	}

	_, _ = m.Update(keyRune('u'))
	if !m.showUnreadOnly {
		t.Fatalf("expected unread filter on")
	}
	if len(m.visible) != 1 {
		t.Fatalf("expected 1 visible unread story, got %d", len(m.visible))
	}
	if m.articles[m.visible[0]].ID != 1 {
		t.Fatalf("wrong story after filter")
	}
}

func TestModelToggleReadLaterAndOpenArticle(t *testing.T) {
	svc := &fakeService{
		read:          map[int]bool{},
		readLater:     map[int]bool{},
		articleBodies: map[int]string{1: "# article"},
	}
	m := New(context.Background(), svc).(*model)
	m.height = 30
	m.width = 100
	m.articles = []domain.Article{{ID: 1, Title: "One", URL: "https://example.com"}}
	m.applyFilters()

	_, _ = m.Update(keyRune('s'))
	if !svc.readLater[1] {
		t.Fatalf("expected read-later enabled")
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected article load command")
	}
	msg := cmd()
	_, _ = m.Update(msg)

	if m.mode != modeArticle {
		t.Fatalf("expected article mode")
	}
	if !svc.read[1] {
		t.Fatalf("expected read state after loading article")
	}
	if m.articleRaw == "" {
		t.Fatalf("expected article markdown")
	}
	if len(m.articleLines) == 0 {
		t.Fatalf("expected rendered article lines")
	}
	if !strings.Contains(m.View(), "article") {
		t.Fatalf("expected article content in rendered view")
	}
}

func TestModelStoriesMsgTriggersPreloadAndRendersReadLaterBadge(t *testing.T) {
	svc := &fakeService{
		read:      map[int]bool{2: true},
		readLater: map[int]bool{1: true},
	}
	m := New(context.Background(), svc).(*model)
	m.width = 120
	m.height = 30

	_, _ = m.Update(storiesMsg{
		stories: []domain.Article{
			{ID: 1, Title: "Unread Saved", By: "a", Score: 10, CommentCount: 2},
			{ID: 2, Title: "Read Item", By: "b", Score: 1, CommentCount: 0},
		},
		source: "network",
	})

	if svc.preloadCalls == 0 {
		t.Fatalf("expected preloader to be called after stories load")
	}

	view := m.View()
	if !strings.Contains(view, "Unread Saved") {
		t.Fatalf("expected first story in view")
	}
	if !strings.Contains(view, "★") {
		t.Fatalf("expected read-later badge in view")
	}
}

func TestModelReadLaterOnlyFilter(t *testing.T) {
	svc := &fakeService{
		read:      map[int]bool{},
		readLater: map[int]bool{2: true},
	}
	m := New(context.Background(), svc).(*model)
	m.articles = []domain.Article{
		{ID: 1, Title: "Nope"},
		{ID: 2, Title: "Keep"},
	}
	m.height = 30
	m.applyFilters()

	_, _ = m.Update(keyRune('l'))
	if !m.showReadLaterOnly {
		t.Fatalf("expected read-later-only filter enabled")
	}
	if len(m.visible) != 1 {
		t.Fatalf("expected one visible item, got %d", len(m.visible))
	}
	if m.articles[m.visible[0]].ID != 2 {
		t.Fatalf("expected ID 2 to stay visible")
	}
}
