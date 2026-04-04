package adminpanel

import (
	"testing"
	"time"

	"ReaperC2/pkg/dbconnections"
)

func TestBuildTopologyGraph_DirectAndPivot(t *testing.T) {
	clients := []dbconnections.BeaconClientDocument{
		{ClientId: "a1111111-1111-1111-1111-111111111111", ConnectionType: "HTTP", BeaconLabel: "edge-1"},
		{ClientId: "b2222222-2222-2222-2222-222222222222", ConnectionType: "HTTP", ParentClientId: "a1111111-1111-1111-1111-111111111111"},
	}
	g := buildTopologyGraph(clients)
	if len(g.Nodes) < 3 {
		t.Fatalf("expected c2 + 2 beacons, got %d nodes", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(g.Edges))
	}
}

func TestBeaconTopoStatus_Intervals(t *testing.T) {
	ts := time.Now().UTC().Add(-20 * time.Second)
	doc := dbconnections.BeaconClientDocument{
		HeartbeatIntervalSec: 30,
		LastSeenAt:            &ts,
	}
	if beaconTopoStatus(doc, time.Now()) != topoStatusOK {
		t.Fatalf("expected ok when age < interval")
	}
	doc2 := doc
	ts2 := time.Now().UTC().Add(-45 * time.Second)
	doc2.LastSeenAt = &ts2
	if beaconTopoStatus(doc2, time.Now()) != topoStatusLate {
		t.Fatalf("expected late when between 1x and 3x interval (30s hb, 45s ago)")
	}
}
