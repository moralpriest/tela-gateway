# Contributing to tela-gateway

Thanks for your interest in improving tela-gateway!

## Development setup

Requires **Go 1.24+**.

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway
go build ./...
go test ./...
```

Run the dev server (picks a free port on localhost, falls back to public DERO
nodes automatically):

```bash
go run ./cmd/local
```

Or the production HTTP server (binds `0.0.0.0:$PORT`, default 8080):

```bash
go run ./cmd/server
```

## Project layout

| Path | Purpose |
| --- | --- |
| `function.go` | Core HTTP handler `ServeTELA` — routing, host/alias resolution |
| `daemon.go` | DERO daemon endpoint failover |
| `aliases.go` | Alias resolution (env, `aliases.json`, S3, built-in) |
| `cmd/local` | Local dev entrypoint (127.0.0.1, random port) |
| `cmd/server` | HTTP server entrypoint (VPS / Docker / container hosts) |
| `cmd/lambda` | AWS Lambda entrypoint (`provided.al2023` bootstrap) |
| `cmd/indexer` | Optional chain scanner that regenerates `aliases.json` |
| `deploy/` | Docker Compose, Caddyfile, systemd unit |
| `docs/` | Per-platform deploy guides |
| `scripts/` | AWS deploy helpers |

## Before you open a PR

Run all three and make sure they pass:

```bash
go vet ./...
go build ./...
go test -race ./...
```

Update the relevant `docs/` page if you change configuration, environment
variables, or deploy behavior.

## Commit style

Short, imperative subject lines (e.g. `fix: preserve Host on Cloud Run`).
Conventional Commit prefixes (`feat:`, `fix:`, `docs:`, `chore:`) are welcome
but not required.

## Reporting bugs

Open an issue using the bug template. Please include how you're running the
gateway (binary / Docker / Lambda / etc.) and your relevant env vars (redact
secrets).
