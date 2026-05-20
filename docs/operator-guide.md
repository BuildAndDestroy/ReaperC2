# Operator guide

This guide describes each page in the ReaperC2 **admin panel** (default `http://127.0.0.1:8443`). The **beacon API** (default `:8080`) is what implants call; operators use the web UI to manage engagements, generate beacons, queue commands, and export reporting data.

Use the **section tabs** above to open Engagements, Beacons, Commands, and the rest of the portal pages.

## Before you start

| Concept | Meaning |
|---------|---------|
| **Engagement** | A workspace (customer assessment) that scopes beacons, commands, topology, notes, chat, and reports. |
| **Active workspace** | The engagement you selected with **Workspace** on the Engagements page. Most nav items require one. |
| **Beacon / client** | One implant identity: a UUID (`ClientId`) plus API secret, stored in MongoDB `clients` and linked to an engagement. |
| **Profile** | A saved generation record in `beacon_profiles` (credentials, Scythe CLI example, Http options) for reuse and exports. |

**Roles**

- **Admin** — Full portal access, including **Users** and **All logs**.
- **Operator** — Beacons, commands, reports, topology, notes, chat, engagement logs; cannot manage portal users.

**Documentation** in the left nav does not require an active workspace. **Engagements** is the home page after login (`/` redirects there).

## Quick reference — which page do I use?

| Goal | Page |
|------|------|
| Start or switch customer workspace | **Engagements** → **Workspace** |
| Create implant credentials / download binary | **Beacons** |
| Run `whoami`, upload file, `download` from host | **Commands** |
| Briefing export / Navigator layer | **Reports** / **Notes & ATT&CK** |
| See pivot chain and liveness | **Topology** |
| Team coordination | **Chat** |
| Investigate what happened | **Engagement logs** (or **All logs** for admins) |
| Add operator account | **Users** (admin) |
| Change password or enable 2FA | **Account** |

## Sections

| Section | Admin path |
|---------|------------|
| [Engagements](/documentation/operator-guide-engagements) | `/engagements` |
| [Beacons](/documentation/operator-guide-beacons) | `/beacons` |
| [Commands](/documentation/operator-guide-commands) | `/commands` |
| [Reports](/documentation/operator-guide-reports) | `/reports` |
| [Topology](/documentation/operator-guide-topology) | `/topology` |
| [Notes & ATT&CK](/documentation/operator-guide-notes) | `/notes` |
| [Chat](/documentation/operator-guide-chat) | `/chat` |
| [Engagement logs](/documentation/operator-guide-engagement-logs) | `/engagement/logs` |
| [All logs](/documentation/operator-guide-all-logs) | `/logs` (admins) |
| [Users](/documentation/operator-guide-users) | `/users` (admins) |
| [Account](/documentation/operator-guide-account) | `/account` |
| [Operator AI](/documentation/operator-guide-ai) | Right-side panel (or `/ai` to open) |
