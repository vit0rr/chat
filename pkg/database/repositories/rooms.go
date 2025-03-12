package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Room struct {
	ID        string    `bson:"_id" json:"id"`
	Users     []UserRef `bson:"users" json:"users"`
	LockedBy  string    `bson:"lockedBy,omitempty" json:"lockedBy,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type CreateRoomData struct {
	UserID   string `json:"userId"`
	RoomID   string `json:"roomId"`
	Nickname string `json:"nickname"`
}

type GetRoomData struct {
	RoomID string `json:"roomId"`
}

type GetRoomsCursorData struct {
	Limit int64
	Skip  int64
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
			"users": UserRef{
				ID:       data.UserID,
				Nickname: data.Nickname,
			},
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Error(ctx, constants.ErrorMessages[constants.FailedToCreateOrUpdateRoom].Message, log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToCreateOrUpdateRoom].Message)
	}

	return result, nil
}

func GetRooms(ctx context.Context, db *mongo.Database, data GetRoomData) (*Room, error) {
	collection := db.Collection(constants.RoomsCollection)

	var room Room
	filter := bson.M{"_id": data.RoomID}

	err := collection.FindOne(ctx, filter).Decode(&room)
	// fmt.Println("eraaaar", err)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Error(ctx, "Failed to get room", log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetRooms].Message)
	}

	return &room, nil
}

func GetRoom(ctx context.Context, db *mongo.Database, data GetRoomData) (*Room, error) {
	collection := db.Collection(constants.RoomsCollection)

	var room Room
	filter := bson.M{"_id": data.RoomID}

	err := collection.FindOne(ctx, filter).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(constants.ErrorMessages[constants.RoomNotFound].Message)
		}
		log.Error(ctx, "Failed to get room", log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetRooms].Message)
	}

	return &room, nil
}

func GetRoomsCursor(ctx context.Context, db *mongo.Database, data GetRoomsCursorData) (*mongo.Cursor, error) {
	collection := db.Collection(constants.RoomsCollection)

	options := options.Find()
	options.SetSort(bson.D{{Key: "createdAt", Value: -1}}) // Sort by newest first
	options.SetLimit(data.Limit)
	options.SetSkip(data.Skip)

	cursor, err := collection.Find(ctx, bson.M{}, options)
	if err == mongo.ErrNoDocuments {
		log.Error(ctx, "Room not found", log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.RoomNotFound].Message)
	}

	if err != nil {
		log.Error(ctx, "Failed to get rooms", log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetRooms].Message)
	}

	return cursor, nil
}

func GetAllRoomsWhereUserIsRegistered(ctx context.Context, db *mongo.Database, data GetUserData) ([]Room, error) {
	collection := db.Collection(constants.RoomsCollection)

	opts := options.Find().SetProjection(bson.M{"users": 0})

	cursor, err := collection.Find(ctx, bson.M{"users.id": data.UserID}, opts)
	if err != nil {
		log.Error(ctx, "Failed to get all rooms where user is registered", log.ErrAttr(err))
		return nil, err
	}

	rooms := []Room{}
	err = cursor.All(ctx, &rooms)
	if err != nil {
		log.Error(ctx, "Failed to get all rooms where user is registered", log.ErrAttr(err))
		return nil, err
	}
	return rooms, nil
}
