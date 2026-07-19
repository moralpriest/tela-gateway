# Reverse proxies & the Host header

Host-based routing (`<app>.tela.example.com` → app) only works if the gateway
receives the **viewer's** hostname. Most proxies preserve `Host` by default;
this page is the per-proxy cheatsheet.

The gateway reads the host in this order: **`X-Forwarded-Host`** first, then
`Host`. So a proxy can either preserve `Host` or set `X-Forwarded-Host`.

## Quick reference

| Proxy / edge | Preserves Host? | Automatic HTTPS? | Notes |
| --- | --- | --- | --- |
| Caddy | yes (default) | yes | simplest; on-demand or wildcard certs |
| Traefik | yes (default) | yes | Docker-native via labels |
| nginx | one line | via certbot | `proxy_set_header Host $host;` |
| Cloudflare proxy | yes | yes (edge) | origin can be plain HTTP |
| AWS CloudFront → Lambda URL | **no** | yes | needs the included CF Function |
| AWS CloudFront → ALB/ECS | configurable | yes | whitelist Host in origin request policy |
| GCP Cloud Run | yes | yes | domain mappings |
| Fly.io | yes | yes | `fly certs add` |

---

## Caddy

Preserves `Host` automatically. Minimal config:

```caddyfile
*.tela.example.com {
	reverse_proxy gateway:8080
}
```

### On-demand TLS (no DNS plugin)

Issues a cert per subdomain on first request. Used by
[`../deploy/Caddyfile`](../deploy/Caddyfile):

```caddyfile
{
	email admin@example.com
	on_demand_tls { ask http://gateway:8080/health }
}
*.tela.example.com {
	tls admin@example.com { on_demand }
	reverse_proxy gateway:8080
}
```

<a id="caddy-wildcard"></a>

### Wildcard cert (needs DNS-provider build)

A single `*.tela.example.com` cert requires a Caddy build with a DNS plugin
(e.g. `caddy-dns/cloudflare`) plus API credentials:

```caddyfile
*.tela.example.com {
	tls { dns cloudflare {env.CF_API_TOKEN} }
	reverse_proxy gateway:8080
}
```

Build such an image with `xcaddy` or the `caddy:builder` image.

---

<a id="nginx"></a>

## nginx

The one line that matters is `proxy_set_header Host $host;`.

```nginx
server {
    listen 443 ssl;
    server_name ~^(?<app>.+)\.tela\.example\.com$;

    ssl_certificate     /etc/letsencrypt/live/tela.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/tela.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Wildcard cert with certbot's DNS plugin (HTTP-01 can't do wildcards):

```bash
sudo certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /etc/letsencrypt/cloudflare.ini \
  -d 'tela.example.com' -d '*.tela.example.com'
```

---

<a id="traefik"></a>

## Traefik

Passes `Host` through by default. With Docker labels on the gateway service:

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.tela.rule=HostRegexp(`{app:.+}.tela.example.com`)"
  - "traefik.http.routers.tela.tls=true"
  - "traefik.http.routers.tela.tls.certresolver=le"
  - "traefik.http.services.tela.loadbalancer.server.port=8080"
```

Configure an ACME cert resolver (`le`) with a DNS challenge for wildcard certs.

---

<a id="cloudflare"></a>

## Cloudflare proxy

When a hostname is proxied (orange cloud), Cloudflare terminates TLS at the edge
and **preserves `Host`** to your origin. Your origin can be a plain-HTTP gateway
(behind any of the setups above, or even bare on a port).

1. DNS: `*.tela` (proxied) → your origin IP/hostname.
2. SSL/TLS mode: **Full** (origin has a cert) or **Flexible** (origin plain
   HTTP — fine for a gateway with no secrets).
3. Nothing to configure on the gateway; Host arrives intact.

For wildcard subdomains you need Cloudflare's Advanced Certificate Manager or a
proxied wildcard record on a supported plan.

---

## AWS CloudFront → Lambda Function URL

This is the one case where you must actively rewrite the host. A Lambda Function
URL returns `403 {"Message":null}` for any `Host` but its own, and CloudFront
can't forward the viewer Host directly. The included CloudFront Function
([`../scripts/cf-function-host-rewrite.js`](../scripts/cf-function-host-rewrite.js),
viewer-request) copies the viewer `Host` into `X-Forwarded-Host`, which the
gateway prefers. `scripts/deploy-cloudfront.sh` wires this up automatically. See
[`deploy-aws-lambda.md`](deploy-aws-lambda.md).

## AWS CloudFront → ALB → ECS/Fargate

An ALB accepts any `Host`, so no rewrite is needed — just set the CloudFront
**origin request policy** to forward the `Host` header (or `AllViewer`). This
sidesteps the Function-URL restriction entirely.
