package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Client struct {
	Name      string     `bson:"name"`
	Slug      string     `bson:"slug"`
	Status    string     `bson:"status"`
	ApiKey    string     `bson:"apiKey"`
	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

type CreateClientData struct {
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Status    string     `json:"status"`
	ApiKey    string     `json:"apiKey"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type UpdateClientData struct {
	Name   string  `json:"name"`
	Slug   *string `json:"slug"`
	Status *string `json:"status"`
	ApiKey *string `json:"apiKey"`
}

type DeleteClientData struct {
	Slug string `json:"slug"`
}

type GetClientData struct {
	Slug string
}

func CreateClient(ctx context.Context, db *mongo.Database, data CreateClientData) (*mongo.InsertOneResult, error) {
	collection := db.Collection(constants.ClientsCollection)

	now := time.Now()
	data.CreatedAt = &now
	data.UpdatedAt = &now
	return collection.InsertOne(ctx, data)
}

func GetClient(ctx context.Context, db *mongo.Database, data GetClientData) (*Client, error) {
	collection := db.Collection(constants.ClientsCollection)

	var client Client
	err := collection.FindOne(ctx, data).Decode(&client)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func UpdateClient(ctx context.Context, db *mongo.Database, data UpdateClientData) (*mongo.UpdateResult, error) {
	client, err := GetClient(ctx, db, GetClientData{Slug: *data.Slug})
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("client not found")
	}

	collection := db.Collection(constants.ClientsCollection)
	filter := bson.M{"slug": data.Slug}

	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if data.Name != "" {
		update["$set"].(bson.M)["name"] = data.Name
	}
	if data.Status != nil {
		update["$set"].(bson.M)["status"] = *data.Status
	}
	if data.ApiKey != nil {
		update["$set"].(bson.M)["apiKey"] = *data.ApiKey
	}
	if data.Slug != nil {
		update["$set"].(bson.M)["slug"] = *data.Slug
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error(ctx, "Failed to update client", log.ErrAttr(err))
		return nil, err
	}

	return result, nil
}

func DeleteClient(ctx context.Context, db *mongo.Database, data DeleteClientData) (*mongo.UpdateResult, error) {
	collection := db.Collection(constants.ClientsCollection)
	return collection.UpdateOne(ctx, bson.M{"slug": data.Slug}, bson.M{"$set": bson.M{"deletedAt": time.Now()}})
}
