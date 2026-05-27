# Docker Compose

The repository root [`docker-compose.yml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/docker-compose.yml) runs **MongoDB 7** and **ReaperC2** on one Docker network: beacon **8080**, admin **8443**, Mongo published on **27017** by default.

## Quick start

1. Copy [`.env.example`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/.env.example) to `.env` and set strong passwords.
2. Recommended helper (initializes the Scythe submodule then builds):

   ```bash
   ./scripts/compose-up.sh
   ```

   Or manually:

   ```bash
   git submodule update --init --recursive   # optional but recommended for pinned Scythe
   docker compose up --build
   ```

3. Open **Admin:** `http://127.0.0.1:8443/login` (or the host port from `ADMIN_HOST_PORT`).
4. First operator: values from `ADMIN_BOOTSTRAP_*` in `.env` when the `operators` collection is empty.

## Scythe sources in the image

- If `third_party/Scythe` is **present** in the build context (after submodule init), the **Dockerfile** uses it.
- If it is **missing**, the Dockerfile **clones** Scythe during `docker build` using `SCYTHE_GIT_REF` (default `main`). Override in compose build args or `.env` for a specific branch/tag.

After changing the submodule pointer, **rebuild** the image and any deployed **Scythe.embedded** binaries you care about.

## Important environment variables (compose)

Compose wires typical values; see `docker-compose.yml` for the full list. Highlights:

| Variable | Role |
|----------|------|
| `MONGO_ROOT_USER` / `MONGO_ROOT_PASSWORD` | Mongo init + ReaperC2 connection (`MONGO_AUTH_SOURCE=admin`) |
| `MONGO_DATABASE` | Application DB name (default `api_db`) |
| `BEACON_PUBLIC_BASE_URL` | Shown in Scythe examples (point at reachable beacon URL) |
| `REAPERC2_ROOT` | Set to `/root` so runtime `go build` for embedded Scythe finds sources |

## Host Ollama

Run **Ollama on the host** (`ollama serve` or the desktop app). In `.env`:

```env
REAPER_AI_OLLAMA_ENABLED=1
REAPER_AI_OLLAMA_API_URL=http://host.docker.internal:11434/v1
REAPER_AI_OLLAMA_MODELS=gpt-oss:latest
```

**Mac / Windows (Docker Desktop):** use `host.docker.internal` as above. Do not add `extra_hosts: host.docker.internal:host-gateway` — it breaks this on Desktop.

**Linux only:** optional `ollama-host` profile runs socat on the docker bridge (`172.17.0.1:11434` → host `127.0.0.1:11434`):

```env
COMPOSE_PROFILES=ollama-host
REAPER_AI_OLLAMA_API_URL=http://172.17.0.1:11434/v1
```

Restart after changes: `docker compose up --build -d`. See [Operator AI](/documentation/operator-guide-ai).

## Volumes

`mongo_data` persists database files across container restarts.

## Next steps

- [Kubernetes](/documentation/kubernetes) for non-Docker production patterns.
- [Usage](/documentation/usage) for operator workflows.
