package chatservice

import (
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/pkg/deps"
	"go.mongodb.org/mongo-driver/mongo"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
type HTTP struct {
	service *Service
}

func NewHTTP(deps *deps.Deps, db *mongo.Database, redisClient *redis.Client) *HTTP {
	return &HTTP{
		service: NewService(deps, db, redisClient),
	}
}

func (h *HTTP) WebSocket(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return h.service.WebSocket(w, r)
}

func (h *HTTP) RegisterUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	result, err := h.service.RegisterUser(r.Context(), r.Body, h.service.Mongo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}

func (h *HTTP) GetMessages(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := r.URL.Query().Get("room_id")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	return h.service.GetMessages(r.Context(), GetMessagesQuery{
		RoomID:   roomID,
		PageStr:  pageStr,
		LimitStr: limitStr,
	})
}

func (h *HTTP) LockRoom(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	result, err := h.service.LockRoom(r.Context(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}

func JSONResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
