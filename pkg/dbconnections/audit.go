package dbconnections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionAuditLogs = "audit_logs"

// Audit action names (portal audit trail).
const (
	AuditActionBeaconCreated       = "beacon_created"
	AuditActionUserCreated         = "user_created"
	AuditActionReportExported      = "report_exported"
	AuditActionBeaconProfileDel    = "beacon_profile_deleted"
	AuditActionAuditLogExported    = "audit_log_exported"
	AuditActionBeaconCommandQueued = "beacon_command_queued"
	AuditActionBeaconKillQueued    = "beacon_kill_queued"
)

// AuditLogsCollection stores operator/C2 portal audit events.
var AuditLogsCollection *mongo.Collection

// AuditLogEntry is one row in the admin audit trail.
type AuditLogEntry struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Time    time.Time          `bson:"time"`
	Actor   string             `bson:"actor"`
	Action  string             `bson:"action"`
	Details bson.M             `bson:"details,omitempty"`
}

func initAuditCollections(db *mongo.Database) {
	AuditLogsCollection = db.Collection(collectionAuditLogs)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = AuditLogsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "time", Value: -1}},
	})
}

// InsertAuditLog appends an audit event (best-effort; errors are logged by callers).
func InsertAuditLog(ctx context.Context, actor, action string, details bson.M) error {
	e := AuditLogEntry{
		Time:    time.Now().UTC(),
		Actor:   actor,
		Action:  action,
		Details: details,
	}
	_, err := AuditLogsCollection.InsertOne(ctx, e)
	return err
}

// ListAuditLogs returns newest entries first.
func ListAuditLogs(ctx context.Context, limit int64) ([]AuditLogEntry, error) {
	if limit < 1 || limit > 10000 {
		limit = 500
	}
	cur, err := AuditLogsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "time", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []AuditLogEntry
	for cur.Next(ctx) {
		var e AuditLogEntry
		if err := cur.Decode(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, cur.Err()
}

// ListAllAuditLogsForExport returns up to max entries (newest first).
func ListAllAuditLogsForExport(ctx context.Context, max int64) ([]AuditLogEntry, error) {
	if max < 1 {
		max = 50000
	}
	if max > 100000 {
		max = 100000
	}
	cur, err := AuditLogsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "time", Value: -1}}).SetLimit(max))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []AuditLogEntry
	for cur.Next(ctx) {
		var e AuditLogEntry
		if err := cur.Decode(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, cur.Err()
}
