# Deploy: single Docker container (bring your own TLS)

Run just the gateway container and let something else handle TLS — an existing
nginx/Caddy/Traefik, a Cloudflare tunnel/proxy, or a cloud load balancer. Good
when you already have a reverse proxy or edge network.

## Build & run

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway
docker build -t tela-gateway:local .
docker run --rm -p 8080:8080 \
  -e TELA_HOST_SUFFIX=.tela.example.com \
  -e DERO_DAEMON_URLS=node.derofoundation.org:11012 \
  tela-gateway:local
```

`curl -s localhost:8080/health` should return JSON.

The image is a static binary on `scratch`, runs as UID 65532, and exposes port
8080.

## Reaching a host-run DERO node

To talk to a `derod` running on the Docker host (e.g. Engram), add
`host.docker.internal`:

```bash
docker run --rm -p 8080:8080 \
  --add-host=host.docker.internal:host-gateway \
  -e DERO_DAEMON_URLS=host.docker.internal:10102,node.derofoundation.org:11012 \
  tela-gateway:local
```

On Docker Desktop (macOS/Windows) `host.docker.internal` resolves without the
flag. The image's default `DERO_DAEMON_URLS` already lists it first, then public
nodes for automatic failover.

## Putting TLS in front

The container serves plain HTTP. Whatever you place in front **must preserve the
`Host` header** so host-based routing works. Examples:

- **nginx**: `proxy_set_header Host $host;` →
  [`reverse-proxies.md#nginx`](reverse-proxies.md#nginx)
- **Traefik**: passes Host by default →
  [`reverse-proxies.md#traefik`](reverse-proxies.md#traefik)
- **Cloudflare proxy**: preserves Host; TLS terminates at the edge →
  [`reverse-proxies.md#cloudflare`](reverse-proxies.md#cloudflare)

If you don't need host-based subdomains, you can skip all of that and just use
path routing: `https://yourhost/scid/<64-hex>/`.

## Persisting the clone cache

Mount a volume at `/data` so the app clone cache survives restarts:

```bash
docker run -d --name tela-gateway -p 8080:8080 \
  -v tela-gateway-data:/data \
  -e TELA_HOST_SUFFIX=.tela.example.com \
  tela-gateway:local
```
