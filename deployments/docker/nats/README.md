# NATS Docker images

- **`Dockerfile.server`**: runs the **NATS broker** (`nats-server`). Nothing here auto-subscribes; clients connect to it.
- **`Dockerfile.client`**: installs the **`nats` CLI** only. Use this image for `nats pub`, `nats sub`, etc.

`nats sub` is a **client** command. Run it from the **client** container (often a second terminal), not from the broker container.

## Build

```bash
cd deployments/docker/nats

docker build -f Dockerfile.server -t reaperc2-nats-server .
docker build -f Dockerfile.client -t reaperc2-nats-cli .
```

Cross-build the client for another CPU (for example `linux/amd64` from Apple Silicon):

```bash
docker buildx build --platform linux/amd64 -f Dockerfile.client -t reaperc2-nats-cli .
```

The `Dockerfile.client` chooses the NATS CLI zip from `uname -m` inside the build stage, so it matches `--platform`.

## Example: Docker Compose

Use a shared network so the hostname `nats` resolves to the broker.

```yaml
services:
  nats:
    image: reaperc2-nats-server
    ports:
      - "4222:4222"

  nats-cli:
    image: reaperc2-nats-cli
    depends_on:
      - nats
    stdin_open: true
    tty: true
    command: ["sleep", "infinity"]
```

Terminal A (subscribe):

```bash
docker compose exec nats-cli nats sub --server=nats://nats:4222 my.subject
```

Terminal B (publish):

```bash
docker compose exec nats-cli nats pub --server=nats://nats:4222 my.subject "hello from cli"
```

Or run one-off without `sleep`:

```bash
docker compose run --rm nats-cli nats pub --server=nats://nats:4222 my.subject "hello"
```

## Optional: JetStream on the server

Rebuild the server image with a custom command, or override in Compose:

```yaml
services:
  nats:
    image: reaperc2-nats-server
    command: ["nats-server", "-js", "-m", "8222"]
```
