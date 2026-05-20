# Usage

## Process model

One **ReaperC2** process listens on:

| Listener | Default | Purpose |
|----------|---------|---------|
| Beacon API | `:8080` | Scythe / implant heartbeats, commands, file staging |
| Admin panel | `:8443` | Operator web UI (`BEACON_ADDR` / `ADMIN_ADDR` override binds) |

Set `ADMIN_DISABLE=1` to run **beacon only** (no web UI on that instance).

## Sign in

Open the admin URL (for example `http://127.0.0.1:8443/login`). Use an account from the `operators` collection or the bootstrap credentials on first run.

After login, open **Engagements** and choose **Workspace** on an engagement. Most operator pages require an **active workspace** (banner in the sidebar). **Documentation** and **Account** work without one.

## Operator UI

For a full walkthrough of every admin page (especially **Beacons**), see **[Operator guide](/documentation/operator-guide)**.

| Area | Purpose |
|------|---------|
| **Engagements** | Create workspaces; assign operators; open/close engagements |
| **Beacons** | Generate clients, Scythe examples, **Scythe.embedded** download, profiles, kill |
| **Commands** | Queue shell / Scythe tasks; stage uploads; view artifacts and output |
| **Reports** | JSON / CSV / Ghostwriter / ATT&CK Navigator layer exports |
| **Topology** | Interactive beacon graph (liveness and pivot chain) |
| **Notes & ATT&CK** | Engagement notes and MITRE Navigator layer source data |
| **Chat** | Operator chat per engagement |
| **Engagement logs** | Audit trail for the active engagement |
| **All logs** | Admin-only global audit + exports |
| **Users** | Admin-only portal accounts |
| **Account** | Password and optional TOTP 2FA |
| **Operator AI** | Red team assistant (right-side panel) with engagement context |

Roles: **Admin** (full access) vs **Operator** (no user APIs). Legacy accounts without `role` are treated as Admin.

## Operator AI

Requires an **active engagement** and at least one configured model. List models with `REAPER_AI_*_MODELS` (comma-separated) or `REAPER_AI_MODELS`. See `.env.example` and [Operator AI](/documentation/operator-guide-ai).

Skill text: repository root **`SKILLS.md`** (also used by Cursor agents).

## Beacon / Scythe

- Set **Beacon C2 base URL** in the UI (or `BEACON_PUBLIC_BASE_URL`) to the **public** origin beacons use (port **8080** / ingress to beacon API — not `localhost:8443` admin).
- **Scythe.embedded** requires **Go** on the server host at runtime; sources resolve via `REAPERC2_ROOT/third_party/Scythe` or submodule path next to the binary.
- Embedded binaries need **`TERM_HARVEST=9`** in the environment before launch (see **Beacons** page in the UI or Operator guide).

## Security-related environment variables

| Variable | Notes |
|----------|--------|
| `ADMIN_COOKIE_SECURE` | Set `true` when the admin site is only served over HTTPS |
| `ADMIN_SESSION_TTL_HOURS` | Server-side session lifetime (default `168`) |
| `ADMIN_TOTP_ISSUER` | Label shown in authenticator apps for TOTP |

## Further reading

- [Operator guide](/documentation/operator-guide) — per-page UI reference
- [Installation](/documentation/installation)
- [Docker Compose](/documentation/docker-compose)
- [Kubernetes](/documentation/kubernetes)
