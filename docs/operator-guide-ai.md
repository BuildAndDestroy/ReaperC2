# Operator AI

**Path:** right-side **Operator AI** panel on every admin page (collapsed tab on the screen edge), or `/ai` to open it  
**Requires:** Active engagement for chat (panel still visible elsewhere with a prompt to select an engagement)

In-dashboard **red team operator assistant**. Pick **Auto** or a specific model from the dropdown (OpenAI, Anthropic, AWS Bedrock, Ollama, etc.). The server merges **`SKILLS.md`** and skill playbooks under [`.cursor/skills/reaper-red-team-operator/`](https://github.com/BuildAndDestroy/ReaperC2/tree/main/.cursor/skills/reaper-red-team-operator) (`red_team_operator_skills.md`, `mitre_attck_skills.md`, and any other `*_skills.md`) when `REAPERC2_ROOT` points at the repo (Docker: `/root`). Set **`REAPER_AI_SKILLS_FILE`** to use one file only. Rebuild/restart after adding skills.

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

**Live discovery (default when the provider is configured):**

| Provider | Discovery | Env to disable |
|----------|-----------|----------------|
| Ollama | `GET /api/tags` (`ollama list`) | `REAPER_AI_OLLAMA_DISCOVER=0` |
| OpenAI | `GET /v1/models` (chat models only) | `REAPER_AI_OPENAI_DISCOVER=0` |
| Anthropic | `GET /v1/models` | `REAPER_AI_ANTHROPIC_DISCOVER=0` |
| Azure AI Foundry | `GET /openai/v1/models` (OpenAI-compatible) | `REAPER_AI_FOUNDRY_DISCOVER=0` |

Discovered models are **merged** with curated latest IDs (`gpt-5.5`, `claude-opus-4-7`, etc.) and any `REAPER_AI_*_MODELS` list. They appear in the Operator AI dropdown even when using `REAPER_AI_MODELS`. Bedrock still uses explicit model IDs (enable models in the AWS console).

Optional comma-separated extras or overrides per provider:

| Variable | Example |
|----------|---------|
| `REAPER_AI_OPENAI_MODELS` | `gpt-5.5,gpt-4.1` |
| `REAPER_AI_ANTHROPIC_MODELS` | `claude-opus-4-7,claude-sonnet-4-6` |
| `REAPER_AI_FOUNDRY_MODELS` | `gpt-5.5,gpt-4.1` (deployment names on Azure) |
| `REAPER_AI_OLLAMA_MODELS` | `llama3.2,mistral` |
| `REAPER_AI_BEDROCK_MODELS` | `us.anthropic.claude-opus-4-7,us.anthropic.claude-sonnet-4-6,amazon.nova-lite-v1:0` |
| `REAPER_AI_BEDROCK_INFERENCE_PREFIX` | `us` (optional; auto from region) |

Or a unified base list (discovery still adds live models for OpenAI, Anthropic, Foundry, and Ollama):

```env
REAPER_AI_MODELS=openai:gpt-5.5,anthropic:claude-opus-4-7,foundry:gpt-5.5,bedrock:anthropic.claude-opus-4-7,ollama:llama3.2
```

If discovery is off and no `*_MODELS` is set, built-in defaults include **`gpt-5.5`**, **`claude-opus-4-7`**, and **`anthropic.claude-opus-4-7`** (Bedrock). Selecting a model still requires access on that provider. Legacy single-model vars (`REAPER_AI_OPENAI_MODEL`, etc.) still work.

### OpenAI

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_OPENAI_API_KEY` | API key (required for OpenAI) |
| `REAPER_AI_OPENAI_API_URL` | Base URL (default `https://api.openai.com/v1`) |
| `REAPER_AI_OPENAI_DISCOVER` | Default `1` when an API key is set: list chat models from `GET /v1/models` |

Legacy: `REAPER_AI_API_KEY`, `REAPER_AI_API_URL`, `REAPER_AI_MODEL`.

### Anthropic

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_ANTHROPIC_API_KEY` | API key |
| `REAPER_AI_ANTHROPIC_API_URL` | Default `https://api.anthropic.com/v1` |
| `REAPER_AI_ANTHROPIC_DISCOVER` | Default `1` when an API key is set: list models from `GET /v1/models` |

### Azure AI Foundry

Uses the [OpenAI v1-compatible endpoint](https://learn.microsoft.com/en-us/azure/foundry/foundry-models/concepts/endpoints) on your Foundry / Azure OpenAI resource (`/openai/v1/chat/completions` and `/openai/v1/models`). On Azure, the model field is often your **deployment name** (from discovery or `REAPER_AI_FOUNDRY_MODELS`). Requests use **`max_completion_tokens`** (not `max_tokens`) for this provider so **GPT-5.x** and similar deployments work, and **`temperature` is set to `1`** (Azure rejects other values for some SKUs). Deployments that do not support **chat completions** on this path (e.g. some Claude SKUs) return `api_not_supported` — use **Anthropic** or **AWS Bedrock** in ReaperC2 for those models instead.

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_FOUNDRY_API_KEY` | API key (also reads `AZURE_OPENAI_API_KEY`, `AZURE_AI_INFERENCE_KEY`) |
| `REAPER_AI_FOUNDRY_API_URL` | Resource base (also reads `AZURE_OPENAI_ENDPOINT`); normalized to `…/openai/v1` |
| `REAPER_AI_FOUNDRY_DISCOVER` | Default `1` when key + URL are set |
| `REAPER_AI_FOUNDRY_MODELS` | Comma-separated deployment / model names |
| `REAPER_AI_FOUNDRY_USE_API_KEY_HEADER` | Set `1` for legacy non-v1 Azure endpoints that expect the `api-key` header |

Catalog id prefix: `foundry:` (aliases `azure`, `azure_foundry` in `REAPER_AI_DEFAULT_PROVIDER`).

### AWS Bedrock

Uses the [Bedrock Converse API](https://docs.aws.amazon.com/bedrock/latest/userguide/conversation-inference.html). **Claude Opus/Sonnet 4.x** must use **inference profile** IDs (e.g. `us.anthropic.claude-opus-4-7`, `us.anthropic.claude-sonnet-4-6` in `us-east-1`), not bare `anthropic.claude-*` foundation IDs — otherwise Converse returns `on-demand throughput isn't supported`. Nova and many other models still use foundation IDs (`amazon.nova-lite-v1:0`). ReaperC2 auto-prefixes bare Claude IDs using `REAPER_AI_BEDROCK_INFERENCE_PREFIX` (default derived from region: `us`, `eu`, `jp`, `au`, `global`).

Reasoning models can return **`reasoningContent`** blocks (chain-of-thought) from Converse in addition to or before plain **`text`**. ReaperC2 includes reasoning text in the assistant reply so you do not see a false **`AWS Bedrock: empty message content`** when the model only populated reasoning blocks.

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_BEDROCK_ENABLED` | Set to `1` to enable |
| `REAPER_AI_BEDROCK_REGION` | AWS region (falls back to `AWS_REGION` / `AWS_DEFAULT_REGION`) |
| `REAPER_AI_BEDROCK_MODELS` | Comma-separated Bedrock model IDs |
| `REAPER_AI_BEDROCK_MODEL` | Single default model when `*_MODELS` is unset |
| `REAPER_AI_BEDROCK_API_KEY` | **Bedrock API key** from the AWS console (bearer token). Also accepts `AWS_BEARER_TOKEN_BEDROCK`. This is **not** an IAM access key and does **not** go in `SESSION_TOKEN`. |
| `REAPER_AI_BEDROCK_ACCESS_KEY_ID` | IAM access key ID (`AKIA…`) — only if using IAM keys, not a Bedrock API key |
| `REAPER_AI_BEDROCK_SECRET_ACCESS_KEY` | IAM secret access key (pair with access key ID) |
| `REAPER_AI_BEDROCK_SESSION_TOKEN` | Optional **STS session token** when using **temporary IAM** credentials (three-part AKIA + secret + token). Not used for Bedrock API keys. |
| `REAPER_AI_BEDROCK_USE_IAM` | Set to `1` to use the default AWS credential chain (recommended on **EKS** with IRSA) |

**Which auth should I use?**

- **Bedrock API key** (single string from console → *Generate API key*): set `REAPER_AI_BEDROCK_API_KEY` only.
- **IAM user access keys**: set `ACCESS_KEY_ID` + `SECRET_ACCESS_KEY`; add `SESSION_TOKEN` only if AWS gave you temporary creds.
- **EKS / instance role**: `REAPER_AI_BEDROCK_USE_IAM=1`, no keys.

Legacy env names `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN` are used when the `REAPER_AI_BEDROCK_*` IAM vars are empty.

**EKS:** attach a Bedrock-capable IAM role to the ReaperC2 service account, set `REAPER_AI_BEDROCK_ENABLED=1`, `REAPER_AI_BEDROCK_REGION`, `REAPER_AI_BEDROCK_USE_IAM=1`, and list models — no static keys in the Secret.

### Ollama

| Variable | Purpose |
|----------|---------|
| `REAPER_AI_OLLAMA_ENABLED` | Set to `1` to enable (no API key) |
| `REAPER_AI_OLLAMA_DISCOVER` | Default `1` when Ollama is enabled: populate the UI model dropdown from `GET /api/tags` (live `ollama list`) |
| `REAPER_AI_OLLAMA_MODELS` | Optional comma-separated extra models (merged with discovered tags) |
| `REAPER_AI_OLLAMA_API_URL` | Default `http://127.0.0.1:11434/v1` |

**Docker Compose + Ollama on the host**

| Platform | `REAPER_AI_OLLAMA_API_URL` |
|----------|----------------------------|
| **Mac / Windows** (Docker Desktop) | `http://host.docker.internal:11434/v1` |
| **Linux** + `ollama-host` profile | `http://172.17.0.1:11434/v1` |

Do **not** add `extra_hosts: host.docker.internal:host-gateway` on Mac/Windows — it points `host.docker.internal` at the docker bridge (`172.17.0.1`) and Operator AI gets `EOF` from the socat proxy.

**Do not use `OLLAMA_HOST=0.0.0.0:11434`** if you want to avoid LAN access — that binds on all interfaces.

**Linux only:** use the Compose **`ollama-host` profile** so socat listens on the docker bridge and forwards to host `127.0.0.1:11434`:

1. `COMPOSE_PROFILES=ollama-host`
2. `REAPER_AI_OLLAMA_API_URL=http://172.17.0.1:11434/v1`

The `ollama-proxy` service uses `network_mode: host` and `bind=172.17.0.1` — not exposed on your LAN. **Host CLI** still uses `http://127.0.0.1:11434`.

If `docker0` is not `172.17.0.1`, adjust the `bind=` address in `docker-compose.yml` (`ip -4 addr show docker0`).

List only models you have pulled (`ollama list`); e.g. `gpt-oss:latest` not `gpt-oss` if that tag is missing.

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
