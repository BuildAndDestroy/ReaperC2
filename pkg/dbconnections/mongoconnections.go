package dbconnections

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

	// Clear the commands array in MongoDB
	update := bson.M{"$set": bson.M{"Commands": []string{}}}
	_, err = ClientCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("Error clearing commands:", err)
		return nil, err
	}

	return result.Commands, nil
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
