package dbconnections

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClientAuth represents a client stored in MongoDB
type ClientAuth struct {
	ClientId       string `bson:"ClientId"`
	Secret         string `bson:"Secret"`
	Active         bool   `bson:"Active"`
	ConnectionType string `bson:"Connection_Type"`
}

// MongoDB connection details
const (
	collectionClients   = "clients"
	collectionHeartbeat = "heartbeat"
	collectionData      = "data"
)

// MongoDB client
var Client *mongo.Client

// Collection references
var ClientCollection *mongo.Collection
var HeartbeatCollection *mongo.Collection
var DataCollection *mongo.Collection

// getEnvWithDefault returns the value of an environment variable or a default value if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// buildMongoURI constructs a MongoDB connection URI based on environment variables
func buildMongoURI(env string) string {
	// Get configuration from environment variables
	host := getEnvWithDefault("MONGO_HOST", "")
	port := getEnvWithDefault("MONGO_PORT", "27017")
	username := getEnvWithDefault("MONGO_USERNAME", "api_user")
	password := getEnvWithDefault("MONGO_PASSWORD", "")
	database := getEnvWithDefault("MONGO_DATABASE", "api_db")

	// Validate required fields
	if host == "" {
		log.Fatal("MONGO_HOST environment variable is required")
	}
	if password == "" {
		log.Fatal("MONGO_PASSWORD environment variable is required")
	}

	// Build base URI
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", username, password, host, port, database)

	// Add TLS configuration for AWS DocumentDB
	if strings.ToUpper(env) == "AWS" {
		tlsCAFile := getEnvWithDefault("MONGO_TLS_CA_FILE", "/etc/ssl/certs/rds-combined-ca-bundle.pem")
		// DocumentDB requires TLS with specific connection parameters
		uri += "?tls=true&tlsCAFile=" + tlsCAFile + "&replicaSet=rs0&readPreference=secondaryPreferred&retryWrites=false"
	} else {
		// For on-prem, check if TLS is explicitly requested
		if useTLS := getEnvWithDefault("MONGO_USE_TLS", "false"); strings.ToLower(useTLS) == "true" {
			uri += "?tls=true"
			if tlsCAFile := os.Getenv("MONGO_TLS_CA_FILE"); tlsCAFile != "" {
				uri += "&tlsCAFile=" + tlsCAFile
			}
		}
	}

	if as := getEnvWithDefault("MONGO_AUTH_SOURCE", ""); as != "" {
		sep := "?"
		if strings.Contains(uri, "?") {
			sep = "&"
		}
		uri += sep + "authSource=" + url.QueryEscape(as)
	}

	return uri
}

// Connect to database for the correct environment
func InitMongoDB(env string) {
	var err error

	// Build connection URI based on environment
	uri := buildMongoURI(env)
	// Log connection info without exposing password
	log.Printf("Connecting to MongoDB (environment: %s)", env)

	// Set MongoDB connection options
	// TLS configuration is handled through URI parameters (tls=true&tlsCAFile=...)
	Client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("MongoDB Connection Error:", err)
	}

	// Verify the connection
	err = Client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("MongoDB Ping Error:", err)
	}

	// Set collections
	databaseName := getEnvWithDefault("MONGO_DATABASE", "api_db")
	db := Client.Database(databaseName)
	ClientCollection = db.Collection(collectionClients)
	HeartbeatCollection = db.Collection(collectionHeartbeat)
	DataCollection = db.Collection(collectionData)
	initAdminCollections(db)
	log.Println("Connected to MongoDB!")
}

// FetchHeartbeat retrieves the latest heartbeat status from MongoDB
func FetchHeartbeat() (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result bson.M

	err := HeartbeatCollection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// FindClientByUUID searches for a client in MongoDB by UUID
func FindClientByUUID(uuid string) (*ClientAuth, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var client ClientAuth
	err := ClientCollection.FindOne(ctx, bson.M{"ClientId": uuid}).Decode(&client)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

// Fetch and clear commands for a given ClientId
func FetchAndClearCommands(clientId string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define a struct to hold just the Commands field
	var result struct {
		Commands []string `bson:"Commands"`
	}

	// Find the document and retrieve the Commands array
	filter := bson.M{"ClientId": clientId}
	err := ClientCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No commands found
		}
		log.Println("Error fetching commands:", err)
		return nil, err
	}

	// Clear the commands array and record check-in time (same write as heartbeat).
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"Commands":   []string{},
		"LastSeenAt": now,
	}}
	_, err = ClientCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("Error clearing commands:", err)
		return nil, err
	}

	return result.Commands, nil
}

// AppendBeaconCommands appends command strings to the client's Commands array (delivered on next heartbeat).
func AppendBeaconCommands(ctx context.Context, clientID string, commands []string) error {
	var push []string
	for _, c := range commands {
		c = strings.TrimSpace(c)
		if c != "" {
			push = append(push, c)
		}
	}
	if len(push) == 0 {
		return fmt.Errorf("no commands to queue")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := ClientCollection.UpdateOne(ctx, bson.M{"ClientId": clientID}, bson.M{
		"$push": bson.M{"Commands": bson.M{"$each": push}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// DeleteBeaconClient removes a row from the clients collection by ClientId.
func DeleteBeaconClient(ctx context.Context, clientID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := ClientCollection.DeleteOne(ctx, bson.M{"ClientId": clientID})
	return err
}

// UpdateBeaconLastSeen sets LastSeenAt for a client (e.g. after receiving task output).
func UpdateBeaconLastSeen(ctx context.Context, clientID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := ClientCollection.UpdateOne(ctx, bson.M{"ClientId": clientID}, bson.M{
		"$set": bson.M{"LastSeenAt": time.Now().UTC()},
	})
	return err
}

func FetchClientData(clientId string) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []bson.M
	cursor, err := DataCollection.Find(ctx, bson.M{"ClientId": clientId})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// StoreClientData saves received command output in the "data" collection
func StoreClientData(clientUUID, command, output string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dataEntry := bson.M{
		"ClientId":  clientUUID,
		"Command":   command,
		"Output":    output,
		"Timestamp": time.Now(),
	}

	_, err := DataCollection.InsertOne(ctx, dataEntry)
	if err != nil {
		return err
	}
	return nil
}

// CommandOutputRecord is one row in the data collection (command result from POST /receive/{uuid}).
type CommandOutputRecord struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ClientID  string             `json:"client_id" bson:"ClientId"`
	Command   string             `json:"command" bson:"Command"`
	Output    string             `json:"output" bson:"Output"`
	Timestamp time.Time          `json:"timestamp" bson:"Timestamp"`
}

// ListCommandOutputForClient returns stored command/output pairs for a beacon, newest first.
func ListCommandOutputForClient(ctx context.Context, clientID string, limit int64) ([]CommandOutputRecord, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cur, err := DataCollection.Find(ctx, bson.M{"ClientId": clientID}, options.Find().
		SetSort(bson.D{{Key: "Timestamp", Value: -1}}).
		SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []CommandOutputRecord
	for cur.Next(ctx) {
		var rec CommandOutputRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, cur.Err()
}

// ListRecentCommandOutputForExport returns newest rows from the data collection (reports / briefings).
func ListRecentCommandOutputForExport(ctx context.Context, limit int64) ([]CommandOutputRecord, error) {
	if limit < 1 || limit > 20000 {
		limit = 5000
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	cur, err := DataCollection.Find(ctx, bson.M{}, options.Find().
		SetSort(bson.D{{Key: "Timestamp", Value: -1}}).
		SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []CommandOutputRecord
	for cur.Next(ctx) {
		var rec CommandOutputRecord
		if err := cur.Decode(&rec); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, cur.Err()
}

// BeaconClientExists reports whether a clients document exists for ClientId.
func BeaconClientExists(ctx context.Context, clientID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	n, err := ClientCollection.CountDocuments(ctx, bson.M{"ClientId": clientID})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
