package deps

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/config"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewRedisClient(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.API.Redis.Dsn,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return redisClient, nil
}

func CleanupStaleRooms(ctx context.Context, redisClient *redis.Client) {
	pattern := "room:*:clients"
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		roomKey := iter.Val()
		if members, _ := redisClient.SCard(ctx, roomKey).Result(); members == 0 {
			if err := redisClient.Del(ctx, roomKey).Err(); err != nil {
				log.Error(ctx, "Failed to delete empty room", log.ErrAttr(err))
			}
		}
	}
}

func RecoverUserStatuses(ctx context.Context, db *mongo.Database, redisClient *redis.Client) error {
	// First, set all users to offline
	if err := UpdateAllOnlineUsersToOffline(ctx, db); err != nil {
		return err
	}

	// Then check Redis for actually connected users and update them
	pattern := "room:*:online"
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		roomKey := iter.Val()
		members, err := redisClient.SMembers(ctx, roomKey).Result()
		if err != nil {
			continue
		}

		// Update these users to online since they have active Redis connections
		if len(members) > 0 {
			collection := db.Collection(constants.UsersCollection)
			_, err = collection.UpdateMany(
				ctx,
				bson.M{"externalId": bson.M{"$in": members}},
				bson.M{"$set": bson.M{
					"activity":  "online",
					"updatedAt": time.Now(),
				}},
			)
			if err != nil {
				log.Error(ctx, "Failed to update recovered users", log.ErrAttr(err))
			}
		}
	}

	return nil
}
