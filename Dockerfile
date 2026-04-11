FROM golang:1.23.1
COPY . /root/

# Scythe sources: prefer the git submodule copied in from the host. If missing (CI tarball, fresh clone
# without submodule init), clone from GitHub so `docker build` does not require a prior submodule step.
# For the exact commit pinned in this repo, run `git submodule update --init` before build so COPY
# includes third_party/Scythe; otherwise the clone uses SCYTHE_GIT_REF (default: main).
ARG SCYTHE_GIT_REF=main
ARG SCYTHE_REPO=https://github.com/BuildAndDestroy/Scythe.git
RUN set -eux; \
  if [ ! -f /root/third_party/Scythe/go.mod ]; then \
    mkdir -p /root/third_party; \
    rm -rf /root/third_party/Scythe; \
    git clone --depth 1 --branch "${SCYTHE_GIT_REF}" "${SCYTHE_REPO}" /root/third_party/Scythe; \
  fi; \
  test -f /root/third_party/Scythe/go.mod

WORKDIR /root/cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ReaperC2
# Process cwd must be repo root so runtime `go build` for Scythe.embedded finds third_party/Scythe (see pkg/scythebuild).
WORKDIR /root
ENV DEPLOY_ENV="ONPREM"
EXPOSE 8080 8443
ENTRYPOINT [ "/root/cmd/ReaperC2" ]
