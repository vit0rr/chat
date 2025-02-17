package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger" // http-swagger middleware
	chatService "github.com/vit0rr/chat/api/internal/chat-service"
	_ "github.com/vit0rr/chat/docs"
	"github.com/vit0rr/chat/pkg/deps"
	"github.com/vit0rr/chat/pkg/telemetry"
	"go.mongodb.org/mongo-driver/mongo"
)

type Router struct {
	Deps        *deps.Deps
	chatService *chatService.HTTP
}

func (router *Router) BuildRoutes(deps *deps.Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)

	r.Use(telemetry.TelemetryMiddleware)
	r.Use(chatService.JSONResponseMiddleware)

	swgUrl := func() string {
		if deps.Config.Env.Env == "production" || deps.Config.Env.Env == "homologation" {
			return fmt.Sprintf("https://%s/swagger/doc.json", deps.Config.Env.Host)
		}

		return fmt.Sprintf("http://%s:%s/swagger/doc.json", deps.Config.Env.Host, deps.Config.Env.Port)
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/ws", telemetry.HandleFuncLogger(router.chatService.WebSocket))
		r.Post("/register-user", telemetry.HandleFuncLogger(router.chatService.RegisterUser))
		r.Get("/messages", telemetry.HandleFuncLogger(router.chatService.GetMessages))
		r.Post("/rooms/lock", telemetry.HandleFuncLogger(router.chatService.LockRoom))
	})

	r.Group(func(r chi.Router) {
		r.Use(SetResponseTypeToJSON)

		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL(swgUrl()),
		))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			http.ServeFile(w, r, "./api/router/index.html")
		})
	})

	return r
}

func New(deps *deps.Deps, db *mongo.Database, redisClient *redis.Client) *Router {
	return &Router{
		Deps: deps,
		chatService: chatService.NewHTTP(
			deps,
			db,
			redisClient,
		),
	}
}
