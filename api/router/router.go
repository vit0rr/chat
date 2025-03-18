package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger" // http-swagger middleware
	authService "github.com/vit0rr/chat/api/internal/auth-service"
	chatService "github.com/vit0rr/chat/api/internal/chat-service"
	_ "github.com/vit0rr/chat/docs"
	"github.com/vit0rr/chat/pkg/deps"
	pkgMiddlware "github.com/vit0rr/chat/pkg/middleware"
	"github.com/vit0rr/chat/pkg/telemetry"
	"go.mongodb.org/mongo-driver/mongo"
)

type Router struct {
	Deps        *deps.Deps
	chatService *chatService.HTTP
	authService *authService.HTTP
}

func (router *Router) BuildRoutes(deps *deps.Deps) *chi.Mux {
	r := chi.NewRouter()

	allowedOrigins := strings.Split(deps.Config.Env.AllowedOrigins, ",")

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

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

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", telemetry.HandleFuncLogger(router.authService.Register))
			r.Post("/login", telemetry.HandleFuncLogger(router.authService.Login))
			r.With(pkgMiddlware.JWTAuth(deps)).Delete("/user", telemetry.HandleFuncLogger(router.authService.DeleteUser))
		})

		r.Group(func(r chi.Router) {
			r.Use(pkgMiddlware.JWTAuth(deps))

			r.Get("/ws", telemetry.HandleFuncLogger(router.chatService.WebSocket))

			r.Route("/rooms", func(r chi.Router) {
				r.Use(pkgMiddlware.VerifyApiKey(deps))
				r.Get("/", telemetry.HandleFuncLogger(router.chatService.GetRooms))
				r.Get("/{roomId}", telemetry.HandleFuncLogger(router.chatService.GetRoom))
				r.Get("/{roomId}/messages", telemetry.HandleFuncLogger(router.chatService.GetMessages))
				r.Get("/{roomId}/online-users", telemetry.HandleFuncLogger(router.chatService.GetOnlineUsersFromARoom))
				r.Post("/{roomId}/register-user", telemetry.HandleFuncLogger(router.chatService.RegisterUser))
				r.Post("/{roomId}/lock", telemetry.HandleFuncLogger(router.chatService.LockRoom))
			})
			r.Route("/users", func(r chi.Router) {
				r.Use(pkgMiddlware.VerifyApiKey(deps))
				r.Get("/", telemetry.HandleFuncLogger(router.chatService.GetUsers))
				r.Get("/{userId}", telemetry.HandleFuncLogger(router.chatService.GetUser))
				r.Get("/all-online-users", telemetry.HandleFuncLogger(router.chatService.GetOnlineUsersFromAllRooms))
				r.Get("/{userId}/contacts", telemetry.HandleFuncLogger(router.chatService.GetUserContacts))
				r.Post("/create-user", telemetry.HandleFuncLogger(router.chatService.CreateUser))
				r.Patch("/{userId}", telemetry.HandleFuncLogger(router.chatService.UpdateUser))
			})

			r.Route("/messages", func(r chi.Router) {
				r.Use(pkgMiddlware.VerifyApiKey(deps))
				r.Route("/total-sent", func(r chi.Router) {
					r.Get("/", telemetry.HandleFuncLogger(router.chatService.GetTotalMessagesSent))
					r.Get("/{roomId}", telemetry.HandleFuncLogger(router.chatService.GetTotalMessagesSentInARoom))
				})
			})

			r.Route("/analytics", func(r chi.Router) {
				r.Use(pkgMiddlware.VerifyApiKey(deps))
				r.Get("/user-last-messages", telemetry.HandleFuncLogger(router.chatService.GetUsersWhoSentMessagesInTheLastDays))
			})
		})
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
		authService: authService.NewHTTP(
			deps,
			db,
		),
	}
}
