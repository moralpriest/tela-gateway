# tela-gateway

A public **TELA** HTTP gateway: serve any on-chain TELA app from a DERO node
over plain HTTP(S) — no browser extension, no PureWolf, nothing for viewers to
install.

- **View apps:** just a URL. `derobeats.tela.example.com` or
  `https://gateway/scid/<64-hex>/`.
- **Wallet / EPOCH:** still handled locally by the user's Engram + XSWD
  (`ws://localhost:44326`); the gateway only serves the app.
- **Deploy anywhere:** single static Go binary or container. Runs as a plain
  binary, in Docker, or serverless (Lambda / Cloud Run / Fly.io / …).

See [`docs/architecture.md`](docs/architecture.md) for how it works.

---

## Quickstart (pick one)

### A. Go binary — no Docker

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway
go run ./cmd/server          # binds 0.0.0.0:8080; falls back to public DERO nodes
curl -s localhost:8080/health
```

Try an app: <http://localhost:8080/durl/derobeats.tela>

### B. Docker — single container

```bash
docker build -t tela-gateway:local .
docker run --rm -p 8080:8080 tela-gateway:local
```

### C. Docker Compose — with automatic HTTPS + subdomains

```bash
cd deploy
cp .env.example .env         # set TELA_HOST_SUFFIX + ACME_EMAIL to your domain
docker compose up -d         # gateway + Caddy (Let's Encrypt)
```

Now `https://<app>.tela.example.com` serves that TELA app. Full walkthrough:
[`docs/deploy-docker-compose.md`](docs/deploy-docker-compose.md).

---

## Choose a deploy target

| You want… | Guide |
| --- | --- |
| Run as a Go binary + systemd (no Docker) | [`docs/deploy-binary.md`](docs/deploy-binary.md) |
| Docker Compose + Caddy on a VPS (recommended self-host) | [`docs/deploy-docker-compose.md`](docs/deploy-docker-compose.md) |
| Single container, bring your own TLS | [`docs/deploy-docker-single.md`](docs/deploy-docker-single.md) |
| Serverless on AWS (Lambda + CloudFront) | [`docs/deploy-aws-lambda.md`](docs/deploy-aws-lambda.md) |
| Google Cloud Run (scale to zero) | [`docs/deploy-cloud-run.md`](docs/deploy-cloud-run.md) |
| Fly.io | [`docs/deploy-fly-io.md`](docs/deploy-fly-io.md) |
| Railway / Render / Kubernetes / ECS | [`docs/deploy-other-container.md`](docs/deploy-other-container.md) |
| Use my existing nginx / Cloudflare / Traefik | [`docs/reverse-proxies.md`](docs/reverse-proxies.md) |
| Custom domains & the `Host` header | [`docs/reverse-proxies.md`](docs/reverse-proxies.md) |
| Add app aliases / run the indexer | [`docs/indexer.md`](docs/indexer.md) |
| Something's broken | [`docs/troubleshooting.md`](docs/troubleshooting.md) |

---

## Endpoints

| Path | Action |
| --- | --- |
| `/` | Landing page |
| `/health` | JSON health/status (also echoes the resolved host) |
| `/scid/{64-hex}/…` | Serve any TELA INDEX directly (no alias needed) |
| `/durl/{name}` | Resolve an alias/dURL → redirect to its `/scid/…` |
| `<app>.<suffix>/…` | Host-based routing (needs `TELA_HOST_SUFFIX`) |

---

## Configuration

All configuration is via environment variables.

| Env | Default | Purpose |
| --- | --- | --- |
| `DERO_DAEMON_URLS` | public nodes (Foundation first) | Comma-separated daemon endpoints, tried in order with failover |
| `DERO_DAEMON_URL` | — | Legacy single-daemon alias |
| `PORT` | `8080` | Listen port (`cmd/server`) |
| `TELA_HOST_SUFFIX` | empty | e.g. `.tela.example.com` — enables `<app>.<suffix>` subdomain routing |
| `TELA_ALIASES` | empty | `name=scid,name2=scid2` — highest-priority aliases |
| `RESERVED_APPS` | `status` | Subdomains treated as reserved (not looked up as apps) |
| `ALIASES_S3_URI` | empty | `s3://bucket/aliases.json` — loaded on cold start |
| `TELA_DATA_DIR` | platform default | App clone cache dir (`/data` in Docker, `/tmp/…` on Lambda) |
| `HOST_MAP` | empty | Static `host=scid` overrides |

### Daemon failover

The gateway tries daemon endpoints in priority order, probing each with a
lightweight `DERO.GetInfo` (1.5s timeout, 30s cache). If one is down it moves to
the next. Run your own node and list it first for best performance.

---

## Development

Requires Go 1.24+.

```bash
go run ./cmd/local     # dev server on a random localhost port
go vet ./...
go build ./...
go test ./...
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and
[`docs/architecture.md`](docs/architecture.md).

## Modules of note

- `github.com/civilware/tela` — TELA reference implementation
- `derohe` is replaced with `github.com/DEROFDN/derohe`

## Notes

- Not all TELA dURLs end in `.tela` — `/scid/{hex}/` works for any app.
- The first request per app clones its files and can take a few seconds; repeat
  requests are served from cache.

## License

[MIT](LICENSE) © moralpriest
