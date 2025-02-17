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

type Room struct {
	ID        string    `bson:"_id" json:"id"`
	Users     []User    `bson:"users" json:"users"`
	LockedBy  string    `bson:"lockedBy,omitempty" json:"lockedBy,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type User struct {
	UserID   string    `bson:"userId" json:"userId"`
	Nickname string    `bson:"nickname" json:"nickname"`
	JoinedAt time.Time `bson:"joinedAt" json:"joinedAt"`
}

type CreateRoomData struct {
	UserID   string `json:"userId"`
	RoomID   string `json:"roomId"`
	Nickname string `json:"nickname"`
}

type GetRoomData struct {
	RoomID string `json:"roomId"`
}

func CreateRoom(ctx context.Context, db *mongo.Database, data CreateRoomData) (*mongo.UpdateResult, error) {
	now := time.Now()
	collection := db.Collection(constants.RoomsCollection)

	filter := bson.M{"_id": data.RoomID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
		"$set": bson.M{
			"updatedAt": now,
		},
		"$addToSet": bson.M{
			"users": User{
				UserID:   data.UserID,
				Nickname: data.Nickname,
				JoinedAt: now,
			},
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Error(ctx, "Failed to create/update room", log.ErrAttr(err))
		return nil, err
	}

	return result, nil
}

func GetRooms(ctx context.Context, db *mongo.Database, data GetRoomData) (*Room, error) {
	collection := db.Collection(constants.RoomsCollection)

	var room Room
	filter := bson.M{"_id": data.RoomID}

	err := collection.FindOne(ctx, filter).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Error(ctx, "Failed to get room", log.ErrAttr(err))
		return nil, err
	}

	return &room, nil
}
