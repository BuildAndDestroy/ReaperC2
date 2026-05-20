# Topology

**Path:** `/topology`  
**Requires:** Active engagement

Interactive **graph** of C2 → beacons (and parent → child when `ParentClientId` is set). Data from `GET /api/topology`.

| Color | Meaning |
|-------|---------|
| **Blue** | C2 server node |
| **Green** | Beacon on time (within expected interval) |
| **Yellow** | Late (missed expected interval) |
| **Gray** | Offline / stale, or reference node |

Drag to rearrange, scroll to zoom, hover for details. Arrows point along the path **toward C2**. **Refresh** reloads layout and status; **Export PNG** saves the canvas.

Set **Expected phone-home interval** on **Beacons** so late/offline coloring matches your operational cadence.
