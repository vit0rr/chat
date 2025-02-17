package deps

import (
	"context"
	"fmt"

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

func CreateKeyIndex(ctx context.Context, dbClient *mongo.Client) error {
	collection := dbClient.Database(constants.DatabaseName).Collection(constants.RoomsCollection)

	index := mongo.IndexModel{
		Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "roomId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := collection.Indexes().CreateOne(ctx, index)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	log.Info(ctx, "✅ Created/Verified compound unique index for 'userId' and 'roomId' fields in 'rooms' collection")

	return nil
}
