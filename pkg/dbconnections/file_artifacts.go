package dbconnections

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionFileArtifacts = "file_artifacts"
	// FileArtifactKindStaging is an operator-uploaded blob waiting to be sent to a beacon.
	FileArtifactKindStaging = "staging"
	// FileArtifactKindDownload is a file pulled from a beacon via Scythe download.
	FileArtifactKindDownload = "download"
)

// FileArtifactsCollection stores metadata for staged uploads and beacon downloads.
var FileArtifactsCollection *mongo.Collection

func initFileArtifactsCollection(db *mongo.Database) {
	FileArtifactsCollection = db.Collection(collectionFileArtifacts)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = FileArtifactsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "client_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
}

// ArtifactStorageRoot is the on-disk root for file bytes (override with REAPER_ARTIFACT_DIR).
func ArtifactStorageRoot() string {
	r := os.Getenv("REAPER_ARTIFACT_DIR")
	if r != "" {
		return r
	}
	return filepath.Join(".", "data", "reaper_artifacts")
}

func artifactPathForID(id primitive.ObjectID) string {
	return filepath.Join(ArtifactStorageRoot(), "artifacts", id.Hex())
}

// FileArtifact is metadata for a staged or downloaded file.
type FileArtifact struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClientID         string             `bson:"client_id" json:"client_id"`
	EngagementID     string             `bson:"engagement_id,omitempty" json:"engagement_id,omitempty"`
	Kind             string             `bson:"kind" json:"kind"`
	RemotePath       string             `bson:"remote_path,omitempty" json:"remote_path,omitempty"`
	OriginalFilename string             `bson:"original_filename,omitempty" json:"original_filename,omitempty"`
	ByteSize         int64              `bson:"byte_size" json:"byte_size"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
}

func ensureArtifactDir() error {
	root := filepath.Join(ArtifactStorageRoot(), "artifacts")
	return os.MkdirAll(root, 0750)
}

// WriteStagingArtifact stores an operator upload for later enqueue as a Scythe upload command.
func WriteStagingArtifact(ctx context.Context, clientID, originalName string, r io.Reader, maxBytes int64) (*FileArtifact, error) {
	if err := ensureArtifactDir(); err != nil {
		return nil, err
	}
	id := primitive.NewObjectID()
	path := artifactPathForID(id)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxBytes+1))
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	if n > maxBytes {
		_ = os.Remove(path)
		return nil, fmt.Errorf("file larger than %d bytes", maxBytes)
	}
	doc := FileArtifact{
		ID:               id,
		ClientID:         clientID,
		Kind:             FileArtifactKindStaging,
		OriginalFilename: originalName,
		ByteSize:         n,
		CreatedAt:        time.Now().UTC(),
	}
	if bc, err := FindBeaconClientByID(ctx, clientID); err == nil && bc != nil {
		doc.EngagementID = strings.TrimSpace(bc.EngagementId)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := FileArtifactsCollection.InsertOne(ctx, doc); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return &doc, nil
}

// WriteDownloadArtifact stores bytes from a Scythe beacon download result.
func WriteDownloadArtifact(ctx context.Context, clientID, remotePath string, data []byte) (*FileArtifact, error) {
	if err := ensureArtifactDir(); err != nil {
		return nil, err
	}
	id := primitive.NewObjectID()
	path := artifactPathForID(id)
	if err := os.WriteFile(path, data, 0640); err != nil {
		return nil, err
	}
	doc := FileArtifact{
		ID:         id,
		ClientID:   clientID,
		Kind:       FileArtifactKindDownload,
		RemotePath: remotePath,
		ByteSize:   int64(len(data)),
		CreatedAt:  time.Now().UTC(),
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if bc, err := FindBeaconClientByID(ctx, clientID); err == nil && bc != nil {
		doc.EngagementID = strings.TrimSpace(bc.EngagementId)
	}
	if _, err := FileArtifactsCollection.InsertOne(ctx, doc); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return &doc, nil
}

// FindFileArtifact loads metadata by id.
func FindFileArtifact(ctx context.Context, id primitive.ObjectID) (*FileArtifact, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var doc FileArtifact
	err := FileArtifactsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// ReadArtifactBytes returns on-disk bytes for an artifact.
func ReadArtifactBytes(id primitive.ObjectID) ([]byte, error) {
	path := artifactPathForID(id)
	return os.ReadFile(path)
}

// ListFileArtifactsForClient returns newest artifacts for a beacon (staging + download).
func ListFileArtifactsForClient(ctx context.Context, clientID string, limit int64) ([]FileArtifact, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cur, err := FileArtifactsCollection.Find(ctx, bson.M{"client_id": clientID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []FileArtifact
	for cur.Next(ctx) {
		var doc FileArtifact
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, cur.Err()
}

// DeleteStagingArtifact removes a staged upload and its file (after successful queue or operator cancel).
func DeleteStagingArtifact(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	res, err := FileArtifactsCollection.DeleteOne(ctx, bson.M{"_id": id, "kind": FileArtifactKindStaging})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	_ = os.Remove(artifactPathForID(id))
	return nil
}

// DeleteArtifactByID removes any artifact row (staging or download) and deletes on-disk bytes if present.
func DeleteArtifactByID(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	res, err := FileArtifactsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	_ = os.Remove(artifactPathForID(id))
	return nil
}
