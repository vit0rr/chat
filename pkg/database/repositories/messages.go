package repositories

import (
	"context"
	"fmt"
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

type GetTotalMessagesSentInARoomData struct {
	RoomID string
}

type UsersWhoSentMessagesInTheLastDaysData struct {
	Limit int64
	Skip  int64
	Days  int
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
	options.SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Sort by newest first
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

func TotalMessagesSent(ctx context.Context, db *mongo.Database) (int64, error) {
	collection := db.Collection(constants.MessagesCollection)

	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Error(ctx, "Failed to get total messages sent", log.ErrAttr(err))
		return 0, err
	}

	return total, nil
}

func TotalMessagesSentInARoom(ctx context.Context, db *mongo.Database, data GetTotalMessagesSentInARoomData) (int64, error) {
	collection := db.Collection(constants.MessagesCollection)

	total, err := collection.CountDocuments(ctx, bson.M{"roomId": data.RoomID})
	if err != nil {
		log.Error(ctx, "Failed to get total messages sent in a room", log.ErrAttr(err))
		return 0, err
	}

	return total, nil
}

func UsersWhoSentMessagesInTheLastDays(ctx context.Context, db *mongo.Database, data UsersWhoSentMessagesInTheLastDaysData) (*mongo.Cursor, error) {
	collection := db.Collection(constants.MessagesCollection)

	fmt.Println("data.Days", data.Days)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"createdAt": bson.M{
					"$gte": time.Now().AddDate(0, 0, -data.Days),
				},
			},
		},
		{
			"$group": bson.M{
				"_id": "$fromUserId",
				"count": bson.M{
					"$sum": 1,
				},
				"lastMessage": bson.M{
					"$last": "$createdAt",
				},
			},
		},
		{
			"$sort": bson.M{
				"count": -1,
			},
		},
		{
			"$lookup": bson.M{
				"from":         constants.UsersCollection,
				"localField":   "_id",
				"foreignField": "id",
				"as":           "user",
			},
		},
		{
			"$unwind": "$user",
		},
		{
			"$project": bson.M{
				"_id":           0,
				"id":            "$user.id",
				"nickname":      "$user.nickname",
				"status":        "$user.status",
				"activity":      "$user.activity",
				"createdAt":     "$user.createdAt",
				"updatedAt":     "$user.updatedAt",
				"messageCount":  "$count",
				"lastMessageAt": "$lastMessage",
			},
		},
		{"$skip": data.Skip},
		{"$limit": data.Limit},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Error(ctx, "Failed to get users who sent messages in the last 30 days", log.ErrAttr(err))
		return nil, err
	}

	return cursor, nil
}
