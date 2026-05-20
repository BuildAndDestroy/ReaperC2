# Beacons

**Path:** `/beacons`  
**Requires:** Active engagement

Beacons is where you **register implant identities** against the C2 server, configure how **Scythe** (or compatible HTTP beacons) phone home, download **Scythe.embedded** binaries, and manage saved **profiles**. Generating a beacon creates:

1. A row in MongoDB **`clients`** (what the implant authenticates with on heartbeat).
2. A **`beacon_profiles`** record (always saved) with the full generation metadata for reports and re-downloads.

## Tabs

| Tab | Purpose |
|-----|---------|
| **Generate beacon** | Create a new client + profile in one step |
| **Saved profiles** | List, copy credentials, rebuild embedded binary, kill, or delete profiles |

Use URL hash `#profiles` (or **Refresh profile list**) to jump to saved profiles after a page reload.

## Generate beacon — fields

**Display label**  
Optional friendly name shown on **Topology**, **Commands** beacon picker, and exports. Does not affect authentication.

**Parent beacon ClientId**  
Optional UUID of an **upstream** beacon for a **pivot chain**. When set, Scythe examples include `--proxy` (see pivot proxy). Topology draws parent → child edges toward C2.

**Pivot proxy `host:port`**  
Used in Scythe `--proxy` when a parent is set (unless overridden in Scythe Http **Proxy**). Server default: environment `BEACON_PIVOT_PROXY`.

**Beacon C2 base URL**  
Public origin implants should call (e.g. `https://c2.example.com` or `10.0.0.5:8080`). Saved on the profile for embedded rebuilds. Server default: `BEACON_PUBLIC_BASE_URL`.  
**Important:** This is the **beacon listener** (usually port **8080**), not the admin UI on **8443**.

**Expected phone-home interval (seconds)**  
Operator-defined check-in window (5–86400, default 60). **Topology** uses this for status: **green** = on time, **yellow** = missed one interval, **gray** = offline or unknown.

**Profile name**  
Optional; otherwise auto-named `beacon-xxxxxxxx-YYYYMMDD-hhmmss`. A profile is **always** persisted even if you leave this blank.

## Scythe Http (collapsible)

These options build the example **`Scythe Http …`** command and the embedded compile. They map to Scythe CLI flags.

| Field | Scythe flag / behavior |
|-------|-------------------------|
| HTTP method | `-method` (default `GET`) |
| HTTP client timeout | `-timeout` (e.g. `30s`) — **not** the phone-home interval |
| Request body | `-body` (optional JSON string) |
| Extra directories | Appended after required `/heartbeat/<ClientId>,/heartbeat` |
| Extra headers | Merged after required `Content-Type`, `X-Client-Id`, `X-API-Secret` |
| Proxy | `-proxy`; pivot proxy fills in when parent is set and this is empty |
| SOCKS5 listener | `-socks5-listen` / `-socks5-port` |
| Skip TLS verify | `-skip-tls-verify` |
| GOOS / GOARCH | Target for **Scythe.embedded** (`linux`/`windows`/`darwin`, `amd64`/`arm64`) |

After **Generate beacon**, the JSON response is shown on the page (ClientId, secret, heartbeat URL, Scythe example). Open **Saved profiles** → **View credentials** to copy values after refresh.

## Generate beacon — actions

**Generate beacon** — `POST /api/beacons` with `connection_type: HTTP` and your form values.

**Download Scythe.embedded** (after successful generate) — `POST /api/beacons/scythe-embedded`. The server runs `go build` on vendored Scythe (`third_party/Scythe` or `REAPERC2_ROOT`). Requires **Go on the admin host**. Build often takes 30s–2m; progress is shown while downloading.

**Run embedded binary on the host**  
Embedded Scythe requires environment variable **`TERM_HARVEST=9`** before start (see collapsible help on the page). Examples:

```bash
export TERM_HARVEST=9 && ./Scythe
```

```powershell
$env:TERM_HARVEST='9'; .\Scythe.exe
```

## Saved profiles — table

| Column | Meaning |
|--------|---------|
| Name | Profile label |
| Client ID | Beacon UUID |
| Type | Connection type (e.g. HTTP) |
| Created by | Operator who generated it |

**View credentials** — Client ID, secret, label, parent, pivot proxy, expected interval, embedded target OS/arch, beacon base URL, heartbeat URL, full Scythe example (copy buttons).

**Scythe.embedded** — Rebuild and download using the profile’s **saved** Http options (no need to re-enter the form).

**Kill** — Queues Scythe self-destruct command `sendmetojesusdog` on the next heartbeat (`POST /api/beacon-kill`). Confirm before use.

**Delete** — Removes the **profile** record only (`DELETE /api/beacon-profiles/{id}`). Does not automatically remove the live `clients` row; the implant may still check in until removed or killed.

## How a beacon talks to C2

Typical Scythe HTTP flow (simplified):

1. **GET** `/heartbeat/<ClientId>` (and related paths) with headers `X-Client-Id`, `X-API-Secret`.
2. Response may include a JSON **`Commands`** array (strings and/or objects).
3. Beacon runs tasks and **POST**s output to `/receive/<ClientId>`.

Queue work from **Commands**; results appear in output history and audit logs.

## Beacon troubleshooting

| Issue | Check |
|-------|--------|
| Implant cannot connect | `BEACON_PUBLIC_BASE_URL` / per-beacon base URL points at **8080** (or your public ingress to it), not admin **8443** |
| Embedded won’t start | `TERM_HARVEST=9` set in the same shell/session |
| Embedded build fails | Go installed on server; Scythe sources at `third_party/Scythe` or `REAPERC2_ROOT` |
| No beacons on Commands page | Generate under **Beacons** for the **active** engagement |
| Topology all gray | Beacon never checked in, or interval much shorter than actual sleep |
