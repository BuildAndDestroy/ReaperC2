package dbconnections

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionMFAChallenges = "operator_mfa_challenges"

// MFAChallengesCollection stores short-lived tokens between password check and TOTP entry at login.
var MFAChallengesCollection *mongo.Collection

// MFAChallenge is a pending second factor step after successful password verification.
type MFAChallenge struct {
	Token     string    `bson:"_id"`
	Username  string    `bson:"username"`
	ExpiresAt time.Time `bson:"expires_at"`
}

func initMFAChallengesCollection(db *mongo.Database) {
	MFAChallengesCollection = db.Collection(collectionMFAChallenges)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = MFAChallengesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(600), // 10 minutes; also enforced in code
	})
}

// InsertMFAChallenge stores a challenge token for the given username.
func InsertMFAChallenge(ctx context.Context, token, username string, expiresAt time.Time) error {
	_, err := MFAChallengesCollection.InsertOne(ctx, MFAChallenge{
		Token:     token,
		Username:  username,
		ExpiresAt: expiresAt,
	})
	return err
}

// DeleteMFAChallenge removes a challenge by token (after successful TOTP or abandon).
func DeleteMFAChallenge(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	_, err := MFAChallengesCollection.DeleteOne(ctx, bson.M{"_id": token})
	return err
}

// PeekMFAChallenge returns the username if the token exists and is not expired (does not delete).
func PeekMFAChallenge(ctx context.Context, token string) (username string, err error) {
	if token == "" {
		return "", mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var ch MFAChallenge
	err = MFAChallengesCollection.FindOne(ctx, bson.M{
		"_id":        token,
		"expires_at": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&ch)
	if err != nil {
		return "", err
	}
	return ch.Username, nil
}

// DeleteMFAChallengesForUser removes any pending login challenges for a user (e.g. after password change).
func DeleteMFAChallengesForUser(ctx context.Context, username string) error {
	if username == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	_, err := MFAChallengesCollection.DeleteMany(ctx, bson.M{"username": username})
	return err
}
