package deps

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/api/constants"
	"github.com/vit0rr/chat/config"
	"github.com/vit0rr/chat/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func NewRedisClient(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	opt, _ := redis.ParseURL(cfg.API.Redis.Dsn)
	redisClient := redis.NewClient(opt)

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
				bson.M{"id": bson.M{"$in": members}},
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

func CheckAndUpdateMessageRateLimit(ctx context.Context, redisClient *redis.Client, userID string, delay time.Duration) (bool, float64) {
	lastMsgKey := fmt.Sprintf("rate_limit:%s:last_msg", userID)
	
	lastMsgStr, err := redisClient.Get(ctx, lastMsgKey).Result()
	if err != nil && err != redis.Nil {
		log.Error(ctx, "Failed to check rate limit", log.ErrAttr(err))
		return true, 0
	}
	
	if err == redis.Nil {
		now := time.Now()
		redisClient.Set(ctx, lastMsgKey, now.Format(time.RFC3339Nano), delay*2)
		return true, 0
	}
	
	lastMsgTime, err := time.Parse(time.RFC3339Nano, lastMsgStr)
	if err != nil {
		log.Error(ctx, "Failed to parse last message time", log.ErrAttr(err))
		return true, 0
	}
	
	now := time.Now()
	timeSinceLastMessage := now.Sub(lastMsgTime)
	if timeSinceLastMessage < delay {
		timeToWait := delay - timeSinceLastMessage
		return false, timeToWait.Seconds()
	}
	

	redisClient.Set(ctx, lastMsgKey, now.Format(time.RFC3339Nano), delay*2)
	return true, 0
}
