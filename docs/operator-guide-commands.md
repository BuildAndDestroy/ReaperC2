# Commands

**Path:** `/commands`  
**Requires:** Active engagement

Queue tasks for beacons in the current engagement and review pending work, staged files, downloads, and output history.

## Shell / Scythe command

1. Select a **beacon** (label + Client ID).
2. Optionally pick a **preset**: `whoami`, `groups`, `environment`, `kube-auth-can-i-list`, or `download` (appends `download ` for a host path).
3. Enter a **command**:
   - Plain text → queued as a string (e.g. `whoami`, `download C:\Windows\Temp\file.txt`).
   - JSON object → Scythe structured ops (`command_obj`), e.g. upload metadata.
4. **Queue command** — `POST /api/beacon-commands`.

Pending commands are delivered on the beacon’s next **`GET /heartbeat/<uuid>`**.

## Upload file to beacon

1. Choose a local file → **Stage on server** (`POST /api/beacon-staging`) → receive a staging ID.
2. Set **remote path** on the target (trailing `/` or `\` uses the staged file name).
3. **Queue upload** — sends JSON such as `upload` with `staging_id` and `remote_path`.

## Pending queue

Shows all engagement beacons and commands waiting for the next heartbeat. **Refresh** reloads from `GET /api/beacon-commands`.

## Files

Lists **file artifacts**: operator-staged uploads and files **downloaded from beacons** via the `download` built-in. Stored under `REAPER_ARTIFACT_DIR` (default `./data/reaper_artifacts`). Download via `GET /api/beacon-artifacts/{id}/file`.

## Output history

Per selected beacon: stored command output from the `data` collection (same rows for **every operator** on this engagement — not tied to who queued the command). **Load history** / **Refresh** call the commands page API; the list also **reloads when you change the beacon** in the dropdown.
