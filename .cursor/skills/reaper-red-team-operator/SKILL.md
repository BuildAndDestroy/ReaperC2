---
name: reaper-red-team-operator
description: >-
  Red team operator workflows for ReaperC2: reconnaissance, enumeration,
  exploitation planning, beacon commands, MITRE ATT&CK mapping, and engagement
  scoping. Use when working in the ReaperC2 repo, admin panel, Scythe beacons,
  or when the user asks for operator tradecraft tied to this C2.
---

# ReaperC2 red team operator

Read and follow:

1. **[SKILLS.md](../../../SKILLS.md)** — ReaperC2 platform map, beacon workflow, authorization.
2. **[red_team_operator_skills.md](red_team_operator_skills.md)** — tradecraft reference (recon, enumeration, cloud, reporting).
3. **[mitre_attck_skills.md](mitre_attck_skills.md)** — MITRE ATT&CK Enterprise matrix reference (tactics, techniques, sub-techniques).

**Operator AI** (`/ai`) merges these from disk when present (`REAPERC2_ROOT`, usually `/root` in Docker). Any other `*_skills.md` in this folder is picked up automatically. Override everything with `REAPER_AI_SKILLS_FILE` for a single custom file.

## Quick triggers

- **Beacons / Scythe / embedded** → see SKILLS.md “Beacon and command workflow” and `docs/operator-guide-beacons.md`
- **Queue commands** → `docs/operator-guide-commands.md`; suggest exact strings for `whoami`, `download`, uploads
- **MITRE / Navigator** → `docs/operator-guide-notes.md`
- **Engagement scope** → `docs/operator-guide-engagements.md`

## Code touchpoints

| Area | Path |
|------|------|
| Operator AI UI + API | `pkg/adminpanel/handlers_ai.go` |
| LLM client | `pkg/ai/` |
| Skill text | `SKILLS.md`, `*_skills.md` under this folder, `pkg/ai/SKILLS.md` (embed fallback) |

When changing Operator AI behavior, update **SKILLS.md** and keep `pkg/ai/SKILLS.md` in sync (embedded at build). Add new playbooks as `something_skills.md` in this directory.
