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
	mongoURI            = "mongodb://172.17.0.2:27017"
	databaseName        = "api_db"
	collectionClients   = "clients"
	collectionHeartbeat = "heartbeat"
)

// MongoDB client
var Client *mongo.Client

// Collection references
var ClientCollection *mongo.Collection
var HeartbeatCollection *mongo.Collection

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
