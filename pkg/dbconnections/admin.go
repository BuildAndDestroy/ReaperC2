package dbconnections

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionOperators        = "operators"
	collectionOperatorSessions = "operator_sessions"
)

// OperatorsCollection holds admin users for the web panel.
var OperatorsCollection *mongo.Collection

// OperatorSessionsCollection stores opaque session tokens for operators.
var OperatorSessionsCollection *mongo.Collection

// Role constants for portal RBAC (stored in operators.role).
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
)

// Operator is a human operator account for the admin panel.
type Operator struct {
	Username     string    `bson:"username"`
	PasswordHash string    `bson:"password_hash"`
	Role         string    `bson:"role,omitempty"`
	CreatedAt    time.Time `bson:"created_at"`
	// TotpEnabled is true after the user completes TOTP enrollment (Google Authenticator, etc.).
	TotpEnabled bool `bson:"totp_enabled,omitempty"`
	// TotpSecret is the base32-encoded shared secret; empty when MFA is off.
	TotpSecret string `bson:"totp_secret,omitempty"`
	// TotpPendingSecret holds the next secret until the user confirms with a valid code.
	TotpPendingSecret string `bson:"totp_pending_secret,omitempty"`
	// Disabled if true: cannot sign in; existing sessions are invalidated when set.
	Disabled bool `bson:"disabled,omitempty"`
}

// OperatorSession is a server-side session record (token is stored as _id).
type OperatorSession struct {
	Token     string    `bson:"_id"`
	Username  string    `bson:"username"`
	ExpiresAt time.Time `bson:"expires_at"`
}

// BeaconClientDocument is the shape stored in the clients collection for beacon auth.
type BeaconClientDocument struct {
	ClientId          string `bson:"ClientId"`
	Secret            string `bson:"Secret"`
	Active            bool   `bson:"Active"`
	ConnectionType    string `bson:"Connection_Type"`
	ExpectedHeartBeat string `bson:"ExpectedHeartBeat"`
	// HeartbeatIntervalSec is the operator-configured expected check-in period (drives topology / UI). 0 means derive from ExpectedHeartBeat.
	HeartbeatIntervalSec int      `bson:"HeartbeatIntervalSec,omitempty"`
	// Commands are delivered on heartbeat as a JSON array: strings (shell/builtins) or JSON objects (Scythe HTTP file upload/download maps).
	Commands []interface{} `bson:"Commands"`
	// ParentClientId, if set, points to another ClientId (pivot chain toward C2).
	ParentClientId string `bson:"ParentClientId,omitempty"`
	// BeaconLabel is a display name for topology / reports.
	BeaconLabel string `bson:"BeaconLabel,omitempty"`
	// LastSeenAt is updated when the beacon checks in (heartbeat or result post).
	LastSeenAt *time.Time `bson:"LastSeenAt,omitempty"`
	// EngagementId links this beacon to an engagement (hex ObjectId string); empty for legacy rows.
	EngagementId string `bson:"EngagementId,omitempty"`
}

func initAdminCollections(db *mongo.Database) {
	OperatorsCollection = db.Collection(collectionOperators)
	OperatorSessionsCollection = db.Collection(collectionOperatorSessions)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, _ = OperatorsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	initMFAChallengesCollection(db)
	initPortalCollections(db)
	initAuditCollections(db)
	initFileArtifactsCollection(db)
	initEngagementsCollection(db)
}

// CountOperators returns how many operator accounts exist.
func CountOperators(ctx context.Context) (int64, error) {
	return OperatorsCollection.CountDocuments(ctx, bson.M{})
}

// InsertOperator creates a new operator (username must be unique).
func InsertOperator(ctx context.Context, op Operator) error {
	op.CreatedAt = time.Now().UTC()
	_, err := OperatorsCollection.InsertOne(ctx, op)
	return err
}

// FindOperatorByUsername loads an operator by username.
func FindOperatorByUsername(ctx context.Context, username string) (*Operator, error) {
	var op Operator
	err := OperatorsCollection.FindOne(ctx, bson.M{"username": username}).Decode(&op)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// ListOperators returns all operator accounts (usernames and roles; no password hashes).
func ListOperators(ctx context.Context) ([]Operator, error) {
	cur, err := OperatorsCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "username", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []Operator
	for cur.Next(ctx) {
		var op Operator
		if err := cur.Decode(&op); err != nil {
			return nil, err
		}
		op.PasswordHash = ""
		op.TotpSecret = ""
		op.TotpPendingSecret = ""
		out = append(out, op)
	}
	return out, cur.Err()
}

// InsertSession stores a new session; token is the document _id.
func InsertSession(ctx context.Context, s OperatorSession) error {
	_, err := OperatorSessionsCollection.InsertOne(ctx, s)
	return err
}

// FindSessionByToken returns a session if it exists and is not expired.
func FindSessionByToken(ctx context.Context, token string) (*OperatorSession, error) {
	var s OperatorSession
	err := OperatorSessionsCollection.FindOne(ctx, bson.M{"_id": token}).Decode(&s)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(s.ExpiresAt) {
		_, _ = OperatorSessionsCollection.DeleteOne(ctx, bson.M{"_id": token})
		return nil, mongo.ErrNoDocuments
	}
	return &s, nil
}

// DeleteSession removes a session by token.
func DeleteSession(ctx context.Context, token string) error {
	_, err := OperatorSessionsCollection.DeleteOne(ctx, bson.M{"_id": token})
	return err
}

// DeleteSessionsForUsername removes all sessions for an operator (e.g. after disable).
func DeleteSessionsForUsername(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := OperatorSessionsCollection.DeleteMany(ctx, bson.M{"username": username})
	return err
}

// OperatorIsDisabled reports whether login should be rejected (legacy docs = active).
func OperatorIsDisabled(op *Operator) bool {
	return op != nil && op.Disabled
}

// CountActiveAdminsExcluding returns how many non-disabled admin accounts exist besides excludeUsername.
func CountActiveAdminsExcluding(ctx context.Context, excludeUsername string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	excludeUsername = strings.TrimSpace(excludeUsername)
	cur, err := OperatorsCollection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"username": 1, "role": 1, "disabled": 1}))
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)
	n := 0
	for cur.Next(ctx) {
		var op Operator
		if err := cur.Decode(&op); err != nil {
			return 0, err
		}
		if strings.EqualFold(op.Username, excludeUsername) {
			continue
		}
		if OperatorIsDisabled(&op) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(op.Role), RoleOperator) {
			continue
		}
		n++
	}
	return n, cur.Err()
}

// SetOperatorDisabled sets disabled flag and returns mongo.ErrNoDocuments if user missing.
func SetOperatorDisabled(ctx context.Context, username string, disabled bool) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"disabled": disabled}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// UpdateOperatorPasswordHash sets a new password hash for an operator.
func UpdateOperatorPasswordHash(ctx context.Context, username, passwordHash string) error {
	username = strings.TrimSpace(username)
	if username == "" || passwordHash == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"password_hash": passwordHash}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// SetOperatorTotpPending stores a base32 secret for enrollment (not yet active).
func SetOperatorTotpPending(ctx context.Context, username, pendingSecretBase32 string) error {
	username = strings.TrimSpace(username)
	if username == "" || pendingSecretBase32 == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$set": bson.M{"totp_pending_secret": pendingSecretBase32}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ConfirmOperatorTotp promotes totp_pending_secret to totp_secret and enables MFA.
func ConfirmOperatorTotp(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var op Operator
	err := OperatorsCollection.FindOne(ctx, bson.M{"username": username}).Decode(&op)
	if err != nil {
		return err
	}
	if strings.TrimSpace(op.TotpPendingSecret) == "" {
		return mongo.ErrNoDocuments
	}
	_, err = OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{
		"$set": bson.M{
			"totp_secret":  op.TotpPendingSecret,
			"totp_enabled": true,
		},
		"$unset": bson.M{"totp_pending_secret": ""},
	})
	return err
}

// DisableOperatorTotp turns off MFA and clears secrets.
func DisableOperatorTotp(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{
		"$set": bson.M{"totp_enabled": false},
		"$unset": bson.M{
			"totp_secret":         "",
			"totp_pending_secret": "",
		},
	})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ClearOperatorTotpPending abandons in-progress enrollment.
func ClearOperatorTotpPending(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return mongo.ErrNoDocuments
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := OperatorsCollection.UpdateOne(ctx, bson.M{"username": username}, bson.M{"$unset": bson.M{"totp_pending_secret": ""}})
	return err
}

// BeaconHeartbeatIntervalSec returns the expected check-in period in seconds (minimum 1, default 60).
func BeaconHeartbeatIntervalSec(c BeaconClientDocument) int {
	if c.HeartbeatIntervalSec > 0 {
		return clampHeartbeatSec(c.HeartbeatIntervalSec)
	}
	return parseExpectedHeartbeatString(c.ExpectedHeartBeat)
}

func clampHeartbeatSec(n int) int {
	if n < 1 {
		return 60
	}
	if n > 86400 {
		return 86400
	}
	return n
}

func parseExpectedHeartbeatString(s string) int {
	s = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(s), "s"))
	if s == "" {
		return 60
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 60
	}
	return clampHeartbeatSec(n)
}

// InsertBeaconClient inserts a beacon row for X-API-Secret / ClientId flows.
func InsertBeaconClient(ctx context.Context, doc BeaconClientDocument) error {
	_, err := ClientCollection.InsertOne(ctx, doc)
	return err
}

// FindBeaconClientByID loads a beacon client document by ClientId.
func FindBeaconClientByID(ctx context.Context, clientID string) (*BeaconClientDocument, error) {
	var doc BeaconClientDocument
	err := ClientCollection.FindOne(ctx, bson.M{"ClientId": clientID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}
