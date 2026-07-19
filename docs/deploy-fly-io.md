# Deploy: Fly.io

Fly runs the container image close to users and issues TLS certs (including for
custom subdomains) with a single command. Simple CLI, generous free-ish
allowance for small apps.

## Prereqs

- [`flyctl`](https://fly.io/docs/flyctl/install/) installed and logged in
  (`fly auth login`)
- A domain you can add DNS records to

## 1. Launch

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway

fly launch --no-deploy        # generates fly.toml; pick a name/region
```

Edit the generated `fly.toml` so the internal port is 8080 and set env:

```toml
[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0

[env]
  TELA_HOST_SUFFIX = ".tela.example.com"
  DERO_DAEMON_URLS = "node.derofoundation.org:11012,dero.rabidmining.com:10102"
```

Deploy (Fly builds the repo `Dockerfile`):

```bash
fly deploy
fly open
curl -s https://<your-app>.fly.dev/health
```

## 2. Custom domain + per-app subdomains

Fly's proxy **preserves the `Host` header**, so host-based routing works.

Point DNS at your app and request a certificate. Fly supports wildcard certs:

```bash
# DNS: CNAME *.tela.example.com -> <your-app>.fly.dev
fly certs add "*.tela.example.com"
fly certs show "*.tela.example.com"     # shows validation records / status
```

Add the DNS validation records Fly prints (a `_acme-challenge` CNAME for
wildcards), then wait for the cert to go green.

For a single apex/status host too:

```bash
fly certs add tela.example.com
```

## 3. Verify

```bash
curl -sI https://derobeats.tela.example.com/
```

## Notes

- With `auto_stop_machines`/`min_machines_running = 0`, idle machines stop and
  the next request pays a cold start + app clone. Set `min_machines_running = 1`
  to keep one warm.
- Update with `fly deploy` after pulling new code.
