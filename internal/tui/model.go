package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/michael/hacker-news/internal/domain"
)

type mode int

const (
	modeList mode = iota
	modeArticle
)

type storiesMsg struct {
	stories []domain.Article
	source  string
	err     error
}

type articleMsg struct {
	article  domain.Article
	markdown string
	err      error
}

type Service interface {
	LoadCachedStories(maxAge time.Duration) ([]domain.Article, bool, error)
	RefreshStories(ctx context.Context, limit, workers int) ([]domain.Article, error)
	FetchArticleMarkdown(ctx context.Context, article domain.Article) (string, error)
	IsRead(id int) bool
	ToggleReadLater(id int) (bool, error)
	IsReadLater(id int) bool
	Preload(ctx context.Context, stories []domain.Article, topN int)
}

type model struct {
	ctx     context.Context
	service Service

	articles  []domain.Article
	visible   []int
	mode      mode
	cursor    int
	listStart int

	showUnreadOnly    bool
	showReadLaterOnly bool

	loadingStories bool
	loadingArticle bool
	status         string
	err            error

	article      domain.Article
	articleRaw   string
	articleANSI  string
	articleLines []string
	articleStart int

	width  int
	height int

	styles styles
}

func New(ctx context.Context, service Service) tea.Model {
	return &model{
		ctx:            ctx,
		service:        service,
		mode:           modeList,
		loadingStories: true,
		status:         "Loading stories...",
		styles:         defaultStyles(),
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.loadCachedStoriesCmd(),
		m.refreshStoriesCmd(),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mode == modeArticle {
			m.renderArticle()
		}
		return m, nil

	case storiesMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loadingStories = false
			m.status = "Failed to load stories"
			return m, nil
		}
		if len(msg.stories) == 0 {
			return m, nil
		}

		m.articles = msg.stories
		m.loadingStories = false
		m.err = nil
		m.status = fmt.Sprintf("Loaded %d stories (%s)", len(msg.stories), msg.source)
		m.applyFilters()
		m.service.Preload(m.ctx, m.articles, 20)
		return m, nil

	case articleMsg:
		m.loadingArticle = false
		if msg.err != nil {
			m.err = msg.err
			m.status = "Failed to load article"
			return m, nil
		}
		m.err = nil
		m.article = msg.article
		m.articleRaw = msg.markdown
		m.articleStart = 0
		m.renderArticle()
		m.status = "Article loaded"
		m.applyFilters()
		return m, nil

	case tea.KeyMsg:
		if m.mode == modeList {
			return m.updateList(msg)
		}
		return m.updateArticle(msg)
	}

	return m, nil
}

func (m *model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
			m.ensureListCursorVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureListCursorVisible()
		}
	case "r":
		m.loadingStories = true
		m.status = "Refreshing stories..."
		return m, m.refreshStoriesCmd()
	case "u":
		m.showUnreadOnly = !m.showUnreadOnly
		m.applyFilters()
	case "l":
		m.showReadLaterOnly = !m.showReadLaterOnly
		m.applyFilters()
	case "s":
		article, ok := m.selectedArticle()
		if !ok {
			return m, nil
		}
		enabled, err := m.service.ToggleReadLater(article.ID)
		if err != nil {
			m.err = err
			m.status = "Failed to toggle read later"
			return m, nil
		}
		if enabled {
			m.status = "Saved to read later"
		} else {
			m.status = "Removed from read later"
		}
		m.applyFilters()
	case "enter", "o":
		article, ok := m.selectedArticle()
		if !ok {
			return m, nil
		}
		m.mode = modeArticle
		m.loadingArticle = true
		m.article = article
		m.status = "Loading article content..."
		return m, m.loadArticleCmd(article)
	}
	return m, nil
}

func (m *model) updateArticle(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.mode = modeList
		m.applyFilters()
		return m, nil
	case "j", "down":
		if m.articleStart < max(0, len(m.articleLines)-m.articleBodyHeight()) {
			m.articleStart++
		}
	case "k", "up":
		if m.articleStart > 0 {
			m.articleStart--
		}
	case "s":
		enabled, err := m.service.ToggleReadLater(m.article.ID)
		if err != nil {
			m.err = err
			m.status = "Failed to toggle read later"
			return m, nil
		}
		if enabled {
			m.status = "Saved to read later"
		} else {
			m.status = "Removed from read later"
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.mode == modeArticle {
		return m.viewArticle()
	}
	return m.viewList()
}

func (m *model) viewList() string {
	headline := m.styles.header.Render("HNX • Hacker News Design TUI")

	filters := []string{}
	if m.showUnreadOnly {
		filters = append(filters, "Unread")
	}
	if m.showReadLaterOnly {
		filters = append(filters, "Read Later")
	}
	filterLabel := "All"
	if len(filters) > 0 {
		filterLabel = strings.Join(filters, " + ")
	}

	sub := m.styles.subtle.Render(
		fmt.Sprintf("Filter: %s | j/k move • enter open • s save later • u unread • l read-later • r refresh • q quit", filterLabel),
	)

	bodyHeight := max(1, m.height-4)
	lines := make([]string, 0, bodyHeight)
	if len(m.visible) == 0 {
		if m.loadingStories {
			lines = append(lines, "Loading stories...")
		} else {
			lines = append(lines, "No stories match current filters.")
		}
	} else {
		m.ensureListCursorVisible()
		end := min(len(m.visible), m.listStart+bodyHeight)
		for i := m.listStart; i < end; i++ {
			article := m.articles[m.visible[i]]
			isCursor := i == m.cursor
			isRead := m.service.IsRead(article.ID)
			isLater := m.service.IsReadLater(article.ID)

			prefix := "  "
			if isCursor {
				prefix = "❯ "
			}
			badge := " "
			if isLater {
				badge = "★"
			}
			title := article.Title
			if title == "" {
				title = "(untitled)"
			}

			meta := fmt.Sprintf("%s | %d pts | %d comments", article.By, article.Score, article.CommentCount)
			line := fmt.Sprintf("%s%s %s", prefix, badge, title)
			if isRead {
				line = m.styles.read.Render(line)
				meta = m.styles.readMeta.Render(meta)
			} else {
				line = m.styles.unread.Render(line)
				meta = m.styles.meta.Render(meta)
			}
			if isCursor {
				line = m.styles.selected.Render(line)
			}
			lines = append(lines, line)
			if len(lines) < bodyHeight {
				lines = append(lines, "   "+meta)
			}
		}
	}

	content := strings.Join(lines, "\n")
	status := m.styles.footer.Render(m.statusLine())

	return lipgloss.JoinVertical(lipgloss.Left, headline, sub, content, status)
}

func (m *model) viewArticle() string {
	headline := m.styles.header.Render(m.article.Title)
	subtitle := m.styles.subtle.Render(
		"b/esc back • j/k scroll • s save later • q quit",
	)

	bodyHeight := m.articleBodyHeight()
	body := "Loading article..."
	if !m.loadingArticle {
		if len(m.articleLines) == 0 {
			body = "No article content"
		} else {
			end := min(len(m.articleLines), m.articleStart+bodyHeight)
			body = strings.Join(m.articleLines[m.articleStart:end], "\n")
		}
	}

	footer := m.styles.footer.Render(m.statusLine())
	return lipgloss.JoinVertical(lipgloss.Left, headline, subtitle, body, footer)
}

func (m *model) statusLine() string {
	if m.err != nil {
		return "Error: " + m.err.Error()
	}
	return m.status
}

func (m *model) applyFilters() {
	m.visible = m.visible[:0]
	for idx, article := range m.articles {
		if m.showUnreadOnly && m.service.IsRead(article.ID) {
			continue
		}
		if m.showReadLaterOnly && !m.service.IsReadLater(article.ID) {
			continue
		}
		m.visible = append(m.visible, idx)
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(0, len(m.visible)-1)
	}
	m.ensureListCursorVisible()
}

func (m *model) selectedArticle() (domain.Article, bool) {
	if len(m.visible) == 0 || m.cursor < 0 || m.cursor >= len(m.visible) {
		return domain.Article{}, false
	}
	return m.articles[m.visible[m.cursor]], true
}

func (m *model) ensureListCursorVisible() {
	if m.cursor < m.listStart {
		m.listStart = m.cursor
	}
	maxVisible := max(1, m.height-4)
	if m.cursor >= m.listStart+maxVisible {
		m.listStart = m.cursor - maxVisible + 1
	}
	if m.listStart < 0 {
		m.listStart = 0
	}
}

func (m *model) articleBodyHeight() int {
	return max(1, m.height-4)
}

func (m *model) renderArticle() {
	wrapWidth := max(40, m.width-4)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(wrapWidth),
	)
	if err != nil {
		m.articleANSI = m.articleRaw
	} else {
		ansi, rerr := renderer.Render(m.articleRaw)
		if rerr != nil {
			m.articleANSI = m.articleRaw
		} else {
			m.articleANSI = ansi
		}
	}
	m.articleLines = strings.Split(m.articleANSI, "\n")
}

func (m *model) loadCachedStoriesCmd() tea.Cmd {
	service := m.service
	return func() tea.Msg {
		stories, ok, err := service.LoadCachedStories(4 * time.Minute)
		if err != nil {
			return storiesMsg{err: err, source: "cache"}
		}
		if !ok {
			return storiesMsg{}
		}
		return storiesMsg{stories: stories, source: "cache"}
	}
}

func (m *model) refreshStoriesCmd() tea.Cmd {
	service := m.service
	ctx := m.ctx
	return func() tea.Msg {
		timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		stories, err := service.RefreshStories(timeoutCtx, 80, 12)
		if err != nil {
			return storiesMsg{err: err, source: "network"}
		}
		return storiesMsg{stories: stories, source: "network"}
	}
}

func (m *model) loadArticleCmd(article domain.Article) tea.Cmd {
	service := m.service
	ctx := m.ctx
	return func() tea.Msg {
		timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		md, err := service.FetchArticleMarkdown(timeoutCtx, article)
		return articleMsg{article: article, markdown: md, err: err}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
