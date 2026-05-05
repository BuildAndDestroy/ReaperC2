package dbconnections

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ReaperC2/pkg/mitreattack"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Engagement status for operator workflow (not the same as calendar end date).
const (
	EngagementStatusOpen   = "open"
	EngagementStatusClosed = "closed"
)

// EngagementHaulType categorizes the operation style (reporting / planning).
const (
	EngagementHaulInteractive = "interactive"
	EngagementHaulShortHaul   = "short_haul"
	EngagementHaulLongHaul    = "long_haul"
)

// NormalizeEngagementHaulType returns a valid haul key; empty or unknown defaults to interactive.
func NormalizeEngagementHaulType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case EngagementHaulShortHaul, EngagementHaulLongHaul, EngagementHaulInteractive:
		return s
	default:
		return EngagementHaulInteractive
	}
}

// EngagementHaulTypeLabel is a display name for reports and UI.
func EngagementHaulTypeLabel(key string) string {
	switch NormalizeEngagementHaulType(key) {
	case EngagementHaulShortHaul:
		return "Short Haul"
	case EngagementHaulLongHaul:
		return "Long Haul"
	default:
		return "Interactive"
	}
}

const collectionEngagements = "engagements"

// EngagementsCollection stores operator-facing engagement records.
var EngagementsCollection *mongo.Collection

func initEngagementsCollection(db *mongo.Database) {
	EngagementsCollection = db.Collection(collectionEngagements)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = EngagementsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
	_, _ = EngagementsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "assigned_operators", Value: 1}},
	})
}

// Engagement is one assessment / operation workspace (scopes beacons, reports, topology, etc.).
type Engagement struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name              string             `bson:"name" json:"name"`
	ClientName        string             `bson:"client_name" json:"client_name"`
	StartDate         time.Time          `bson:"start_date" json:"start_date"`
	EndDate           time.Time          `bson:"end_date" json:"end_date"`
	SlackDiscordRoom  string             `bson:"slack_discord_room,omitempty" json:"slack_discord_room,omitempty"`
	AssignedOperators []string           `bson:"assigned_operators" json:"assigned_operators"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	CreatedBy         string             `bson:"created_by,omitempty" json:"created_by,omitempty"`
	// Status is open or closed (legacy documents with no field are treated as open).
	Status string `bson:"status,omitempty" json:"status,omitempty"`
	// Notes is free-form operator text (scope, reminders, handoff).
	Notes string `bson:"notes,omitempty" json:"notes,omitempty"`
	// AttackTacticNotes maps enterprise ATT&CK tactic keys (Navigator shortnames) to operator notes.
	AttackTacticNotes map[string]string `bson:"attack_tactic_notes,omitempty" json:"attack_tactic_notes,omitempty"`
	// HaulType is interactive | short_haul | long_haul (planning / reporting category).
	HaulType string `bson:"haul_type,omitempty" json:"haul_type,omitempty"`
}

// InsertEngagement stores a new engagement; creator is added to AssignedOperators if missing.
func InsertEngagement(ctx context.Context, e Engagement) (primitive.ObjectID, error) {
	e.CreatedAt = time.Now().UTC()
	if e.AssignedOperators == nil {
		e.AssignedOperators = []string{}
	}
	if e.CreatedBy != "" {
		found := false
		for _, u := range e.AssignedOperators {
			if u == e.CreatedBy {
				found = true
				break
			}
		}
		if !found {
			e.AssignedOperators = append(e.AssignedOperators, e.CreatedBy)
		}
	}
	if strings.TrimSpace(e.Status) == "" {
		e.Status = EngagementStatusOpen
	}
	e.HaulType = NormalizeEngagementHaulType(e.HaulType)
	e.AttackTacticNotes = mitreattack.NormalizeTacticNotes(e.AttackTacticNotes)
	res, err := EngagementsCollection.InsertOne(ctx, e)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

// FindEngagementByID loads an engagement by MongoDB ObjectID hex.
func FindEngagementByID(ctx context.Context, idHex string) (*Engagement, error) {
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, err
	}
	var e Engagement
	err = EngagementsCollection.FindOne(ctx, bson.M{"_id": oid}).Decode(&e)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// UserCanAccessEngagement returns true if admin may see all engagements; operators only when listed in AssignedOperators.
func UserCanAccessEngagement(role, username string, e *Engagement) bool {
	if e == nil {
		return false
	}
	if role == RoleAdmin {
		return true
	}
	for _, u := range e.AssignedOperators {
		if u == username {
			return true
		}
	}
	return false
}

// NormalizeAssignedOperatorList trims, deduplicates, and drops empty strings.
func NormalizeAssignedOperatorList(usernames []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range usernames {
		u := strings.TrimSpace(raw)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

// ValidateAssignedOperatorUsernames ensures each name exists and is not disabled.
func ValidateAssignedOperatorUsernames(ctx context.Context, usernames []string) error {
	for _, u := range usernames {
		op, err := FindOperatorByUsername(ctx, u)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return fmt.Errorf("unknown operator %q", u)
			}
			return err
		}
		if OperatorIsDisabled(op) {
			return fmt.Errorf("operator %q is disabled", u)
		}
	}
	return nil
}

// ListEngagementsForUser returns engagements visible to this portal user (newest first).
func ListEngagementsForUser(ctx context.Context, role, username string) ([]Engagement, error) {
	var filter bson.M
	if role == RoleAdmin {
		filter = bson.M{}
	} else {
		filter = bson.M{"assigned_operators": username}
	}
	cur, err := EngagementsCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []Engagement
	for cur.Next(ctx) {
		var e Engagement
		if err := cur.Decode(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, cur.Err()
}

// EngagementIsOpen returns true if e is nil or status is empty/open (legacy rows).
func EngagementIsOpen(e *Engagement) bool {
	if e == nil {
		return true
	}
	s := strings.ToLower(strings.TrimSpace(e.Status))
	return s == "" || s == EngagementStatusOpen
}

// EngagementPatch is a partial update for engagements.
type EngagementPatch struct {
	Status            *string            // "open" | "closed", or nil to skip
	Notes             *string            // nil = skip; empty string clears notes
	AttackTacticNotes *map[string]string // nil = skip; empty map clears tactic notes
	HaulType          *string            // nil = skip; validated when set
	AssignedOperators *[]string          // nil = skip; replaces list; each user must exist and not be disabled
}

// UpdateEngagement applies non-nil patch fields. Validates status when set.
func UpdateEngagement(ctx context.Context, idHex string, patch EngagementPatch) error {
	idHex = strings.TrimSpace(idHex)
	if idHex == "" {
		return mongo.ErrNoDocuments
	}
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return err
	}
	set := bson.M{}
	if patch.Status != nil {
		s := strings.ToLower(strings.TrimSpace(*patch.Status))
		if s != EngagementStatusOpen && s != EngagementStatusClosed {
			return fmt.Errorf("invalid engagement status %q (use open or closed)", *patch.Status)
		}
		set["status"] = s
	}
	if patch.Notes != nil {
		set["notes"] = *patch.Notes
	}
	if patch.AttackTacticNotes != nil {
		norm := mitreattack.NormalizeTacticNotes(*patch.AttackTacticNotes)
		if norm == nil {
			set["attack_tactic_notes"] = bson.M{}
		} else {
			set["attack_tactic_notes"] = norm
		}
	}
	if patch.HaulType != nil {
		h := strings.ToLower(strings.TrimSpace(*patch.HaulType))
		if h != EngagementHaulInteractive && h != EngagementHaulShortHaul && h != EngagementHaulLongHaul {
			return fmt.Errorf("invalid haul_type %q (use interactive, short_haul, or long_haul)", *patch.HaulType)
		}
		set["haul_type"] = h
	}
	if patch.AssignedOperators != nil {
		norm := NormalizeAssignedOperatorList(*patch.AssignedOperators)
		if err := ValidateAssignedOperatorUsernames(ctx, norm); err != nil {
			return err
		}
		set["assigned_operators"] = norm
	}
	if len(set) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	res, err := EngagementsCollection.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
