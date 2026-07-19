# syntax=docker/dockerfile:1
#
# General portable image (static Go binary on scratch).
#   docker build -t tela-gateway:local .
#   docker run --rm -p 8080:8080 tela-gateway:local

FROM golang:1.22-bookworm AS build

WORKDIR /src

ENV CGO_ENABLED=0 \
	GOOS=linux \
	GOFLAGS="-trimpath"

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /out/tela-gateway ./cmd/server \
	&& mkdir -p /out/certs /out/etc \
	&& mkdir -p /out/tmp /out/data/datashards/clone \
	&& cp /etc/ssl/certs/ca-certificates.crt /out/certs/ \
	&& printf 'hosts: files dns\n' > /out/etc/nsswitch.conf \
	&& chmod 1777 /out/tmp \
	&& chown -R 65532:65532 /out/data /out/tmp

FROM scratch

COPY --from=build /out/tela-gateway /tela-gateway
COPY --from=build /out/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/etc/nsswitch.conf /etc/nsswitch.conf
COPY --from=build --chown=65532:65532 /out/tmp /tmp
COPY --from=build --chown=65532:65532 /out/data /data

ENV PORT=8080 \
	DERO_DAEMON_URLS=host.docker.internal:10102,node.derofoundation.org:11012,dero.rabidmining.com:10102,dero-node.net:10102,community-pools.mysrv.cloud:10102 \
	SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt \
	TMPDIR=/tmp \
	TELA_DATA_DIR=/data

USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/tela-gateway"]
