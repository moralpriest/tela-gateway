# Deploy: Google Cloud Run

Cloud Run runs the container image directly, scales to zero when idle, and
manages TLS for you (including custom domains). No load balancer or CDN
required. Great for low, bursty traffic.

## Prereqs

- `gcloud` CLI authenticated (`gcloud auth login`) and a project selected
  (`gcloud config set project <PROJECT_ID>`)
- Cloud Run + Artifact Registry APIs enabled
- A domain you can add DNS records to

## 1. Build & deploy from source

Cloud Run can build the image for you using the repo `Dockerfile`:

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway

gcloud run deploy tela-gateway \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 1Gi \
  --timeout 300 \
  --set-env-vars TELA_HOST_SUFFIX=.tela.example.com,DERO_DAEMON_URLS=node.derofoundation.org:11012
```

The command prints a `https://tela-gateway-xxxx-uc.a.run.app` URL. Test it:

```bash
curl -s https://tela-gateway-xxxx-uc.a.run.app/health
curl -sI https://tela-gateway-xxxx-uc.a.run.app/scid/<64-hex>/
```

## 2. Custom domain + per-app subdomains

Map your wildcard domain to the service. Cloud Run **preserves the `Host`
header**, so host-based routing works without extra config.

```bash
gcloud beta run domain-mappings create \
  --service tela-gateway \
  --domain "*.tela.example.com" \
  --region us-central1
```

The command outputs DNS records to add at your registrar (a CNAME/A record set).
Add them, wait for propagation, and Cloud Run provisions a managed certificate.

> Wildcard domain mappings require domain verification in Google Search Console.
> If wildcards aren't available in your setup, either map specific subdomains
> (`derobeats.tela.example.com`, `explorer.tela.example.com`, …) individually,
> or put Cloudflare in front (see
> [`reverse-proxies.md#cloudflare`](reverse-proxies.md#cloudflare)) and use path
> routing / a small set of mapped hosts.

## 3. Verify

```bash
curl -sI https://derobeats.tela.example.com/
```

## Notes

- **Scale to zero**: idle instances are shut down, so the first request after
  idle pays a cold start + app clone. Set `--min-instances 1` to keep one warm.
- **Memory**: 1Gi is comfortable; the app clone cache lives on the instance's
  ephemeral disk and is lost when the instance is recycled.
- Update by re-running the same `gcloud run deploy --source .` command.
