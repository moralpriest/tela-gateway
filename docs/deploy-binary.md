# Deploy: Go binary (no Docker)

The gateway is a single static Go binary with no runtime dependencies. This is
the lightest way to self-host — no container runtime required.

**You get TLS + custom subdomains by putting a reverse proxy (Caddy or nginx)
in front.** The binary itself only speaks plain HTTP on `$PORT`.

## Prereqs

- A Linux host (a $5 VPS is plenty) with a public IP
- Go 1.24+ to build (only needed on the build machine — you can build elsewhere
  and copy the binary)
- A domain with a wildcard DNS record: `*.tela.example.com → <server IP>`

## 1. Build

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway
CGO_ENABLED=0 go build -ldflags="-s -w" -o tela-gateway ./cmd/server
```

Cross-compile for a different host from your laptop:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o tela-gateway ./cmd/server
scp tela-gateway user@server:/tmp/
```

## 2. Install as a systemd service

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin tela
sudo cp tela-gateway /usr/local/bin/tela-gateway
sudo cp deploy/systemd/tela-gateway.service /etc/systemd/system/
sudoedit /etc/systemd/system/tela-gateway.service   # set TELA_HOST_SUFFIX, daemons
sudo systemctl daemon-reload
sudo systemctl enable --now tela-gateway
systemctl status tela-gateway
curl -s localhost:8080/health
```

The unit runs as an unprivileged `tela` user, keeps its clone cache under
`/var/lib/tela-gateway`, and restarts on failure. See
[`../deploy/systemd/tela-gateway.service`](../deploy/systemd/tela-gateway.service).

## 3. Put a reverse proxy in front (TLS + subdomains)

### Option A — Caddy (simplest, automatic HTTPS)

Install Caddy ([official instructions](https://caddyserver.com/docs/install)),
then use this `/etc/caddy/Caddyfile`:

```caddyfile
{
	email admin@example.com
	on_demand_tls {
		ask http://127.0.0.1:8080/health
	}
}

*.tela.example.com {
	tls admin@example.com {
		on_demand
	}
	reverse_proxy 127.0.0.1:8080
}

tela.example.com {
	reverse_proxy 127.0.0.1:8080
}
```

```bash
sudo systemctl reload caddy
```

Caddy preserves the `Host` header automatically and mints a cert per subdomain
on first hit. Done.

### Option B — nginx + certbot

See [`reverse-proxies.md`](reverse-proxies.md#nginx) for the full nginx config
(the key line is `proxy_set_header Host $host;`) and wildcard cert setup with
certbot's DNS plugin.

## 4. Verify

```bash
curl -s https://tela.example.com/health
curl -sI https://derobeats.tela.example.com/    # 200 → DeroBeats
```

## Ops

```bash
# logs
journalctl -u tela-gateway -f

# update
CGO_ENABLED=0 go build -o tela-gateway ./cmd/server   # rebuild
sudo systemctl stop tela-gateway
sudo cp tela-gateway /usr/local/bin/
sudo systemctl start tela-gateway
```

- **First request per app** clones the TELA app and can take a few seconds
  (longer on a fresh box that still needs to reach a daemon).
- Run your own DERO node and list it first in `DERO_DAEMON_URLS` for the
  fastest, most reliable resolution.
