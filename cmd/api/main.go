package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/vit0rr/chat/api/server"
	"github.com/vit0rr/chat/config"
	"github.com/vit0rr/chat/pkg/deps"
	"github.com/vit0rr/chat/pkg/log"
	"github.com/vit0rr/chat/shared"
)

// @title Chat API
// @version 1.0
// @description Chat API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
func main() {
	godotenv.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to the config file (see config/config_local.hcl for an example)")
	flag.Parse()

	var cfg config.Config
	if configPath == "" {
		cfg = config.DefaultConfig(cfg)
	} else if configPath != "" {
		parseConfig, err := config.GetConfig(configPath)
		if err != nil {
			log.Error(ctx, "Error parsing config file", "error", err)
			os.Exit(1)
		}

		cfg = parseConfig
	}

	logLevel, err := log.ParseLogLevel(cfg.Server.LogLevel)
	if err != nil {
		log.Error(ctx, "Error parsing log level", "error", err)
		logLevel = slog.LevelInfo
	}
	log.New(ctx, logLevel)

	// create mongo client
	mongoClient, err := deps.NewMongoClient(ctx, cfg)
	if err != nil {
		log.Error(ctx, "❌ Unable to parse database connection", log.ErrAttr(err))
		os.Exit(1)
	}

	db := mongoClient.Database(shared.DatabaseName)

	log.Info(ctx, "✅ Connected to MongoDB")

	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Error(ctx, "❌ Failed to disconnect from MongoDB", log.ErrAttr(err))
			os.Exit(1)
		}
	}()

	if err := deps.CreateKeyIndex(ctx, db); err != nil {
		log.Error(ctx, "❌ Failed to create key index", log.ErrAttr(err))
		os.Exit(1)
	}

	if err := deps.CreateMessagesTTLIndex(ctx, db); err != nil {
		log.Error(ctx, "❌ Failed to create messages TTL index", log.ErrAttr(err))
		os.Exit(1)
	}

	redisClient, err := deps.NewRedisClient(ctx, cfg)
	if err != nil {
		log.Error(ctx, "❌ Failed to create redis client", log.ErrAttr(err))
		os.Exit(1)
	}

	log.Info(ctx, "✅ Connected to Redis")

	dependencies := deps.New(cfg, db)

	if err := deps.RecoverUserStatuses(ctx, db, redisClient); err != nil {
		log.Error(ctx, "❌ Failed to recover user statuses", log.ErrAttr(err))
		os.Exit(1)
	}

	httpServer := server.New(ctx, dependencies, db, redisClient)

	// Start cleanup routine
	go func() {
		ticker := time.NewTicker(10 * time.Minute) // 10m, we can increase/decrease it later
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deps.CleanupStaleRooms(ctx, redisClient)
			}
		}
	}()

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		if err := deps.UpdateAllOnlineUsersToOffline(ctx, db); err != nil {
			log.Error(ctx, "❌ Failed to update all online users to offline", log.ErrAttr(err))
		}

		// We received an interrupt signal, shut down.
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error(ctx, "unexpected error during server shutdown", log.ErrAttr(err))
		}
		close(idleConnsClosed)
	}()

	log.Info(ctx, "🌎 Starting API at", log.AnyAttr("bind_addr", cfg.Server.BindAddr))
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(ctx, "❌ Error starting server", log.ErrAttr(err))
	}

	<-idleConnsClosed

}
