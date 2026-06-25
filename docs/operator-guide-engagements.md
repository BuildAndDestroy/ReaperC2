# Engagements

**Path:** `/engagements`

Engagements are the top-level container for an operation. Everything under **Beacons**, **Commands**, **Reports**, **Topology**, **Notes & ATT&CK**, **Chat**, and **Engagement logs** is filtered to the **active workspace**.

## Your engagements

Lists **open** engagements you may access (operators see only engagements where they are in **Assigned operators**; admins see all).

| Column | Meaning |
|--------|---------|
| Name | Engagement title |
| Client | Customer / org display name |
| Start / End | Planned dates (reporting) |
| Haul | **Interactive**, **Short Haul**, or **Long Haul** (planning category) |
| Operators | Usernames assigned to the engagement |

**Workspace** — Sets this engagement as the active workspace and takes you to operator pages (banner at top of nav shows the current engagement).

**Manage** — Opens a dialog to change **status** (open/closed), **haul type**, **Slack / Discord room** (operator chat key), **start** and **end** dates (admins only), **engagement name** (admins only), and (admins only) **assigned operators**. Operators can change status, haul, and room name; name and dates are read-only for non-admins. General notes and MITRE fields are on **Notes & ATT&CK**, not in this dialog.

## Archived engagements

Closed engagements appear here. Filter by client name substring, then **Workspace** or **Manage** (re-open by setting status back to **open**).

## Create engagement

Fill name, client, dates, haul type, optional **Slack / Discord room** (used as the chat room key), initial notes, and operator checkboxes. The creator is typically included in assigned operators.
