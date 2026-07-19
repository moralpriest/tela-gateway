# Deploy: other container hosts

The gateway is a standard container listening on port 8080, built from the repo
`Dockerfile`. Any platform that runs a container works. Below are one-liners /
minimal manifests. The universal rules:

- Expose/target **port 8080**.
- Set `TELA_HOST_SUFFIX` and `DERO_DAEMON_URLS` as env vars.
- Whatever terminates TLS in front must **preserve the `Host` header** for
  host-based subdomain routing (see [`reverse-proxies.md`](reverse-proxies.md)).
  If you only use path routing (`/scid/<hex>/`), Host doesn't matter.

## Railway

1. New Project → Deploy from GitHub repo → select your fork.
2. Railway detects the `Dockerfile`.
3. Variables: `TELA_HOST_SUFFIX=.tela.example.com`,
   `DERO_DAEMON_URLS=node.derofoundation.org:11012`.
4. Settings → Networking → expose port 8080, add a custom domain. Railway's edge
   preserves Host.

## Render

1. New → Web Service → connect the repo.
2. Runtime: **Docker**. Render uses the `Dockerfile`.
3. Environment: `PORT` is provided by Render — the gateway already honors
   `$PORT`. Also set `TELA_HOST_SUFFIX` and `DERO_DAEMON_URLS`.
4. Add a custom domain (wildcard supported on paid plans).

## Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tela-gateway
spec:
  replicas: 2
  selector:
    matchLabels: { app: tela-gateway }
  template:
    metadata:
      labels: { app: tela-gateway }
    spec:
      containers:
        - name: tela-gateway
          image: ghcr.io/OWNER/tela-gateway:latest   # or your registry
          ports: [{ containerPort: 8080 }]
          env:
            - { name: TELA_HOST_SUFFIX, value: ".tela.example.com" }
            - { name: DERO_DAEMON_URLS, value: "node.derofoundation.org:11012" }
          readinessProbe:
            httpGet: { path: /health, port: 8080 }
---
apiVersion: v1
kind: Service
metadata:
  name: tela-gateway
spec:
  selector: { app: tela-gateway }
  ports: [{ port: 80, targetPort: 8080 }]
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tela-gateway
  annotations:
    # nginx-ingress preserves Host by default; for others ensure Host pass-through
    cert-manager.io/cluster-issuer: letsencrypt
spec:
  tls:
    - hosts: ["*.tela.example.com"]
      secretName: tela-wildcard-tls
  rules:
    - host: "*.tela.example.com"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: tela-gateway
                port: { number: 80 }
```

> You'll need to build and push the image to a registry yourself (this project
> is source-only and does not publish images). Example:
> `docker build -t ghcr.io/OWNER/tela-gateway:latest . && docker push ...`.

## Nomad / ECS / Docker Swarm

Same pattern: run the image, target port 8080, set the two env vars, put a
Host-preserving proxy/load balancer in front. For AWS specifically, an
ALB → ECS/Fargate setup avoids the Lambda Function URL Host restriction entirely
(the ALB accepts any Host) — see [`deploy-aws-lambda.md`](deploy-aws-lambda.md)
for why that matters.
