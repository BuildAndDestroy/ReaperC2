package dbconnections

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupArtifactMongo(t *testing.T) context.Context {
	t.Helper()
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		uri = "mongodb://127.0.0.1:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetServerSelectionTimeout(500*time.Millisecond))
	if err != nil {
		t.Skipf("mongo unavailable: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		t.Skipf("mongo ping failed: %v", err)
	}

	prevColl := FileArtifactsCollection
	prevDB := fileArtifactsDB
	prevClients := ClientCollection

	dbName := "reaperc2_artifact_test_" + primitive.NewObjectID().Hex()
	db := client.Database(dbName)
	initFileArtifactsCollection(db)
	ClientCollection = db.Collection("clients")

	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
		FileArtifactsCollection = prevColl
		fileArtifactsDB = prevDB
		ClientCollection = prevClients
	})
	return context.Background()
}

func TestArtifactBucketNotInitialized(t *testing.T) {
	prev := fileArtifactsDB
	fileArtifactsDB = nil
	t.Cleanup(func() { fileArtifactsDB = prev })
	if _, err := artifactBucket(); err == nil {
		t.Fatal("expected error when GridFS db is unset")
	}
}

func TestWriteReadDeleteStagingArtifact(t *testing.T) {
	ctx := setupArtifactMongo(t)
	payload := []byte("staged payload")
	doc, err := WriteStagingArtifact(ctx, "client-a", "tool.sh", bytes.NewReader(payload), ScytheMaxFileBytes)
	if err != nil {
		t.Fatalf("WriteStagingArtifact: %v", err)
	}
	if doc.Kind != FileArtifactKindStaging || doc.OriginalFilename != "tool.sh" || doc.ByteSize != int64(len(payload)) {
		t.Fatalf("metadata: %+v", doc)
	}

	got, err := ReadArtifactBytes(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ReadArtifactBytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("bytes mismatch: %q", got)
	}

	listed, err := ListFileArtifactsForClient(ctx, "client-a", 10)
	if err != nil {
		t.Fatalf("ListFileArtifactsForClient: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != doc.ID {
		t.Fatalf("list: %+v", listed)
	}

	if err := DeleteArtifactByID(ctx, doc.ID); err != nil {
		t.Fatalf("DeleteArtifactByID: %v", err)
	}
	if _, err := FindFileArtifact(ctx, doc.ID); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("metadata after delete: %v", err)
	}
	if _, err := ReadArtifactBytes(ctx, doc.ID); !errors.Is(err, gridfs.ErrFileNotFound) {
		t.Fatalf("gridfs after delete: %v", err)
	}
}

func TestWriteReadDeleteDownloadArtifact(t *testing.T) {
	ctx := setupArtifactMongo(t)
	payload := []byte("downloaded from beacon")
	doc, err := WriteDownloadArtifact(ctx, "client-b", "/tmp/out.bin", payload)
	if err != nil {
		t.Fatalf("WriteDownloadArtifact: %v", err)
	}
	if doc.Kind != FileArtifactKindDownload || doc.RemotePath != "/tmp/out.bin" {
		t.Fatalf("metadata: %+v", doc)
	}
	got, err := ReadArtifactBytes(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ReadArtifactBytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("bytes mismatch: %q", got)
	}
	if err := DeleteStagingArtifact(ctx, doc.ID); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("DeleteStagingArtifact on download: %v", err)
	}
	got, err = ReadArtifactBytes(ctx, doc.ID)
	if err != nil {
		t.Fatalf("download bytes should remain: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("bytes after failed staging delete: %q", got)
	}
	if err := DeleteArtifactByID(ctx, doc.ID); err != nil {
		t.Fatalf("DeleteArtifactByID: %v", err)
	}
}

func TestWriteArtifactBytesOversizeLeavesNoGridFSOrphan(t *testing.T) {
	ctx := setupArtifactMongo(t)
	const chunk = 255 * 1024
	maxBytes := int64(chunk + 100)
	payload := bytes.Repeat([]byte("x"), int(maxBytes)+1)

	id := primitive.NewObjectID()
	_, err := writeArtifactBytes(ctx, id, bytes.NewReader(payload), maxBytes)
	if err == nil || !strings.Contains(err.Error(), "file larger than") {
		t.Fatalf("expected oversize error, got %v", err)
	}

	bucket, err := artifactBucket()
	if err != nil {
		t.Fatal(err)
	}
	files, err := bucket.GetFilesCollection().CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		t.Fatal(err)
	}
	if files != 0 {
		t.Fatalf("files leftover: %d", files)
	}
	chunks, err := bucket.GetChunksCollection().CountDocuments(ctx, bson.M{"files_id": id})
	if err != nil {
		t.Fatal(err)
	}
	if chunks != 0 {
		t.Fatalf("chunks leftover: %d", chunks)
	}

	_, err = WriteStagingArtifact(ctx, "client-c", "big.bin", bytes.NewReader(payload), 8)
	if err == nil || !strings.Contains(err.Error(), "file larger than") {
		t.Fatalf("WriteStagingArtifact oversize: %v", err)
	}
	n, err := FileArtifactsCollection.CountDocuments(ctx, bson.M{"client_id": "client-c"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("metadata leftover: %d", n)
	}
}

func TestWriteStagingArtifactEmptyFile(t *testing.T) {
	ctx := setupArtifactMongo(t)
	doc, err := WriteStagingArtifact(ctx, "client-d", "empty.bin", bytes.NewReader(nil), ScytheMaxFileBytes)
	if err != nil {
		t.Fatalf("empty file: %v", err)
	}
	got, err := ReadArtifactBytes(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ReadArtifactBytes: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty bytes, got %d", len(got))
	}
}
