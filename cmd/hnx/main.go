package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/michael/hacker-news/internal/app"
	"github.com/michael/hacker-news/internal/cache"
	"github.com/michael/hacker-news/internal/hnapi"
	"github.com/michael/hacker-news/internal/reader"
	"github.com/michael/hacker-news/internal/store"
	"github.com/michael/hacker-news/internal/tui"
	"github.com/michael/hacker-news/internal/util"
)

func main() {
	if printHelpAndExitIfRequested() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "hnx failed: %v\n", err)
		os.Exit(1)
	}
}

func printHelpAndExitIfRequested() bool {
	help := flag.Bool("help", false, "Show usage")
	helpShort := flag.Bool("h", false, "Show usage")
	flag.Parse()
	if !*help && !*helpShort {
		return false
	}

	fmt.Println(`hnx - Hacker News Design TUI

Usage:
  hnx

Keys:
  j/k or arrows  Move
  enter          Open article
  s              Toggle read later
  u              Toggle unread filter
  l              Toggle read-later filter
  r              Refresh stories
  b or esc       Back from article
  q              Quit`)
	return true
}

func run(ctx context.Context) error {
	dataDir, err := util.AppDataDir()
	if err != nil {
		return err
	}

	stateStore, err := store.NewStateStore(filepath.Join(dataDir, "state.json"))
	if err != nil {
		return err
	}

	storiesCache := cache.NewStoriesCache(filepath.Join(dataDir, "stories_cache.json"))
	contentCache, err := cache.NewContentCache(filepath.Join(dataDir, "content_cache.json"))
	if err != nil {
		return err
	}

	hnClient := hnapi.New(12 * time.Second)
	fetcher := reader.NewFetcher(18*time.Second, 24*time.Hour, contentCache)
	preloader := app.NewPreloader(fetcher, 6, 64)
	service := app.NewService(hnClient, storiesCache, stateStore, fetcher, preloader)

	p := tea.NewProgram(
		tui.New(ctx, service),
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}
