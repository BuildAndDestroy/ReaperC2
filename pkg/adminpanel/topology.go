package adminpanel

import (
	"time"

	"ReaperC2/pkg/dbconnections"
)

const c2NodeID = "c2"

// Topology status for beacons: ok (on time), late (missed interval), offline (stale / never seen).
const (
	topoStatusC2        = "c2"
	topoStatusOK        = "ok"
	topoStatusLate      = "late"
	topoStatusOffline   = "offline"
	topoStatusBeaconRef = "beacon_ref"
)

// TopologyNode is one vertex for the admin topology graph.
type TopologyNode struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	ConnectionType string `json:"connection_type,omitempty"`
}

// TopologyEdge is a directed hop (e.g. C2 → beacon, parent → child).
type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TopologyGraph is returned by /api/topology for rendering.
type TopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// BeaconHealthStatus returns ok | late | offline for topology / presence (same rules at time now).
func BeaconHealthStatus(c dbconnections.BeaconClientDocument, now time.Time) string {
	return beaconTopoStatus(c, now)
}

func beaconTopoStatus(c dbconnections.BeaconClientDocument, now time.Time) string {
	if c.LastSeenAt == nil {
		return topoStatusOffline
	}
	age := now.Sub(*c.LastSeenAt)
	sec := dbconnections.BeaconHeartbeatIntervalSec(c)
	iv := time.Duration(sec) * time.Second
	if age <= iv {
		return topoStatusOK
	}
	// Yellow: past one interval but not yet fully stale (3× interval without contact).
	if age <= 3*iv {
		return topoStatusLate
	}
	return topoStatusOffline
}

func buildTopologyGraph(clients []dbconnections.BeaconClientDocument) TopologyGraph {
	now := time.Now()
	nodes := []TopologyNode{{ID: c2NodeID, Label: "ReaperC2", Type: "c2", Status: topoStatusC2}}
	edges := []TopologyEdge{}

	known := map[string]bool{c2NodeID: true}
	for _, c := range clients {
		known[c.ClientId] = true
	}

	stub := map[string]bool{}
	for _, c := range clients {
		pid := c.ParentClientId
		if pid != "" && !known[pid] && !stub[pid] {
			stub[pid] = true
			nodes = append(nodes, TopologyNode{ID: pid, Label: "(unknown parent)", Type: "beacon_ref", Status: topoStatusBeaconRef})
		}
	}

	for _, c := range clients {
		label := c.BeaconLabel
		if label == "" {
			if len(c.ClientId) > 8 {
				label = c.ClientId[:8] + "…"
			} else {
				label = c.ClientId
			}
		}
		st := beaconTopoStatus(c, now)
		nodes = append(nodes, TopologyNode{
			ID:             c.ClientId,
			Label:          label,
			Type:           "beacon",
			Status:         st,
			ConnectionType: c.ConnectionType,
		})
	}

	for _, c := range clients {
		if c.ParentClientId != "" {
			edges = append(edges, TopologyEdge{From: c.ParentClientId, To: c.ClientId})
		} else {
			edges = append(edges, TopologyEdge{From: c2NodeID, To: c.ClientId})
		}
	}

	return TopologyGraph{Nodes: nodes, Edges: edges}
}
