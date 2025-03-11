package deps

import (
	"context"
	"fmt"
	"time"

	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/config"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoClient(ctx context.Context, cfg config.Config) (*mongo.Client, error) {
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.API.Mongo.Dsn))
	if err != nil {
		return nil, err
	}

	return mongoClient, nil
}

func CreateKeyIndex(ctx context.Context, db *mongo.Database) error {
	// create room-user index for compound unique
	collection := db.Collection(constants.RoomsCollection)

	roomUserIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "_id", Value: 1},
			{Key: "users.userId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, roomUserIndex)
	if err != nil {
		return fmt.Errorf("failed to create room-user index: %v", err)
	}

	log.Info(ctx, "✅ Created/Verified compound unique index for '_id' and 'users.userId' fields in 'rooms' collection")

	// create user index for externalId
	collection = db.Collection(constants.UsersCollection)

	userIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "externalId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, userIndex)
	if err != nil {
		return fmt.Errorf("failed to create user index: %v", err)
	}

	log.Info(ctx, "✅ Created/Verified unique index for 'externalId' field in 'users' collection")

	// create client index for apiKey
	collection = db.Collection(constants.ClientsCollection)

	clientIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "apiKey", Value: 1},
			{Key: "slug", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err = collection.Indexes().CreateOne(ctx, clientIndex)
	if err != nil {
		return fmt.Errorf("failed to create client index: %v", err)
	}

	log.Info(ctx, "✅ Created/Verified unique index for 'apiKey' field in 'clients' collection")

	return nil
}

func CreateMessagesTTLIndex(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection(constants.MessagesCollection)

	messagesTTLIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "createdAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90 days
	}

	_, err := collection.Indexes().CreateOne(ctx, messagesTTLIndex)
	if err != nil {
		return fmt.Errorf("failed to create messages TTL index: %v", err)
	}

	log.Info(ctx, "✅ Created/Verified TTL index for messages (90 days expiration)")

	return nil
}

func UpdateAllOnlineUsersToOffline(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection(constants.UsersCollection)
	_, err := collection.UpdateMany(
		ctx,
		bson.M{"activity": "online"},
		bson.M{"$set": bson.M{
			"activity":  "offline",
			"updatedAt": time.Now(),
		}},
	)

	return err
}
