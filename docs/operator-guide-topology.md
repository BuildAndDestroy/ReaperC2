# Topology

**Path:** `/topology`  
**Requires:** Active engagement

Interactive **graph** of C2 → beacons (and parent → child when `ParentClientId` is set). Data from `GET /api/topology`.

The edge is the parent UUID stored at generation. Traffic follows that path only when the upstream beacon was built with a **SOCKS5 listener** and the child was built with **Pivot proxy** set to that listener. See [Initial proxy](/documentation/operator-guide-beacons) on the Beacons page.

| Color | Meaning |
|-------|---------|
| **Blue** | C2 server node |
| **Green** | Beacon on time (within expected interval) |
| **Yellow** | Late (missed expected interval) |
| **Gray** | Offline / stale, or reference node |

Drag to rearrange, scroll to zoom, hover for details. Arrows point along the path **toward C2**. **Refresh** reloads layout and status; **Export PNG** saves the canvas.

Set **Expected phone-home interval** on **Beacons** so late/offline coloring matches your operational cadence.
