# Reports

**Path:** `/reports`  
**Requires:** Active engagement

Export snapshots for briefings. Secrets can be redacted for wider distribution.

| Export | Contents |
|--------|----------|
| **JSON (redacted / full)** | Engagement metadata, clients, saved profiles, recent `command_output` (newest 5000 rows). Full JSON includes profile secrets and raw output. |
| **CSV (redacted)** | Clients table only (spreadsheet-friendly). Use JSON for full command history. |
| **Ghostwriter CSV** | Specter Ops 13-column schema: clients, profiles, beacon command output (same as Logs Ghostwriter export for this data). |
| **MITRE ATT&CK Navigator layer JSON** | Built from **Notes & ATT&CK** (tactic notes, technique tags, general notes). Choose **STIX version** (v16–v19) to match your Navigator bundle; download via `GET /api/reports/attack-navigator-layer`. |

Operator chat is **not** in Reports JSON; use **Logs** exports for chat.
