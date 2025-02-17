package server

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/api/router"
	"github.com/vit0rr/chat/pkg/deps"
	"go.mongodb.org/mongo-driver/mongo"
)

func New(ctx context.Context, deps *deps.Deps, db *mongo.Database, redisClient *redis.Client) *http.Server {
	router := router.New(deps, db, redisClient)

	return &http.Server{
		Addr:              deps.Config.Server.BindAddr,
		Handler:           router.BuildRoutes(deps),
		ReadHeaderTimeout: 10 * time.Second,
	}
}
