# Architecture

tela-gateway is a single stateless Go HTTP handler (`ServeTELA` in
`function.go`) that serves any on-chain **TELA** app from a DERO node — no
browser extension, no PureWolf, no local install for viewers.

## Request flow

```
                         ┌──────────────────────────┐
  viewer                 │        tela-gateway        │        DERO node(s)
  browser  ── HTTPS ──►  │  (reverse proxy strips TLS │  ── RPC ──►  daemon
                         │   & preserves Host header) │   GetSC / GetInfo
                         │                            │
                         │  1. resolve host/path → SCID
                         │  2. clone TELA app to disk (cached)
                         │  3. serve static files
                         └──────────────────────────┘
```

1. A reverse proxy (Caddy, nginx, Cloudflare, CloudFront, a cloud load
   balancer, …) terminates TLS and forwards the request to the gateway as plain
   HTTP, **preserving the original `Host`**.
2. The gateway resolves the request to a TELA INDEX contract (SCID):
   - **By host** — `<app>.<TELA_HOST_SUFFIX>` (e.g. `derobeats.tela.example.com`)
     → alias `derobeats` → SCID.
   - **By path** — `/durl/<name>` → alias → SCID, or `/scid/<64-hex>/…` for any
     app directly (no alias needed).
3. The gateway fetches the INDEX from a DERO daemon (with failover), clones the
   app's files into `TELA_DATA_DIR`, and serves them as static content. The
   clone is cached, so only the first request per app pays the fetch cost.

## Components

| File | Role |
| --- | --- |
| `function.go` | `ServeTELA` HTTP handler: routing, host→SCID, path→SCID, static serving, `/health`, 404 pages |
| `daemon.go` | Daemon endpoint list + failover (probes `DERO.GetInfo`, 1.5s timeout, 30s cache) |
| `aliases.go` | Alias resolution: `TELA_ALIASES` env → `aliases.json` (optionally from S3) → built-ins |
| `cmd/*` | Thin entrypoints wiring `ServeTELA` to local / server / Lambda / indexer |

## Routing rules

| Incoming | Resolves to |
| --- | --- |
| `<app>.<suffix>/…` | alias `<app>` → SCID (requires `TELA_HOST_SUFFIX`) |
| `/scid/<64-hex>/…` | that INDEX directly (works everywhere, no config) |
| `/durl/<name>` | alias `<name>` → 302 to `/scid/<scid>/` |
| `/health` | JSON health/status |
| `/` | landing page |
| unknown `<app>.<suffix>` | 404 "no such TELA app", links to `status.<suffix>` |

Reserved subdomains (not treated as apps) are configured with `RESERVED_APPS`
(default: `status`).

## Why the Host header matters

Host-based routing only works if the gateway receives the **viewer's** hostname.
Most reverse proxies preserve `Host` by default (Caddy, nginx with one line,
Cloudflare). A few strip or rewrite it:

- **AWS Lambda Function URLs** reject requests whose `Host` doesn't match the
  Function URL, returning `403 {"Message":null}`. In front of Lambda you must
  forward the viewer host as `X-Forwarded-Host` — the included CloudFront
  Function (`scripts/cf-function-host-rewrite.js`) does exactly this.

See [`reverse-proxies.md`](reverse-proxies.md) for a per-proxy cheatsheet.

## Statelessness & scaling

The gateway holds no state between requests except the on-disk app clone cache
in `TELA_DATA_DIR`. You can run any number of replicas behind a load balancer;
each maintains its own cache. Cold instances re-clone on first request per app
(the "cold start" cost).
