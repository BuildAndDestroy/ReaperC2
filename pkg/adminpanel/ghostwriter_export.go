package adminpanel

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const ghostwriterExportDataLimit int64 = 50000

// ghostwriterC2EndpointName is the logical source/destination name for the ReaperC2 server in Ghostwriter CSV (source_ip / dest_ip columns).
const ghostwriterC2EndpointName = "ReaperC2"

var ghostwriterCSVHeader = []string{
	"entry_identifier",
	"start_date",
	"end_date",
	"source_ip",
	"dest_ip",
	"tool",
	"user_context",
	"command",
	"description",
	"output",
	"comments",
	"operator_name",
	"tags",
}

type ghostwriterSortRow struct {
	t   time.Time
	row []string
}

func detailsJSONComments(d bson.M) string {
	if d == nil {
		return ""
	}
	b, err := json.Marshal(d)
	if err != nil {
		return fmt.Sprint(d)
	}
	return string(b)
}

func detailsOutputPreview(d bson.M) string {
	if d == nil {
		return ""
	}
	if s, ok := d["output_preview"].(string); ok {
		return s
	}
	return ""
}

func detailsCommandString(d bson.M) string {
	if d == nil {
		return ""
	}
	if s, ok := d["command"].(string); ok && s != "" {
		return s
	}
	return stringifyCommandsValue(d["commands"])
}

func stringifyCommandsValue(raw interface{}) string {
	if raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case []string:
		return strings.Join(v, "; ")
	case primitive.A:
		var parts []string
		for _, x := range v {
			if s, ok := x.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "; ")
	case []interface{}:
		var parts []string
		for _, x := range v {
			parts = append(parts, dbconnections.StringifyBeaconCommand(x))
		}
		return strings.Join(parts, "; ")
	default:
		return fmt.Sprint(raw)
	}
}

func ghostwriterAuditDescription(action string, d bson.M) string {
	var b strings.Builder
	b.WriteString(action)
	if d == nil {
		return b.String()
	}
	if cid, ok := d["client_id"].(string); ok && cid != "" {
		b.WriteString(" | client_id=")
		b.WriteString(cid)
	}
	if pn, ok := d["profile_name"].(string); ok && pn != "" {
		b.WriteString(" | profile=")
		b.WriteString(pn)
	}
	return b.String()
}

func beaconLabelsFromClients(clients []dbconnections.BeaconClientDocument) map[string]string {
	m := make(map[string]string, len(clients))
	for _, c := range clients {
		if c.BeaconLabel != "" {
			m[c.ClientId] = c.BeaconLabel
		} else {
			m[c.ClientId] = c.ClientId
		}
	}
	return m
}

func ghostwriterLabelForClient(labelByClient map[string]string, clientID string) string {
	if clientID == "" {
		return ""
	}
	if labelByClient != nil {
		if l, ok := labelByClient[clientID]; ok && l != "" {
			return l
		}
	}
	return clientID
}

// C2 → beacon: commands queued/delivered. source_ip = server, dest_ip = beacon display label.
func ghostwriterEndpointsC2ToBeacon(clientID string, labelByClient map[string]string) (src, dst string) {
	label := ghostwriterLabelForClient(labelByClient, clientID)
	if label == "" {
		return ghostwriterC2EndpointName, ghostwriterC2EndpointName
	}
	return ghostwriterC2EndpointName, label
}

// Beacon → C2: command execution result / output received. source_ip = beacon label, dest_ip = server.
func ghostwriterEndpointsBeaconToC2(clientID string, labelByClient map[string]string) (src, dst string) {
	label := ghostwriterLabelForClient(labelByClient, clientID)
	if label == "" {
		return ghostwriterC2EndpointName, ghostwriterC2EndpointName
	}
	return label, ghostwriterC2EndpointName
}

func ghostwriterEndpointsForAudit(action string, d bson.M, labelByClient map[string]string) (src, dst string) {
	var clientID string
	if d != nil {
		if cid, ok := d["client_id"].(string); ok {
			clientID = cid
		}
	}
	switch action {
	case dbconnections.AuditActionBeaconOutputReceived:
		return ghostwriterEndpointsBeaconToC2(clientID, labelByClient)
	case dbconnections.AuditActionBeaconCommandsDelivered,
		dbconnections.AuditActionBeaconCommandQueued,
		dbconnections.AuditActionBeaconKillQueued,
		dbconnections.AuditActionBeaconCreated,
		dbconnections.AuditActionBeaconProfileDel:
		return ghostwriterEndpointsC2ToBeacon(clientID, labelByClient)
	default:
		return ghostwriterC2EndpointName, ghostwriterC2EndpointName
	}
}

func ghostwriterProfileDestinationLabel(p dbconnections.BeaconProfile, labelByClient map[string]string) string {
	l := ghostwriterLabelForClient(labelByClient, p.ClientID)
	if l != p.ClientID {
		return l
	}
	if p.Label != "" {
		return p.Label
	}
	if p.Name != "" {
		return p.Name
	}
	return p.ClientID
}

func ghostwriterRowsFromCommandOutput(data []dbconnections.CommandOutputRecord, labelByClient map[string]string) []ghostwriterSortRow {
	var rows []ghostwriterSortRow
	for _, rec := range data {
		id := "data-" + rec.ID.Hex()
		ts := rec.Timestamp.UTC().Format(time.RFC3339)
		desc := fmt.Sprintf("Beacon command result | client_id=%s", rec.ClientID)
		tags := "reaperc2,beacon-output,ghostwriter"
		src, dst := ghostwriterEndpointsBeaconToC2(rec.ClientID, labelByClient)
		row := []string{
			id,
			ts,
			ts,
			src,
			dst,
			"Scythe",
			"Beacon | command execution result",
			rec.Command,
			desc,
			rec.Output,
			"",
			"beacon",
			tags,
		}
		rows = append(rows, ghostwriterSortRow{t: rec.Timestamp, row: row})
	}
	return rows
}

func ghostwriterUserContext(action string) string {
	switch action {
	case dbconnections.AuditActionBeaconCommandsDelivered:
		return "Beacon | commands delivered on heartbeat"
	case dbconnections.AuditActionBeaconOutputReceived:
		return "Beacon | command output received"
	case dbconnections.AuditActionBeaconCommandQueued:
		return "Operator | command queued"
	case dbconnections.AuditActionBeaconKillQueued:
		return "Operator | kill (self-destruct) queued"
	case dbconnections.AuditActionBeaconCreated:
		return "Operator | beacon created"
	case dbconnections.AuditActionBeaconProfileDel:
		return "Operator | profile deleted"
	case dbconnections.AuditActionReportExported:
		return "Operator | report exported"
	case dbconnections.AuditActionAuditLogExported:
		return "Operator | audit log exported"
	case dbconnections.AuditActionUserCreated:
		return "Admin | user created"
	case dbconnections.AuditActionUserDisabled:
		return "Admin | user disabled"
	case dbconnections.AuditActionUserEnabled:
		return "Admin | user enabled"
	case dbconnections.AuditActionEngagementOperatorsUpdated:
		return "Admin | engagement operators updated"
	default:
		return "ReaperC2 | " + action
	}
}

// WriteGhostwriterCSV writes Specter Ops Ghostwriter–compatible CSV (newest events first): audit entries, beacon data rows, and operator chat.
// labelByClient maps beacon ClientId → display label (BeaconLabel); used for source_ip/dest_ip (may be nil; falls back to client UUID).
func WriteGhostwriterCSV(w io.Writer, audits []dbconnections.AuditLogEntry, data []dbconnections.CommandOutputRecord, chat []dbconnections.ChatMessage, labelByClient map[string]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(ghostwriterCSVHeader); err != nil {
		return err
	}

	var rows []ghostwriterSortRow

	for _, e := range audits {
		id := "audit"
		if !e.ID.IsZero() {
			id = "audit-" + e.ID.Hex()
		}
		ts := e.Time.UTC().Format(time.RFC3339)
		d := e.Details
		cmd := detailsCommandString(d)
		out := detailsOutputPreview(d)
		desc := ghostwriterAuditDescription(e.Action, d)
		comments := detailsJSONComments(d)
		tags := "reaperc2," + e.Action + ",audit"
		src, dst := ghostwriterEndpointsForAudit(e.Action, d, labelByClient)
		row := []string{
			id,
			ts,
			ts,
			src,
			dst,
			"ReaperC2",
			ghostwriterUserContext(e.Action),
			cmd,
			desc,
			out,
			comments,
			e.Actor,
			tags,
		}
		rows = append(rows, ghostwriterSortRow{t: e.Time, row: row})
	}

	rows = append(rows, ghostwriterRowsFromCommandOutput(data, labelByClient)...)

	for _, m := range chat {
		id := "chat-" + m.ID.Hex()
		ts := m.CreatedAt.UTC().Format(time.RFC3339)
		desc := fmt.Sprintf("Operator chat | room=%s", m.Room)
		// Operator message into C2-hosted chat: source = sender, destination = server.
		src, dst := m.Username, ghostwriterC2EndpointName
		if strings.TrimSpace(src) == "" {
			src = ghostwriterC2EndpointName
		}
		row := []string{
			id,
			ts,
			ts,
			src,
			dst,
			"ReaperC2",
			"Operator chat",
			"",
			desc,
			m.Body,
			"",
			m.Username,
			"reaperc2,operator-chat,ghostwriter",
		}
		rows = append(rows, ghostwriterSortRow{t: m.CreatedAt, row: row})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].t.After(rows[j].t)
	})

	for _, r := range rows {
		if err := cw.Write(r.row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func engagementReportGhostwriterRow(eng *dbconnections.Engagement, snapshotAt time.Time) ghostwriterSortRow {
	ht := dbconnections.NormalizeEngagementHaulType(eng.HaulType)
	label := dbconnections.EngagementHaulTypeLabel(ht)
	ts := snapshotAt.UTC().Format(time.RFC3339)
	meta, _ := json.Marshal(map[string]interface{}{
		"engagement_id":   eng.ID.Hex(),
		"haul_type":       ht,
		"haul_type_label": label,
	})
	desc := fmt.Sprintf("Engagement | name=%s | client=%s | haul_type=%s (%s)", eng.Name, eng.ClientName, ht, label)
	row := []string{
		"engagement-" + eng.ID.Hex(),
		ts,
		ts,
		ghostwriterC2EndpointName,
		ghostwriterC2EndpointName,
		"ReaperC2",
		"Reports | engagement context",
		"",
		desc,
		"",
		string(meta),
		"",
		"reaperc2,reports,engagement," + ht,
	}
	return ghostwriterSortRow{t: snapshotAt.Add(time.Second), row: row}
}

// WriteReportsGhostwriterCSV writes the same 13-column Ghostwriter schema from report snapshot data (clients, profiles, command output) without audit or operator chat.
// If eng is non-nil, a first row documents the engagement (including haul type) so it appears in Ghostwriter imports.
func WriteReportsGhostwriterCSV(w io.Writer, eng *dbconnections.Engagement, clients []dbconnections.BeaconClientDocument, profiles []dbconnections.BeaconProfile, cmdOut []dbconnections.CommandOutputRecord, snapshotAt time.Time, redact bool) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(ghostwriterCSVHeader); err != nil {
		return err
	}

	labelBy := beaconLabelsFromClients(clients)

	var rows []ghostwriterSortRow
	if eng != nil {
		rows = append(rows, engagementReportGhostwriterRow(eng, snapshotAt))
	}
	rows = append(rows, ghostwriterRowsFromCommandOutput(cmdOut, labelBy)...)

	for _, c := range clients {
		id := "client-" + c.ClientId
		ts := snapshotAt.UTC().Format(time.RFC3339)
		desc := fmt.Sprintf("Registered client | client_id=%s | active=%v | connection_type=%s", c.ClientId, c.Active, c.ConnectionType)
		if c.BeaconLabel != "" {
			desc += " | label=" + c.BeaconLabel
		}
		if c.ParentClientId != "" {
			desc += " | parent=" + c.ParentClientId
		}
		meta := map[string]interface{}{
			"expected_heartbeat":     c.ExpectedHeartBeat,
			"heartbeat_interval_sec": c.HeartbeatIntervalSec,
			"commands":               c.Commands,
		}
		if redact {
			meta["secret"] = "[REDACTED]"
		} else {
			meta["secret"] = c.Secret
		}
		comments, _ := json.Marshal(meta)
		src, dst := ghostwriterEndpointsC2ToBeacon(c.ClientId, labelBy)
		row := []string{
			id,
			ts,
			ts,
			src,
			dst,
			"ReaperC2",
			"Reports | registered beacon client",
			"",
			desc,
			"",
			string(comments),
			"",
			"reaperc2,reports,client",
		}
		rows = append(rows, ghostwriterSortRow{t: snapshotAt, row: row})
	}

	for _, p := range profiles {
		id := "profile-" + p.ID.Hex()
		pt := p.CreatedAt
		if pt.IsZero() {
			pt = snapshotAt
		}
		ts := pt.UTC().Format(time.RFC3339)
		desc := fmt.Sprintf("Saved profile | name=%s | client_id=%s | connection_type=%s", p.Name, p.ClientID, p.ConnectionType)
		if p.Label != "" {
			desc += " | label=" + p.Label
		}
		meta := map[string]interface{}{
			"name":                   p.Name,
			"client_id":              p.ClientID,
			"connection_type":        p.ConnectionType,
			"parent_client_id":       p.ParentClientID,
			"pivot_proxy":            p.PivotProxy,
			"label":                  p.Label,
			"heartbeat_interval_sec": p.HeartbeatIntervalSec,
			"scythe_example":         p.ScytheExample,
			"beacon_base_url":        p.BeaconBaseURL,
			"heartbeat_url":          p.HeartbeatURL,
			"created_by":             p.CreatedBy,
		}
		if redact {
			meta["secret"] = "[REDACTED]"
		} else {
			meta["secret"] = p.Secret
		}
		comments, _ := json.Marshal(meta)
		pdest := ghostwriterProfileDestinationLabel(p, labelBy)
		src, dst := ghostwriterC2EndpointName, pdest
		if pdest == "" {
			src, dst = ghostwriterC2EndpointName, ghostwriterC2EndpointName
		}
		row := []string{
			id,
			ts,
			ts,
			src,
			dst,
			"ReaperC2",
			"Reports | saved beacon profile",
			"",
			desc,
			"",
			string(comments),
			p.CreatedBy,
			"reaperc2,reports,profile",
		}
		rows = append(rows, ghostwriterSortRow{t: pt, row: row})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].t.After(rows[j].t)
	})

	for _, r := range rows {
		if err := cw.Write(r.row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
