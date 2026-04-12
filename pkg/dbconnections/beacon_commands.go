package dbconnections

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// StringifyBeaconCommand returns a short human-readable form for UI, audit, and logs.
func StringifyBeaconCommand(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case map[string]interface{}:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(b)
	case bson.M:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(b)
	default:
		b, err := json.Marshal(x)
		if err == nil && len(b) > 0 && b[0] == '{' {
			return string(b)
		}
		return fmt.Sprint(x)
	}
}

// StringifyBeaconCommands joins queued commands for display (pending queue tables, exports).
func StringifyBeaconCommands(cmds []interface{}) []string {
	if len(cmds) == 0 {
		return nil
	}
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, StringifyBeaconCommand(c))
	}
	return out
}
