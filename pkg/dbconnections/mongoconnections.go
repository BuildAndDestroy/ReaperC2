package dbconnections

import (
	"context"
	"log"
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
	mongoApiUsername    = "api_user"
	mongoApiPassword    = "api_mongoApiPassword"
	databaseName        = "api_db"
	mongoURI            = "mongodb://" + mongoApiUsername + ":" + mongoApiPassword + "@172.17.0.2:27017/" + databaseName
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

// Connect to MongoDB
func InitMongoDB() {
	var err error
	// Set MongoDB connection options
	Client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB Connection Error:", err)
	}

	// Verify the connection
	err = Client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("MongoDB Ping Error:", err)
	}

	// Set collections
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
