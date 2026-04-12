package dbconnections

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// ScytheMaxFileBytes matches Scythe httpclient.MaxHTTPBuiltinFileBytes (HTTP built-in file read/write cap).
const ScytheMaxFileBytes = 64 * 1024 * 1024

const scytheDownloadHeaderLine = "SCYTHE_FILE_DOWNLOAD v1"

// ProcessScytheDownloadOutput parses beacon output from Scythe's download built-in. When the payload is
// a file download, bytes are stored as a FileArtifact and the returned string is a short summary for the data collection.
func ProcessScytheDownloadOutput(ctx context.Context, clientID string, output string) (string, error) {
	if !strings.HasPrefix(output, scytheDownloadHeaderLine) {
		return output, nil
	}
	remotePath, raw, ok := parseScytheDownloadPayload(output)
	if !ok || len(raw) == 0 {
		return output, nil
	}
	if int64(len(raw)) > ScytheMaxFileBytes {
		return output, fmt.Errorf("scythe download: decoded size exceeds cap")
	}
	doc, err := WriteDownloadArtifact(ctx, clientID, remotePath, raw)
	if err != nil {
		return output, err
	}
	return fmt.Sprintf("%s\npath=%s\nbytes_stored=%d\nartifact_id=%s\n(stored for operator download — Files on Commands or GET /api/beacon-artifacts)",
		scytheDownloadHeaderLine, remotePath, len(raw), doc.ID.Hex()), nil
}

func parseScytheDownloadPayload(output string) (remotePath string, raw []byte, ok bool) {
	if !strings.HasPrefix(output, scytheDownloadHeaderLine) {
		return "", nil, false
	}
	rest := strings.TrimPrefix(output, scytheDownloadHeaderLine)
	rest = strings.TrimPrefix(rest, "\n")
	parts := strings.SplitN(rest, "\n\n", 2)
	if len(parts) < 2 {
		return "", nil, false
	}
	headerBlock, b64 := parts[0], strings.TrimSpace(parts[1])
	var path string
	var sizeHint int64 = -1
	for _, line := range strings.Split(headerBlock, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "path=") {
			path = strings.TrimPrefix(line, "path=")
		}
		if strings.HasPrefix(line, "size=") {
			if n, err := strconv.ParseInt(strings.TrimPrefix(line, "size="), 10, 64); err == nil {
				sizeHint = n
			}
		}
	}
	if path == "" {
		return "", nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", nil, false
	}
	if sizeHint >= 0 && int64(len(raw)) != sizeHint {
		return "", nil, false
	}
	return path, raw, true
}
