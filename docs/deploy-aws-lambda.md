# Deploy: AWS Lambda + CloudFront

A serverless deploy: a pure-Go Lambda (custom runtime `provided.al2023`, no
Docker) behind CloudFront for TLS and per-app subdomains. Light traffic often
fits the AWS free tier (an account and card are still required).

> This is one option, not the only one. For simpler paths see
> [`deploy-docker-compose.md`](deploy-docker-compose.md) (VPS) or
> [`deploy-cloud-run.md`](deploy-cloud-run.md) (scale-to-zero, no CDN needed).

## Prereqs

- AWS CLI configured (`aws sts get-caller-identity` works)
- Region set (`aws configure set region us-east-1`) — CloudFront + ACM certs
  must be in **us-east-1**
- A domain you control (DNS at any registrar; Route 53 not required)

## 1. Deploy the gateway Lambda

```bash
git clone https://github.com/moralpriest/tela-gateway
cd tela-gateway
./scripts/deploy-lambda-zip.sh
```

Build only (produces `dist/fn.zip`):

```bash
./scripts/build-lambda-zip.sh
```

Defaults: **arm64**, 1024 MB, 120 s timeout, 1 GB `/tmp`, public Function URL
(`AuthType=NONE`). Override:

```bash
LAMBDA_ARCH=amd64 AWS_REGION=ca-central-1 ./scripts/deploy-lambda-zip.sh
```

The first request can take **1–3+ minutes** (cold start + cloning app data from
a public node). Subsequent requests are fast until the instance is recycled.

Key env vars set by the deploy script (edit there): `TELA_HOST_SUFFIX`,
`TELA_ALIASES`, `ALIASES_S3_URI`, `DERO_DAEMON_URLS`. On Lambda `DERO_DAEMON_URLS`
lists public nodes only (no localhost).

## 2. Front it with CloudFront (TLS + subdomains)

```bash
# Creates the ACM cert + CloudFront distribution in us-east-1.
# Approve the ACM cert (DNS/email validation), then re-run to finish.
bash scripts/deploy-cloudfront.sh
```

Set `TELA_HOST_SUFFIX=.tela.example.com` on the Lambda (in
`deploy-lambda-zip.sh`). The gateway strips the suffix and resolves the `<app>`
part as an alias.

### The Host-header gotcha (important)

```text
<app>.tela.example.com ──DNS CNAME──▶ CloudFront ──▶ Lambda Function URL
        │                               │
        │ viewer Host forwarded as      │ origin Host = the Function URL's own host
        └── X-Forwarded-Host ───────────┘ (the Function URL rejects any other Host)
```

A Lambda **Function URL rejects any request whose `Host` isn't its own**,
returning `403 {"Message":null}`. CloudFront can't just forward the viewer Host.
So `deploy-cloudfront.sh` attaches a small **CloudFront Function**
(`scripts/cf-function-host-rewrite.js`, viewer-request) that copies the viewer
`Host` into `X-Forwarded-Host`. The gateway prefers `X-Forwarded-Host` over
`Host`, so host-based routing works behind the CDN.

## 3. DNS

At your registrar, create a wildcard CNAME to the CloudFront domain:

```
CNAME  *.tela.example.com  →  dxxxxxxxxxxxxx.cloudfront.net
```

## 4. Verify

```bash
curl -sI https://derobeats.tela.example.com/       # 200 → DeroBeats
curl -s  https://status.tela.example.com/health    # JSON
curl -sI https://typo.tela.example.com/            # 404 "no such TELA app"
```

## Reserved / status subdomains

`status.tela.example.com` is the canonical status URL (reserved, not looked up as
an app). Add more reserved names with `RESERVED_APPS=status,www,...`. A subdomain
that matches the suffix but has no alias returns a 404 page listing known apps.

## Optional: weekly alias indexer

```bash
bash scripts/deploy-indexer-lambda.sh
```

Deploys `cmd/indexer` as a weekly Lambda that scans the chain and writes
`aliases.json` to S3; the gateway loads it on cold start via `ALIASES_S3_URI`.
See [`indexer.md`](indexer.md).
