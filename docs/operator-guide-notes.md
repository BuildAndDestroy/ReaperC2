# Notes & ATT&CK

**Path:** `/notes`  
**Requires:** Active engagement

Engagement-scoped documentation and MITRE mapping for reporting.

## Engagement notes

Free-form text (scope, handoff, contacts). Not sent to beacons. Saved with **Save** (`PATCH /api/engagements/{id}`).

## MITRE ATT&CK

- **Tactic notes** — One textarea per enterprise tactic (matrix version selector drives tactic/technique menus; v19 adds tactics such as **Stealth** and **Defense Impairment**).
- **Technique tags** — Rows of tactic + technique + optional note. Exported to Navigator as green highlights (`#74c476`) with per-technique comments.
- **Matrix (STIX) version** — Must match the ATT&CK release you use in [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/). **Download Navigator layer JSON** uses the same version for catalog placement (each tagged technique ID is repeated on every matrix tactic column where that ID appears).

Status, haul type, and operator assignment remain under **Engagements → Manage**.
