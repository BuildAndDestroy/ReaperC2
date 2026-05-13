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

After login, pick an **Engagement** workspace (required for most operator pages). **Documentation** is available without switching context.

## Operator UI (summary)

| Area | Purpose |
|------|---------|
| **Engagements** | Create and manage engagements; admins assign operators |
| **Beacons** | Generate client profiles, Scythe examples, **Scythe.embedded** download |
| **Commands** | Queue shell / Scythe tasks; view artifacts |
| **Reports** | JSON / CSV exports |
| **Topology** | Beacon graph |
| **Notes & ATT&CK** | Engagement notes and MITRE layer |
| **Chat** | Operator chat room |
| **Users** / **Logs** | Admin-only user and audit views |

Roles: **Admin** (full access) vs **Operator** (no user APIs). See repository README for API details.

## Beacon / Scythe

- Set **Beacon C2 base URL** in the UI (or `BEACON_PUBLIC_BASE_URL`) to the **public** origin beacons use (not `localhost:8443`).
- **Scythe.embedded** requires **Go** on the server host (or container) at runtime; sources resolve via `REAPERC2_ROOT/third_party/Scythe` or submodule path next to the binary.

## Security-related environment variables

| Variable | Notes |
|----------|--------|
| `ADMIN_COOKIE_SECURE` | Set `true` when the admin site is only served over HTTPS |
| `ADMIN_SESSION_TTL_HOURS` | Server-side session lifetime (default `168`) |
| `ADMIN_TOTP_ISSUER` | Label shown in authenticator apps for TOTP |

## Further reading

- [Installation](/documentation/installation)
- [Docker Compose](/documentation/docker-compose)
- [Kubernetes](/documentation/kubernetes)
