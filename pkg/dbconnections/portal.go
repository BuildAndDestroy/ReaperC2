package dbconnections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionBeaconProfiles = "beacon_profiles"
	collectionOperatorChat   = "operator_chat"
)

// BeaconProfilesCollection stores saved beacon configurations from the admin UI.
var BeaconProfilesCollection *mongo.Collection

// OperatorChatCollection stores operator-to-operator chat messages.
var OperatorChatCollection *mongo.Collection

// BeaconProfile is a saved beacon profile (for reuse and reporting).
type BeaconProfile struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	Name             string             `bson:"name"`
	ClientID         string             `bson:"client_id"`
	Secret           string             `bson:"secret"`
	ConnectionType   string             `bson:"connection_type"`
	ParentClientID   string             `bson:"parent_client_id,omitempty"`
	Label                 string `bson:"label,omitempty"`
	HeartbeatIntervalSec  int    `bson:"heartbeat_interval_sec,omitempty"`
	ScytheExample         string `bson:"scythe_example"`
	BeaconBaseURL    string             `bson:"beacon_base_url"`
	HeartbeatURL     string             `bson:"heartbeat_url"`
	CreatedAt        time.Time          `bson:"created_at"`
	CreatedBy        string             `bson:"created_by"`
}

// ChatMessage is one operator chat line.
type ChatMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Room      string             `bson:"room"`
	Username  string             `bson:"username"`
	Body      string             `bson:"body"`
	CreatedAt time.Time          `bson:"created_at"`
}

func initPortalCollections(db *mongo.Database) {
	BeaconProfilesCollection = db.Collection(collectionBeaconProfiles)
	OperatorChatCollection = db.Collection(collectionOperatorChat)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, _ = OperatorChatCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
	_, _ = BeaconProfilesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
}

// ListBeaconClients returns all rows in the clients collection (for topology / reports).
func ListBeaconClients(ctx context.Context) ([]BeaconClientDocument, error) {
	cur, err := ClientCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "ClientId", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []BeaconClientDocument
	for cur.Next(ctx) {
		var doc BeaconClientDocument
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, cur.Err()
}

// InsertBeaconProfile saves a profile document.
func InsertBeaconProfile(ctx context.Context, p BeaconProfile) (primitive.ObjectID, error) {
	p.CreatedAt = time.Now().UTC()
	res, err := BeaconProfilesCollection.InsertOne(ctx, p)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

// ListBeaconProfiles returns saved profiles newest first.
func ListBeaconProfiles(ctx context.Context, limit int64) ([]BeaconProfile, error) {
	if limit < 1 {
		limit = 200
	}
	cur, err := BeaconProfilesCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []BeaconProfile
	for cur.Next(ctx) {
		var p BeaconProfile
		if err := cur.Decode(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, cur.Err()
}

// FindBeaconProfileByID loads a saved profile by MongoDB ObjectID hex, or nil if missing.
func FindBeaconProfileByID(ctx context.Context, idHex string) (*BeaconProfile, error) {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, err
	}
	var p BeaconProfile
	err = BeaconProfilesCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteBeaconProfile removes a profile by id hex string.
func DeleteBeaconProfile(ctx context.Context, idHex string) error {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return err
	}
	_, err = BeaconProfilesCollection.DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

// InsertChatMessage appends a chat line.
func InsertChatMessage(ctx context.Context, m ChatMessage) error {
	m.CreatedAt = time.Now().UTC()
	if m.Room == "" {
		m.Room = "global"
	}
	_, err := OperatorChatCollection.InsertOne(ctx, m)
	return err
}

// ListChatMessagesSince returns messages with created_at >= since (UTC), oldest first for display.
func ListChatMessagesSince(ctx context.Context, since time.Time, limit int64) ([]ChatMessage, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	filter := bson.M{"created_at": bson.M{"$gt": since}}
	cur, err := OperatorChatCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []ChatMessage
	for cur.Next(ctx) {
		var m ChatMessage
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, cur.Err()
}

// ListRecentChatMessages returns the last N messages (newest last in slice).
func ListRecentChatMessages(ctx context.Context, limit int64) ([]ChatMessage, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	cur, err := OperatorChatCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var rev []ChatMessage
	for cur.Next(ctx) {
		var m ChatMessage
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		rev = append(rev, m)
	}
	// reverse to chronological
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev, cur.Err()
}
