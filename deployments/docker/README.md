# Docker helpers (optional)

Miscellaneous Docker-related examples for local development. This is **not** the main ReaperC2 stack — use the repo-root [`docker-compose.yml`](../../docker-compose.yml) and [`docs/docker-compose.md`](../../docs/docker-compose.md) for that.

| File | Purpose |
|------|---------|
| [`ollama-docker-proxy.service.example`](ollama-docker-proxy.service.example) | Optional **systemd** unit (Linux) to expose host Ollama to containers via the `docker0` bridge. Alternative to the Compose `ollama-host` profile. See [Operator AI](../../docs/operator-guide-ai.md). |

Do not run the systemd proxy and the Compose ollama-proxy service at the same time — both bind the same bridge address.
