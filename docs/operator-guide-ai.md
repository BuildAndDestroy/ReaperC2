# Operator AI

**Path:** right-side **Operator AI** panel on every admin page (collapsed tab on the screen edge), or `/ai` to open it  
**Requires:** Active engagement for chat (panel still visible elsewhere with a prompt to select an engagement)

In-dashboard **red team operator assistant**. Pick **Auto** or a specific model from the dropdown (OpenAI, Anthropic, Ollama, etc.). The server merges **`SKILLS.md`** and skill playbooks under [`.cursor/skills/reaper-red-team-operator/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/.cursor/skills/reaper-red-team-operator) (`red_team_operator_skills.md`, `mitre_attck_skills.md`, and any other `*_skills.md`) when `REAPERC2_ROOT` points at the repo (Docker: `/root`). Set **`REAPER_AI_SKILLS_FILE`** to use one file only. Rebuild/restart after adding skills.

## Configuration

Set on the ReaperC2 process (see `.env.example`). Configure **provider credentials** plus **one or more models per provider**.

### Shared

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_ENABLED` | Set to `0` to disable all providers |
| `REAPER_AI_DEFAULT_MODEL` | `auto` (default) or a catalog id such as `openai:gpt-4o` (used when UI is Auto) |
| `REAPER_AI_DEFAULT_PROVIDER` | Fallback when Auto has no `REAPER_AI_DEFAULT_MODEL`: first model for this provider |
| `REAPER_AI_MAX_TOKENS` | Max reply tokens (default `2048`) |

### Model catalog

List models as comma-separated names per provider:

| Variable | Example |
|----------|---------|
| `REAPER_AI_OPENAI_MODELS` | `gpt-4o-mini,gpt-4o` |
| `REAPER_AI_ANTHROPIC_MODELS` | `claude-sonnet-4-20250514,claude-3-5-haiku-20241022` |
| `REAPER_AI_OLLAMA_MODELS` | `llama3.2,mistral` |

Or one unified list (overrides per-provider lists):

```env
REAPER_AI_MODELS=openai:gpt-4o-mini,anthropic:claude-sonnet-4-20250514,ollama:llama3.2
```

Legacy single-model vars (`REAPER_AI_OPENAI_MODEL`, etc.) still work and register one model.

### OpenAI

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_OPENAI_API_KEY` | API key (required for OpenAI) |
| `REAPER_AI_OPENAI_API_URL` | Base URL (default `https://api.openai.com/v1`) |

Legacy: `REAPER_AI_API_KEY`, `REAPER_AI_API_URL`, `REAPER_AI_MODEL`.

### Anthropic

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_ANTHROPIC_API_KEY` | API key |
| `REAPER_AI_ANTHROPIC_API_URL` | Default `https://api.anthropic.com/v1` |

### Ollama

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_OLLAMA_ENABLED` | Set to `1` to enable (no API key) |
| `REAPER_AI_OLLAMA_API_URL` | Default `http://127.0.0.1:11434/v1` |

Docker (Linux): set `REAPER_AI_OLLAMA_API_URL=http://host.docker.internal:11434/v1` and ensure `docker-compose.yml` includes `extra_hosts: ["host.docker.internal:host-gateway"]` on the `reaperc2` service (included in this repo).

**Do not use `OLLAMA_HOST=0.0.0.0:11434`** if you want to avoid LAN access — that binds on all interfaces.

Keep **Ollama on the host** at `127.0.0.1:11434` (default systemd). Use the Compose **`ollama-host` profile** to run a tiny socat container that listens only on the docker bridge and forwards to localhost:

1. In `.env`: `COMPOSE_PROFILES=ollama-host` (or run `docker compose --profile ollama-host up`).
2. `REAPER_AI_OLLAMA_API_URL=http://host.docker.internal:11434/v1` (already wired via `extra_hosts` in `docker-compose.yml`).

The `ollama-proxy` service uses `network_mode: host` and `bind=172.17.0.1` — not exposed on your LAN. **Host CLI** still uses `http://127.0.0.1:11434`.

If `docker0` is not `172.17.0.1`, adjust the `bind=` address in `docker-compose.yml` (`ip -4 addr show docker0`).

Optional host systemd alternative: [`deployments/docker/ollama-docker-proxy.service.example`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/docker/ollama-docker-proxy.service.example) (do not run both proxy and the compose service at once).

Restart ReaperC2 after changing env vars.

**Kubernetes:** use ConfigMap `reaperc2-ai-config` and Secret `reaperc2-ai-secrets` from [`deployments/k8s/operator-ai.yaml`](https://github.com/BuildAndDestroy/ReaperC2/blob/main/deployments/k8s/operator-ai.yaml). See [Kubernetes](/documentation/kubernetes#operator-ai-multi-model).

## UI

- **Panel** — click **Operator AI** on the right edge of the window to expand; **»** collapses it. Drag the **left edge** of the panel to resize (300–960px; double-click the edge to reset). Width is remembered in the browser.
- **Model** — **Auto** or any enabled catalog entry (`Provider · model name`).
- **Chat history** — stored in **sessionStorage** per engagement; survives moving between Beacons, Commands, Notes, etc. in the same tab. **Clear** removes it. Switching engagements loads that engagement’s thread.
- **/ai** — opens the panel automatically (shortcut page).

**Auto** uses `REAPER_AI_DEFAULT_MODEL` when set to a catalog id; otherwise the first model for `REAPER_AI_DEFAULT_PROVIDER`; otherwise the first enabled model.

## What the model sees

1. Full **SKILLS.md** operator playbook (system).
2. **Engagement context** — name, client, notes, beacon list, recent command output (truncated).
3. Your **chat history** in the panel (persisted per engagement in the browser tab until **Clear** or the tab closes).

## Audit

Each successful chat turn is logged as `ai_chat` with the **user prompt** and **assistant reply** (readable in Engagement logs / All logs; full text in JSON export). Very long messages are truncated for storage.

## Cursor / external agents

Same skill at repository root **`SKILLS.md`** and **`.cursor/skills/reaper-red-team-operator/`**.
