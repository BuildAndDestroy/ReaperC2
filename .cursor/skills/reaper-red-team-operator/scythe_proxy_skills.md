# Scythe proxy (SOCKS pivot)

Use this whenever the operator asks how to pivot, set a proxy, chain beacons, or phone home through an already-landed host. Give the exact **Generate beacon** field values. Do not tell them to edit a running binary or to rebuild a saved profile to change these flags.

A pivot is two beacons. The upstream beacon phones home directly and listens as a SOCKS5 proxy. Each downstream beacon still targets the public C2 URL; `-proxy` only changes the TCP dial.

`-socks5-listen`, `-socks5-port`, and `-proxy` are compiled into **Scythe.embedded**. **Saved profiles → Scythe.embedded** rebuilds the options already stored on that profile. To turn a listener on, or to point a child at a different address, generate a new beacon and download again.

## What the server actually does

- **Parent beacon ClientId** draws the Topology edge. It does not add `-proxy`.
- **Pivot proxy** (`host:port`) is copied into Scythe `-proxy` only when a parent is set and Scythe Http **Proxy** is empty. After save, both **Pivot proxy** and **Scythe HTTP proxy** show that same `host:port`.
- If Scythe Http **Proxy** is filled in, that value wins and **Pivot proxy** is ignored for the dial.
- `BEACON_PIVOT_PROXY` on the server is a fallback only when a new beacon has a parent and both proxy fields are empty. The k3s deployment does not set it. Put the address on the form.
- **SOCKS5 listener** must be checked. The port box alone does nothing. The form still stores port **9050** when the box is unchecked; the binary omits `-socks5-listen` unless the box was checked.
- The listener binds all interfaces (`:9050`) and accepts SOCKS5 with no username or password.
- **Beacon C2 base URL** stays the public beacon origin on every hop (ingress or port 8080). It is not the admin UI on 8443, and it is not the pivot address.
- HTTP client timeout (`-timeout`, these profiles use `15s`) is not the phone-home interval (these profiles use 120 seconds).

## 1. Upstream beacon (the proxy)

Generate this first. It must check in before any child is built. Match this shape (the engagement profile **Initial Access - pivot**):

| Field | Value |
|-------|--------|
| Display label | Name the landing host, for example `Initial Access - pivot` |
| Parent beacon ClientId | Empty |
| Pivot proxy | Empty |
| Beacon C2 base URL | Public beacon origin, for example `https://metrics.harvestrangelabs.com` |
| Expected phone-home interval | Operator cadence (120 on the saved profile) |
| HTTP method | `GET` |
| HTTP client timeout | `15s` unless the operator needs a different client timeout |
| Extra headers | Optional. A browser User-Agent is fine and does not affect the proxy |
| Extra directories | Empty (the server always adds `/heartbeat/<ClientId>,/heartbeat`) |
| Proxy | Empty |
| SOCKS5 listener | **Checked** |
| SOCKS5 listen port | **9050** unless this host already uses that port |
| Skip TLS verify | Off |
| GOOS / GOARCH | The landing host (`linux` / `amd64` on the saved profile) |

Download **Scythe.embedded**. The example line ends with `-socks5-listen -socks5-port 9050` and has no `-proxy`.

Start it with `TERM_HARVEST=9` in the same shell. Without that variable the process prints random hex and exits, and it never listens.

Confirm it is green on **Topology**. The address children will use is the upstream host’s IP **as the child can route to it**, plus `9050`. That is not the public C2 hostname. Use `127.0.0.1` only when the child runs on the same machine.

## 2. Downstream beacon (through the proxy)

The upstream client must already exist in this engagement. Match this shape (the engagement profile **pivot-through-proxy**):

| Field | Value |
|-------|--------|
| Display label | Name the hop, for example `pivot-through-proxy` |
| Parent beacon ClientId | Upstream beacon’s ClientId |
| Pivot proxy | `<upstream-ip>:9050` reachable from the child. The saved child uses the upstream host on the child’s network, port **9050** |
| Beacon C2 base URL | Same public origin as the upstream beacon |
| HTTP method | `GET` |
| HTTP client timeout | `15s` |
| Extra headers | Empty unless the operator wants them |
| Proxy | **Empty.** Do not type the address here; **Pivot proxy** is what the server copies into `-proxy` |
| SOCKS5 listener | **Unchecked**, unless this host should accept a further hop |
| SOCKS5 listen port | Leave the default. It is not compiled in while the box is unchecked |
| Skip TLS verify | Off |
| GOOS / GOARCH | The child host |

Download a new embedded binary. The example line includes `-proxy <upstream-ip>:9050` and does not include `-socks5-listen`.

The child still requests `https://<public-c2>/heartbeat/<its-client-id>`. Scythe opens SOCKS5 to the pivot address, then asks that proxy to connect to the C2 host.

Start it with `TERM_HARVEST=9`.

## 3. Further hops

Repeat step 2. Each new beacon’s parent is the beacon one hop closer to C2. Its **Pivot proxy** is that parent’s SOCKS address (`<that-host-ip>:9050`). Check **SOCKS5 listener** only on a beacon that should accept the next hop.

## How to tell a profile is wrong

Read **Saved profiles → View credentials** before telling the operator to run a binary.

| Saved state | What to say |
|-------------|-------------|
| Label looks like an initial beacon, **SOCKS5 listener** is false | It is not a proxy. The earlier pattern **Initial Access no pivot** was built this way. Generate a new upstream with the box checked. |
| Child has a parent but **Pivot proxy** is empty and the example has no `-proxy` | It dials C2 directly. Generate again with **Pivot proxy** set. Topology can still draw the edge. |
| Child **Proxy** was typed and does not match the upstream listen port | That typed value is what got compiled. Generate again; leave **Proxy** empty and set **Pivot proxy** to `<upstream-ip>:9050`. |
| Operator wants to “just rebuild” Scythe.embedded on an old profile | Rebuild keeps the saved flags. Generate a new beacon. |

## Launch

```bash
export TERM_HARVEST=9 && ./Scythe-embedded-<client-prefix>-linux-amd64.bin
```

```powershell
$env:TERM_HARVEST='9'; .\Scythe-embedded-<client-prefix>-windows-amd64.exe
```
