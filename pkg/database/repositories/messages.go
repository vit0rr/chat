package repositories

import (
	"context"
	"time"

	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Message struct {
	RoomID     string    `bson:"roomId"`
	Message    string    `bson:"message"`
	FromUserID string    `bson:"fromUserId"`
	Nickname   string    `bson:"nickname"`
	CreatedAt  time.Time `bson:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt"`
}

type CreateMessageData struct {
	RoomID     string `json:"roomId"`
	Message    string `json:"message"`
	FromUserID string `json:"fromUserId"`
	Nickname   string `json:"nickname"`
}

type GetMessagesData struct {
	RoomID string
	Limit  int64
	Skip   int64
}

func CreateMessage(ctx context.Context, db *mongo.Database, data CreateMessageData) (*mongo.InsertOneResult, error) {
	now := time.Now()

	collection := db.Collection(constants.MessagesCollection)

	messages, err := collection.InsertOne(ctx, Message{
		RoomID:     data.RoomID,
		Message:    data.Message,
		FromUserID: data.FromUserID,
		Nickname:   data.Nickname,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	if err != nil {
		log.Error(ctx, "Failed to create message", log.ErrAttr(err))
		return nil, err
	}

	return messages, nil
}

func GetMessages(ctx context.Context, db *mongo.Database, data GetMessagesData) (*mongo.Cursor, error) {
	collection := db.Collection(constants.MessagesCollection)

	options := options.Find()
	options.SetSort(bson.D{{Key: "createdAt", Value: 1}}) // Sort by oldest first
	options.SetLimit(data.Limit)
	options.SetSkip(data.Skip)

	filter := bson.M{"roomId": data.RoomID}

	cursor, err := collection.Find(ctx, filter, options)
	if err != nil {
		log.Error(ctx, "Failed to get messages", log.ErrAttr(err))
		return nil, err
	}

	return cursor, nil
}
