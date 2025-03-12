package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	Id        string    `json:"id" bson:"_id"`
	Email     string    `json:"email" bson:"email"`
	Password  string    `json:"password" bson:"password"`
	Nickname  string    `json:"nickname" bson:"nickname"`
	Activity  string    `json:"activity" bson:"activity"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateUserData struct {
	ID       string `json:"_id"`
	Nickname string `json:"nickname"`
	Activity string `json:"activity"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type GetUserData struct {
	UserID string
}

type UpdateUserData struct {
	UserID   string
	Nickname *string
	Activity *string
}

type GetAllOnlineUsersData struct {
	Limit int64
	Skip  int64
}

type GetAllOnlineUsersFromARoomData struct {
	RoomID string
}

type GetUserContactsData struct {
	UserID string
	Limit  int64
	Skip   int64
}

func CreateUser(ctx context.Context, db *mongo.Database, data CreateUserData) (*mongo.InsertOneResult, error) {
	now := time.Now()

	id := primitive.NewObjectID().Hex()

	collection := db.Collection(constants.UsersCollection)

	user, err := collection.InsertOne(ctx, User{
		Id:        id,
		Nickname:  data.Nickname,
		Activity:  data.Activity,
		Password:  data.Password,
		Email:     data.Email,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if err != nil {
		log.Error(ctx, constants.ErrorMessages[constants.FailedToCreateUser].Message, log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToCreateUser].Message)
	}

	user.InsertedID = id

	return user, nil
}

func GetUser(ctx context.Context, db *mongo.Database, data GetUserData) (*User, error) {
	collection := db.Collection(constants.UsersCollection)
	options := options.FindOne()
	filter := bson.M{"_id": data.UserID}

	user := User{}
	err := collection.FindOne(ctx, filter, options).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Error(ctx, constants.ErrorMessages[constants.FailedToGetUsers].Message, log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetUsers].Message)
	}

	return &user, nil
}

func GetAllOnlineUsersFromARoom(ctx context.Context, db *mongo.Database, data GetAllOnlineUsersFromARoomData) ([]User, error) {
	collection := db.Collection(constants.UsersCollection)

	room, err := GetRoom(ctx, db, GetRoomData{RoomID: data.RoomID})
	if err != nil {
		log.Error(ctx, "Failed to get room", log.ErrAttr(err))
		return nil, err
	}

	usersIds := []string{}
	for _, userObj := range room.Users {
		usersIds = append(usersIds, userObj.ID)
	}

	cursor, err := collection.Find(ctx, bson.M{"_id": bson.M{"$in": usersIds}, "activity": "online"})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(constants.ErrorMessages[constants.UserNotFound].Message)
		}

		log.Error(ctx, constants.ErrorMessages[constants.FailedToGetUsers].Message, log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetUsers].Message)
	}

	users := []User{}
	err = cursor.All(ctx, &users)
	if err != nil {
		log.Error(ctx, constants.ErrorMessages[constants.FailedToGetUsers].Message, log.ErrAttr(err))
		return nil, errors.New(constants.ErrorMessages[constants.FailedToGetUsers].Message)
	}

	return users, nil
}

func GetAllOnlineUsers(ctx context.Context, db *mongo.Database, data GetAllOnlineUsersData) (*mongo.Cursor, error) {
	collection := db.Collection(constants.UsersCollection)

	options := options.Find()
	options.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	options.SetLimit(data.Limit)
	options.SetSkip(data.Skip)

	cursor, err := collection.Find(ctx, bson.M{"activity": "online"})
	if err != nil {
		log.Error(ctx, "Failed to get all online users", log.ErrAttr(err))
	}

	return cursor, nil

}

func GetUsers(ctx context.Context, db *mongo.Database, data GetUserData) (*mongo.Cursor, error) {
	collection := db.Collection(constants.UsersCollection)
	options := options.Find()

	cursor, err := collection.Find(ctx, options)
	if err != nil {
		log.Error(ctx, "Failed to get users", log.ErrAttr(err))
		return nil, err
	}
	return cursor, nil
}

func UpdateUser(ctx context.Context, db *mongo.Database, data UpdateUserData) (*mongo.UpdateResult, error) {
	user, err := GetUser(ctx, db, GetUserData{UserID: data.UserID})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	collection := db.Collection(constants.UsersCollection)
	filter := bson.M{"_id": data.UserID}

	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if data.Nickname != nil {
		update["$set"].(bson.M)["nickname"] = *data.Nickname
	}

	if data.Activity != nil {
		update["$set"].(bson.M)["activity"] = *data.Activity
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Error(ctx, "Failed to update user", log.ErrAttr(err))
		return nil, err
	}

	return result, nil
}

func GetUserContacts(ctx context.Context, db *mongo.Database, data GetUserContactsData) (*mongo.Cursor, error) {
	roomsCollection := db.Collection(constants.RoomsCollection)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"users.id": data.UserID,
			},
		},
		{
			"$unwind": "$users",
		},
		{
			"$match": bson.M{
				"users.id": bson.M{
					"$ne": data.UserID,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":      "$users.id",
				"nickname": bson.M{"$first": "$users.nickname"},
			},
		},
		{
			"$lookup": bson.M{
				"from":         constants.UsersCollection,
				"localField":   "_id",
				"foreignField": "id",
				"as":           "userDetails",
			},
		},
		{
			"$unwind": "$userDetails",
		},
		{
			"$project": bson.M{
				"_id":      0,
				"nickname": "$userDetails.nickname",
				"status":   "$userDetails.status",
				"activity": "$userDetails.activity",
			},
		},
		{"$skip": data.Skip},
		{"$limit": data.Limit},
	}

	cursor, err := roomsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Error(ctx, "Failed to get user contacts", log.ErrAttr(err))
		return nil, err
	}

	return cursor, nil
}

func GetUserByEmail(ctx context.Context, db *mongo.Database, email string) (*User, error) {
	collection := db.Collection(constants.UsersCollection)
	filter := bson.M{"email": email}

	var user User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments
		}
		log.Error(ctx, "Failed to get user by email", log.ErrAttr(err))
		return nil, err
	}

	return &user, nil
}

func DeleteUser(ctx context.Context, db *mongo.Database, userID string) error {
	collection := db.Collection(constants.UsersCollection)
	filter := bson.M{"_id": userID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		log.Error(ctx, "Failed to delete user", log.ErrAttr(err))
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("user not found")
	}

	return nil
}
