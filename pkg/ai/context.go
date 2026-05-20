package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
)

const (
	maxContextNoteRunes   = 4000
	maxOutputRows         = 12
	maxOutputRunesPerRow  = 1200
	maxBeaconList         = 40
)

// BuildEngagementContext formats engagement, beacon, and recent output for the system prompt.
func BuildEngagementContext(ctx context.Context, eng *dbconnections.Engagement) (string, error) {
	if eng == nil {
		return "", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Active engagement (server context)\n")
	fmt.Fprintf(&b, "- **Name:** %s\n", eng.Name)
	fmt.Fprintf(&b, "- **Client:** %s\n", eng.ClientName)
	fmt.Fprintf(&b, "- **Haul:** %s\n", dbconnections.EngagementHaulTypeLabel(eng.HaulType))
	fmt.Fprintf(&b, "- **Status:** %s\n", engagementStatusLabel(eng))
	if n := strings.TrimSpace(eng.Notes); n != "" {
		fmt.Fprintf(&b, "- **Notes:**\n%s\n", truncateRunes(n, maxContextNoteRunes))
	}

	clients, err := dbconnections.ListBeaconClientsByEngagement(ctx, eng.ID.Hex())
	if err != nil {
		return b.String(), err
	}
	fmt.Fprintf(&b, "\n## Beacons (%d)\n", len(clients))
	if len(clients) == 0 {
		b.WriteString("_No beacons registered for this engagement yet. Direct the operator to Beacons → Generate._\n")
	} else {
		limit := len(clients)
		if limit > maxBeaconList {
			limit = maxBeaconList
		}
		for i := 0; i < limit; i++ {
			c := clients[i]
			label := strings.TrimSpace(c.BeaconLabel)
			if label == "" {
				label = c.ClientId
			}
			line := fmt.Sprintf("- **%s** (`%s`)", label, c.ClientId)
			if p := strings.TrimSpace(c.ParentClientId); p != "" {
				line += fmt.Sprintf(" parent=`%s`", p)
			}
			b.WriteString(line + "\n")
		}
		if len(clients) > maxBeaconList {
			fmt.Fprintf(&b, "_…and %d more beacons omitted._\n", len(clients)-maxBeaconList)
		}
	}

	outputs, err := dbconnections.ListRecentCommandOutputForEngagement(ctx, eng.ID.Hex(), maxOutputRows)
	if err != nil {
		return b.String(), err
	}
	fmt.Fprintf(&b, "\n## Recent command output (newest %d, truncated)\n", len(outputs))
	if len(outputs) == 0 {
		b.WriteString("_No stored output yet._\n")
	} else {
		for _, row := range outputs {
			ts := row.Timestamp.UTC().Format(time.RFC3339)
			cmd := truncateRunes(strings.TrimSpace(row.Command), 200)
			out := truncateRunes(strings.TrimSpace(row.Output), maxOutputRunesPerRow)
			fmt.Fprintf(&b, "\n### %s · `%s`\n**Command:** `%s`\n**Output:**\n```\n%s\n```\n",
				ts, row.ClientID, cmd, out)
		}
	}
	return b.String(), nil
}

func engagementStatusLabel(e *dbconnections.Engagement) string {
	if dbconnections.EngagementIsOpen(e) {
		return "open"
	}
	return "closed"
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
