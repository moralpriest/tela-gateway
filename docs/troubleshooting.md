# Troubleshooting

## `403 {"Message":null}` behind CloudFront + Lambda

The Lambda Function URL rejected the request because the `Host` header wasn't its
own. Attach the CloudFront Function that copies the viewer Host into
`X-Forwarded-Host` — `scripts/deploy-cloudfront.sh` does this. See
[`reverse-proxies.md`](reverse-proxies.md#aws-cloudfront--lambda-function-url).

## `<app>.tela.example.com` shows the landing page instead of the app

The gateway isn't receiving the viewer host. Your reverse proxy is stripping or
rewriting `Host`. Fix it per [`reverse-proxies.md`](reverse-proxies.md) (for
nginx: `proxy_set_header Host $host;`). Confirm with:

```bash
curl -s https://<app>.tela.example.com/health   # "host" field should echo your host
```

## 404 "no such TELA app" for a real app

- The alias isn't known. Add it: `TELA_ALIASES="<app>=<scid>"`, or use
  `/scid/<64-hex>/` directly.
- Typo in the subdomain — the 404 page lists the aliases the gateway knows.
- `TELA_HOST_SUFFIX` doesn't match the domain you're requesting (it must start
  with a dot and equal the part after `<app>`).

## First request is very slow (seconds to minutes)

Expected on a cold instance: it must reach a DERO daemon and clone the app's
files. To reduce it:

- Run/point at a **local DERO node** and list it first in `DERO_DAEMON_URLS`.
- Keep an instance warm (`--min-instances 1` on Cloud Run,
  `min_machines_running = 1` on Fly, provisioned concurrency on Lambda).
- Persist `TELA_DATA_DIR` / `/data` so the clone cache survives restarts.

## All requests fail — no daemon reachable

The gateway probes each endpoint in `DERO_DAEMON_URLS` with `DERO.GetInfo`
(1.5s timeout, 30s cache) and fails over. If all are down, resolution fails.
Check:

```bash
curl -s https://<host>/health    # shows daemon status
```

Add more endpoints (comma-separated) or a node you control.

## Docker container can't reach a daemon on the host

Add `--add-host=host.docker.internal:host-gateway` and use
`host.docker.internal:10102` in `DERO_DAEMON_URLS`. See
[`deploy-docker-single.md`](deploy-docker-single.md#reaching-a-host-run-dero-node).

## Let's Encrypt rate limits (Caddy)

Don't delete the `caddy-data` volume — it stores issued certs. Deleting it forces
re-issuance and can hit Let's Encrypt's weekly limits. Use the staging CA while
testing (`acme_ca https://acme-staging-v02.api.letsencrypt.org/directory` in the
Caddy global block).
