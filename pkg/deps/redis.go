package deps

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/config"
	"github.com/vit0rr/chat/pkg/log"
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
