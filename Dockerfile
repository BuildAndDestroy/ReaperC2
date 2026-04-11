FROM golang:1.23.1
COPY . /root/
# Embedded Scythe builds need sources at runtime; fail the image build if submodule was not checked out.
RUN test -f /root/third_party/Scythe/go.mod || (echo "Missing third_party/Scythe — run: git submodule update --init" >&2; exit 1)
WORKDIR /root/cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ReaperC2
# Process cwd must be repo root so runtime `go build` for Scythe.embedded finds third_party/Scythe (see pkg/scythebuild).
WORKDIR /root
ENV DEPLOY_ENV="ONPREM"
EXPOSE 8080 8443
ENTRYPOINT [ "/root/cmd/ReaperC2" ]
