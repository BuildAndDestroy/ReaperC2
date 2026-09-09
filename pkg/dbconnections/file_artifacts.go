package dbconnections

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionFileArtifacts = "file_artifacts"
	gridFSBucketName        = "reaper_artifacts"
	// FileArtifactKindStaging is an operator-uploaded blob waiting to be sent to a beacon.
	FileArtifactKindStaging = "staging"
	// FileArtifactKindDownload is a file pulled from a beacon via Scythe download.
	FileArtifactKindDownload = "download"
)

// FileArtifactsCollection stores metadata for staged uploads and beacon downloads.
var FileArtifactsCollection *mongo.Collection

var fileArtifactsDB *mongo.Database

func initFileArtifactsCollection(db *mongo.Database) {
	fileArtifactsDB = db
	FileArtifactsCollection = db.Collection(collectionFileArtifacts)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, _ = FileArtifactsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "client_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
}

func artifactBucket() (*gridfs.Bucket, error) {
	if fileArtifactsDB == nil {
		return nil, fmt.Errorf("file artifacts storage not initialized")
	}
	return gridfs.NewBucket(fileArtifactsDB, options.GridFSBucket().SetName(gridFSBucketName))
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

func applyBucketWriteDeadline(bucket *gridfs.Bucket, ctx context.Context) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = bucket.SetWriteDeadline(deadline)
	}
}

func applyBucketReadDeadline(bucket *gridfs.Bucket, ctx context.Context) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = bucket.SetReadDeadline(deadline)
	}
}

// writeArtifactBytes stores file bytes in GridFS under the given ObjectID. The GridFS filename is the hex id.
func writeArtifactBytes(ctx context.Context, id primitive.ObjectID, r io.Reader, maxBytes int64) (int64, error) {
	bucket, err := artifactBucket()
	if err != nil {
		return 0, err
	}
	applyBucketWriteDeadline(bucket, ctx)
	us, err := bucket.OpenUploadStreamWithID(id, id.Hex())
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(us, io.LimitReader(r, maxBytes+1))
	if err != nil {
		_ = us.Abort()
		return 0, err
	}
	if n > maxBytes {
		_ = us.Abort()
		return 0, fmt.Errorf("file larger than %d bytes", maxBytes)
	}
	if err := us.Close(); err != nil {
		_ = bucket.DeleteContext(ctx, id)
		return 0, err
	}
	return n, nil
}

func deleteArtifactBytes(ctx context.Context, id primitive.ObjectID) error {
	bucket, err := artifactBucket()
	if err != nil {
		return err
	}
	err = bucket.DeleteContext(ctx, id)
	if err == nil || errors.Is(err, gridfs.ErrFileNotFound) {
		return nil
	}
	return err
}

// WriteStagingArtifact stores an operator upload for later enqueue as a Scythe upload command.
func WriteStagingArtifact(ctx context.Context, clientID, originalName string, r io.Reader, maxBytes int64) (*FileArtifact, error) {
	id := primitive.NewObjectID()
	n, err := writeArtifactBytes(ctx, id, r, maxBytes)
	if err != nil {
		return nil, err
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
		_ = deleteArtifactBytes(ctx, id)
		return nil, err
	}
	return &doc, nil
}

// WriteDownloadArtifact stores bytes from a Scythe beacon download result.
func WriteDownloadArtifact(ctx context.Context, clientID, remotePath string, data []byte) (*FileArtifact, error) {
	id := primitive.NewObjectID()
	n, err := writeArtifactBytes(ctx, id, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	doc := FileArtifact{
		ID:         id,
		ClientID:   clientID,
		Kind:       FileArtifactKindDownload,
		RemotePath: remotePath,
		ByteSize:   n,
		CreatedAt:  time.Now().UTC(),
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if bc, err := FindBeaconClientByID(ctx, clientID); err == nil && bc != nil {
		doc.EngagementID = strings.TrimSpace(bc.EngagementId)
	}
	if _, err := FileArtifactsCollection.InsertOne(ctx, doc); err != nil {
		_ = deleteArtifactBytes(ctx, id)
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

// ReadArtifactBytes returns GridFS bytes for an artifact.
func ReadArtifactBytes(ctx context.Context, id primitive.ObjectID) ([]byte, error) {
	bucket, err := artifactBucket()
	if err != nil {
		return nil, err
	}
	applyBucketReadDeadline(bucket, ctx)
	var buf bytes.Buffer
	if _, err := bucket.DownloadToStream(id, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	return deleteArtifactBytes(ctx, id)
}

// DeleteArtifactByID removes any artifact row (staging or download) and deletes GridFS bytes if present.
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
	return deleteArtifactBytes(ctx, id)
}
