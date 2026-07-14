# syntax=docker/dockerfile:1.7
# Compile inside Docker. Prefer a host-prepared vendor/ tree (`make vendor`) for offline
# builds; if vendor/ is missing (typical CI), go mod download runs during the image build.
# On Apple Silicon, prefer `make build` (cross-compiles on the host, then Dockerfile.pack).
FROM golang:1.24-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH
ARG SCYTHE_GIT_REF=main
ARG SCYTHE_REPO=https://github.com/BuildAndDestroy/Scythe.git

ENV CGO_ENABLED=0 \
    GOTELEMETRY=off

WORKDIR /src

RUN apt-get update \
  && apt-get install -y --no-install-recommends git ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
COPY . .

RUN set -eux; \
  if [ ! -f third_party/Scythe/go.mod ]; then \
    mkdir -p third_party; \
    rm -rf third_party/Scythe; \
    git clone --depth 1 --branch "${SCYTHE_GIT_REF}" "${SCYTHE_REPO}" third_party/Scythe; \
  fi; \
  test -f third_party/Scythe/go.mod

WORKDIR /src/cmd
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    if [ -d /src/vendor ] && [ -f /src/vendor/modules.txt ]; then \
      GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
        go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/ReaperC2 .; \
    else \
      (cd /src && go mod download); \
      GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
        go build -mod=mod -trimpath -ldflags="-s -w" -o /out/ReaperC2 .; \
    fi

FROM golang:1.24-bookworm

COPY --from=builder /src /root
COPY --from=builder /out/ReaperC2 /root/cmd/ReaperC2

WORKDIR /root

ARG DEPLOY_ENV=ONPREM
ENV DEPLOY_ENV=${DEPLOY_ENV}

EXPOSE 8080 8443
ENTRYPOINT ["/root/cmd/ReaperC2"]
