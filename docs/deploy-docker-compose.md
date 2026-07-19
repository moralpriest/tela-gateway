# Deploy: Docker Compose + Caddy (recommended self-host path)

This is the easiest way to run a public gateway with automatic HTTPS and
per-app subdomains. Two containers: the gateway and Caddy. **Caddy runs as a
separate container** in the same compose project — it terminates TLS and proxies
to the gateway over the internal Docker network.

> Prefer no Docker? See [`deploy-binary.md`](deploy-binary.md).
> Already have nginx/Cloudflare/a load balancer? Use
> [`deploy-docker-single.md`](deploy-docker-single.md) and bring your own TLS.

## Prereqs

- A host with Docker + the Compose plugin, and a public IP
- A domain with a wildcard DNS record: `*.tela.example.com → <server IP>`
- Ports 80 and 443 open

## 1. Configure

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway/deploy
cp .env.example .env
$EDITOR .env        # set TELA_HOST_SUFFIX and ACME_EMAIL to your domain/email
```

Edit `deploy/Caddyfile` and replace every `example.com` with your domain (keep
it consistent with `TELA_HOST_SUFFIX` in `.env`).

## 2. Run

```bash
docker compose up -d
docker compose logs -f
```

The gateway image is built from the repo `Dockerfile` on first `up`.

## 3. Verify

```bash
curl -s https://tela.example.com/health
curl -sI https://derobeats.tela.example.com/
```

The first request to each subdomain triggers a Let's Encrypt certificate
issuance (on-demand TLS) and an app clone, so it can take a few seconds; repeat
requests are fast.

## How TLS works here

`deploy/Caddyfile` uses **on-demand TLS**: Caddy issues a certificate the first
time each `<app>.tela.example.com` is requested (HTTP-01 challenge), asking the
gateway's `/health` endpoint whether the host is allowed. No DNS-provider plugin
or wildcard cert is required, so the stock `caddy:2` image works.

If you'd rather use a single **wildcard certificate** (`*.tela.example.com`),
you need a DNS-provider build of Caddy and API credentials — see
[`reverse-proxies.md`](reverse-proxies.md#caddy-wildcard).

## Ops

```bash
docker compose ps
docker compose logs -f gateway
docker compose restart gateway

# update to latest code
git pull
docker compose up -d --build
```

Persistent volumes:

- `gateway-data` — the TELA app clone cache
- `caddy-data` — issued certificates (don't delete, or you'll re-issue and may
  hit Let's Encrypt rate limits)

## Using your own DERO node

Set `DERO_DAEMON_URLS` in `.env`, listing your node first:

```dotenv
DERO_DAEMON_URLS=192.168.1.10:10102,node.derofoundation.org:11012
```
