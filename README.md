# HNX - Hacker News Design TUI (Go)

A production-style terminal app for reading new Hacker News stories with:

- Fast startup via local story cache
- Async network refresh and background preloading
- Rich article reader (fetch page -> convert HTML to Markdown -> render in terminal)
- Read tracking and read-later bookmarking
- Unread/read-later filters

## Stack

- TUI: Bubble Tea + Lip Gloss
- Markdown rendering: Glamour
- HTML -> Markdown conversion: html-to-markdown
- Persistence: JSON files in user cache dir

## Run And Use

```bash
go run ./cmd/hnx
```

Or build a binary:

```bash
go build -o hnx ./cmd/hnx
./hnx
```

Show usage:

```bash
go run ./cmd/hnx --help
```

At startup:

- The app loads cached stories first for fast initial rendering.
- It then refreshes in background from Hacker News and updates the list.
- It preloads top story pages in the background to speed up article opening.

## Controls

### List view

- `j` / `k` (or arrows): move selection
- `enter`: open article
- `s`: toggle read-later on selected story
- `u`: toggle unread-only filter
- `l`: toggle read-later-only filter
- `r`: refresh from network
- `q`: quit

### Article view

- `j` / `k`: scroll
- `s`: toggle read-later
- `b` or `esc`: back to list
- `q`: quit

## Storage

The app stores state in your OS cache directory (e.g. `~/Library/Caches/hnx`):

- `state.json` - read + read-later status
- `stories_cache.json` - cached story list for fast startup
- `content_cache.json` - rendered markdown cache for article pages

How persistence behaves:

- Opening an article with a URL marks it as read.
- `s` toggles read-later on the selected story (or current article in reader view).
- Read and read-later states survive app restarts.

## Architecture

- `cmd/hnx`: app composition + startup
- `internal/hnapi`: Hacker News API client with concurrent story fetch
- `internal/reader`: page fetch + HTML-to-Markdown + cache-aware loading
- `internal/cache`: story/content cache persistence
- `internal/store`: read/read-later state store
- `internal/app`: orchestration service + background preloader
- `internal/tui`: Bubble Tea model, view, and keyboard interaction

## Tests

Run all tests:

```bash
go test ./...
```

Run race tests:

```bash
go test -race ./...
```

Generate coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Detailed testing strategy and verification scope:

- [TESTING.md](/Users/michael/Developer/hacker-news/TESTING.md)

Test coverage includes:

- Unit tests for cache and state persistence
- Unit tests for HN API client behavior
- Unit tests for markdown fetcher/cache interactions
- App-service integration test across API + reader + state
- TUI model tests for filtering, bookmarking, article loading flow, and view markers

## CI

This repository includes a CI workflow at
`/Users/michael/Developer/hacker-news/.github/workflows/ci.yml`
that runs on every push and pull request.

It enforces:

- Go formatting (`gofmt`)
- Test suite (`go test ./...`)
- Race checks (`go test -race ./...`)
- Coverage threshold gate (`scripts/coverage_gate.sh`)

Coverage gate fails under `60%`.

You can run the same checks locally using commands in
[/Users/michael/Developer/hacker-news/TESTING.md](/Users/michael/Developer/hacker-news/TESTING.md).

## Security And Compliance

Security/compliance workflow:

- `/Users/michael/Developer/hacker-news/.github/workflows/security-compliance.yml`

It runs:

- `govulncheck` for known Go vulnerabilities
- `gosec` for static security checks
- FOSS license compliance (`scripts/foss_license_check.sh`)

## Release (Google Release Please)

Release PR automation is configured using:

- `/Users/michael/Developer/hacker-news/.github/workflows/release-please.yml`
- `/Users/michael/Developer/hacker-news/release-please-config.json`
- `/Users/michael/Developer/hacker-news/.release-please-manifest.json`

When a GitHub Release is published, this workflow builds and uploads:

- `hnx_darwin_arm64.tar.gz`

Workflow file:

- `/Users/michael/Developer/hacker-news/.github/workflows/release.yml`

## Homebrew Install

This repo includes a formula at:

- `/Users/michael/Developer/hacker-news/Formula/hnx.rb`

Install from this repository tap:

```bash
brew tap michael/hacker-news https://github.com/michael/hacker-news
brew install --HEAD michael/hacker-news/hnx
```
