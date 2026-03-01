# Testing Guide

This project is designed to be testable across all important layers:

- API ingestion from Hacker News
- Website content fetch + HTML-to-Markdown conversion
- Caching and persistence (`stories`, `content`, `read`, `read-later`)
- TUI interaction flows (filters, open article, save for later)
- Service orchestration and background preloading

## Test Suite Types

- Unit tests
  - `internal/hnapi`: request/response parsing, limits, and story filtering
  - `internal/reader`: fetch + conversion + cache hit behavior
  - `internal/cache`: cache freshness and serialization
  - `internal/store`: read/read-later persistence and toggling
  - `internal/tui`: model logic, keyboard flows, and rendered view assertions
  - `internal/util`: cache-dir behavior
- Integration-style tests
  - `internal/app`: end-to-end service flow over mocked transports:
    - refresh stories
    - persist and reload cache
    - fetch/render article
    - persist read state
    - toggle read-later
  - `internal/app/preloader_test.go`: background preloading warms content cache

## Deterministic Network Testing

Tests use mocked `http.RoundTripper` implementations, not live network calls.
This gives:

- deterministic outputs
- no flakiness from external services
- full control over error paths

## Commands

Run all tests:

```bash
go test ./...
```

Run race detector (recommended):

```bash
go test -race ./...
```

Run with coverage profile:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Run one package:

```bash
go test ./internal/tui -v
```

## What Is Explicitly Verified

- New story list fetching and per-item expansion
- Non-story item rejection
- Story ordering
- HTML page conversion to markdown
- Content cache reuse (second read avoids network)
- Story list cache freshness rules
- Read state persistence across store reloads
- Read-later add/remove behavior
- Service behavior for stories with and without external URL
- Preloader behavior for warming article cache
- TUI unread/read-later filtering and open-article flow
- TUI view output includes expected badges/content markers

## CI Recommendation

Use this minimum pipeline:

1. `gofmt -w` check (or `gofmt -l`)
2. `go test ./...`
3. `go test -race ./...`
4. Coverage threshold gate (team-defined, e.g. 70%+)


## Continuous Integration

GitHub Actions workflow: `.github/workflows/ci.yml`

The CI pipeline runs on push and pull request and enforces:

- gofmt formatting check
- `go test ./...`
- `go test -race ./...`
- total coverage gate via `scripts/coverage_gate.sh`

Current default coverage threshold is `60%` (set by `MIN_COVERAGE` in CI).

To run the same quality gate locally:

```bash
go test -covermode=atomic -coverprofile=coverage.out ./...
./scripts/coverage_gate.sh coverage.out 60
```

## Security And Compliance Checks

Run security scans locally:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...
```

Run FOSS license compliance check:

```bash
./scripts/foss_license_check.sh
```
