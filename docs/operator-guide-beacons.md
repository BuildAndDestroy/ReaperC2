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
Optional UUID of an **upstream** beacon for a **pivot chain**. Topology draws a parent → child edge toward C2. A parent does **not** by itself add `-proxy`. The child’s dial path comes from **Pivot proxy** or Scythe Http **Proxy** (see **Initial proxy** below).

**Pivot proxy `host:port`**  
Address of the upstream beacon’s SOCKS5 listener, as the **child** can reach it (for example `10.0.1.20:9050`). Copied into Scythe `-proxy` only when a parent is set and Scythe Http **Proxy** is empty. If both are empty, the server uses environment `BEACON_PIVOT_PROXY` when that variable is set. The k3s deployment does not set it; put the address on the form for each hop.

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
| Proxy | `-proxy`. If this is set, it wins over **Pivot proxy**. Leave it empty on a child so **Pivot proxy** is applied. |
| SOCKS5 listener | `-socks5-listen` / `-socks5-port`. The checkbox must be on; the port field alone does nothing. UI default port is **9050**. |
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

## Initial proxy (pivot)

A pivot is two beacons. The **upstream** beacon phones home directly and also listens as a SOCKS5 proxy. Each **downstream** beacon dials the public C2 URL through that proxy.

`-proxy` and `-socks5-listen` are compiled into **Scythe.embedded**. **Saved profiles → Scythe.embedded** rebuilds the options saved on that profile. To turn a listener on, or to point a child at a different address, generate a new beacon and download again.

### 1. Upstream beacon (the proxy)

On **Generate beacon**:

| Field | Value |
|-------|--------|
| Parent beacon ClientId | Empty |
| Pivot proxy | Empty |
| Beacon C2 base URL | Public beacon origin (ingress or `:8080`), not the admin UI |
| Scythe Http → SOCKS5 listener | **Checked** |
| SOCKS5 listen port | Port the child will connect to (form default **9050**) |
| Scythe Http → Proxy | Empty |

Download **Scythe.embedded** and start it with `TERM_HARVEST=9`. On startup it phones home as usual and listens on **all interfaces** at `:<port>` (`net.Listen("tcp", ":9050")`). The listener accepts SOCKS5 with **no authentication**.

Confirm the beacon is green on **Topology** before you build children. From a host that should pivot through it, the address you will type later is that host’s route to the upstream machine plus the listen port (`10.0.1.20:9050`). That is not the public C2 hostname, and it is not `127.0.0.1` unless the child runs on the same machine.

### 2. Downstream beacon (through the proxy)

The upstream client must already exist in this engagement.

| Field | Value |
|-------|--------|
| Parent beacon ClientId | Upstream beacon’s ClientId |
| Pivot proxy | `<upstream-ip>:<socks-port>` reachable **from the child** |
| Beacon C2 base URL | Same public origin the upstream uses |
| Scythe Http → Proxy | Empty (a value here replaces Pivot proxy) |
| SOCKS5 listener | Off, unless this host should also proxy a further hop |

Generate and download a new embedded binary. The child still requests `https://<c2>/heartbeat/<its-client-id>`. `-proxy` only changes the TCP dial: Scythe opens a SOCKS5 connection to the pivot address, then asks that proxy to connect to the C2 host.

Topology draws the edge from **Parent beacon ClientId** alone. If Pivot proxy is missing, or nothing is listening at that address, the graph can still show the link while the child dials C2 directly (no `-proxy`) or fails to connect (bad `-proxy`).

### Further hops

Repeat step 2. Each new beacon’s parent is the beacon **one hop closer to C2**, and its pivot proxy is that parent’s SOCKS5 address. Check **SOCKS5 listener** on any beacon that should accept the next hop.

`BEACON_PIVOT_PROXY` on the ReaperC2 process is only a fallback when a new beacon has a parent and both proxy fields are empty. One server-wide address is usually wrong once you have more than one hop, so set **Pivot proxy** on the form.

## Beacon troubleshooting

| Issue | Check |
|-------|--------|
| Implant cannot connect | `BEACON_PUBLIC_BASE_URL` / per-beacon base URL points at **8080** (or your public ingress to it), not admin **8443** |
| Embedded won’t start | `TERM_HARVEST=9` set in the same shell/session |
| Embedded build fails | Go installed on server; Scythe sources at `third_party/Scythe` or `REAPERC2_ROOT` |
| No beacons on Commands page | Generate under **Beacons** for the **active** engagement |
| Topology all gray | Beacon never checked in, or interval much shorter than actual sleep |
| Child does not use the pivot | Parent ClientId was set but **Pivot proxy** and Scythe Http **Proxy** were both empty, and `BEACON_PIVOT_PROXY` is unset. Generate again with **Pivot proxy** set to the upstream `host:port`. |
| Child cannot connect through the pivot | Upstream was built without **SOCKS5 listener** checked, the port does not match, or the child cannot route to that `host:port`. The listener binds all interfaces and has no SOCKS username. |
| **`tls: failed to verify certificate: x509: certificate signed by unknown authority`** (Scythe `Http` logs `[-] Error: request failed: Get "https://…/heartbeat"`) | The beacon host does **not** trust the **issuer** of the certificate it received. Common causes: (1) **Split DNS / internal VIP** — the hostname resolves to an **internal** load balancer that presents a **different** cert (corporate CA, self-signed, or old staging) while the public Internet sees Let’s Encrypt. Compare `dig` / `curl -v` from the **beacon machine** vs your laptop. (2) **TLS interception** (corporate proxy) — trust the proxy root or use lab-only `-skip-tls-verify`. (3) **Stale or custom-built Scythe** — rebuild **Scythe.embedded** (or CLI Scythe) with a **current Go** (repo uses Go 1.24); very old runtimes or minimal containers **without `ca-certificates`** can fail verification. (4) **Let’s Encrypt staging** still in front of some paths — staging chains use CAs that are not in default trust stores. |

### TLS: quick checks (run on the **same machine** that runs the failing Scythe)

```bash
# What DNS and cert does this host actually see?
curl -vI "https://YOUR_BEACON_HOST/heartbeat" 2>&1 | sed -n '1,30p'
echo | openssl s_client -connect YOUR_BEACON_HOST:443 -servername YOUR_BEACON_HOST 2>&1 | openssl x509 -noout -issuer -subject
```

If `openssl` shows a **corporate** issuer or **Fake LE** / staging, fix DNS or ingress before chasing ReaperC2 bugs.

**Lab only:** Scythe `Http` supports **`-skip-tls-verify`** (see Scythe Http options in the UI). Do not use in production engagements.
