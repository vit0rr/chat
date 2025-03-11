package chatservice

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/vit0rr/chat/pkg/deps"
	"go.mongodb.org/mongo-driver/mongo"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	ErrorID string `json:"error_id"`
}
type HTTP struct {
	service *Service
}

type GetRoomsResponse struct {
	Rooms []RoomDetails `json:"rooms"`
	Error *int          `json:"error,omitempty"`
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
	roomID := chi.URLParam(r, "roomId")

	result, svcErr := h.service.RegisterUser(r.Context(), r.Body, h.service.Mongo, roomID)
	if svcErr.ErrorCode != nil {
		code := http.StatusInternalServerError
		if svcErr.ErrorCode != nil {
			code = *svcErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *svcErr.ErrorMessage,
			Code:    code,
			ErrorID: *svcErr.ErrorID,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetMessages(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := chi.URLParam(r, "roomId")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	result, svcErr := h.service.GetMessages(r.Context(), GetMessagesQuery{
		RoomID:   roomID,
		PageStr:  pageStr,
		LimitStr: limitStr,
	})
	if svcErr.ErrorMessage != nil {
		code := http.StatusInternalServerError
		if svcErr.ErrorCode != nil {
			code = *svcErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *svcErr.ErrorMessage,
			Code:    code,
			ErrorID: *svcErr.ErrorID,
		}, nil
	}

	return result, nil
}

func (h *HTTP) LockRoom(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := chi.URLParam(r, "roomId")

	result, svcErr := h.service.LockRoom(r.Context(), r.Body, roomID)
	if svcErr.ErrorMessage != nil {
		code := http.StatusInternalServerError
		if svcErr.ErrorCode != nil {
			code = *svcErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *svcErr.ErrorMessage,
			Code:    code,
			ErrorID: *svcErr.ErrorID,
		}, nil
	}
	return result, nil
}

func (h *HTTP) GetUsers(w http.ResponseWriter, r *http.Request) (interface{}, error) {

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	result, err := h.service.GetUsers(r.Context(), GetUsersQuery{
		PageStr:  pageStr,
		LimitStr: limitStr,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}

func (h *HTTP) GetUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	userID := chi.URLParam(r, "userId")

	result, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}

func (h *HTTP) UpdateUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ID := chi.URLParam(r, "id")

	_, err := h.service.UpdateUser(r.Context(), ID, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return map[string]interface{}{
		"message": "User updated successfully",
	}, nil
}

func (h *HTTP) GetOnlineUsersFromARoom(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := chi.URLParam(r, "roomId")

	result, svcErr := h.service.GetOnlineUsersFromARoom(r.Context(), roomID)
	if svcErr.ErrorMessage != nil {
		code := http.StatusInternalServerError
		if svcErr.ErrorCode != nil {
			code = *svcErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *svcErr.ErrorMessage,
			Code:    code,
			ErrorID: *svcErr.ErrorID,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetOnlineUsersFromAllRooms(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	result, err := h.service.GetOnlineUsersFromAllRooms(r.Context(), GetOnlineUsersFromAllRoomsQuery{
		PageStr:  pageStr,
		LimitStr: limitStr,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetRoom(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := chi.URLParam(r, "roomId")

	result, roomErr := h.service.GetRoom(r.Context(), roomID)
	if roomErr.ErrorMessage != nil {
		code := http.StatusInternalServerError
		if roomErr.ErrorCode != nil {
			code = *roomErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *roomErr.ErrorMessage,
			Code:    code,
			ErrorID: *roomErr.ErrorID,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetRooms(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	result, roomErr := h.service.GetRooms(r.Context(), GetRoomsQuery{
		PageStr:  pageStr,
		LimitStr: limitStr,
	})

	if roomErr.ErrorMessage != nil {
		code := http.StatusInternalServerError
		if roomErr.ErrorCode != nil {
			code = *roomErr.ErrorCode
		}
		w.WriteHeader(code)
		return ErrorResponse{
			Error:   *roomErr.ErrorMessage,
			Code:    code,
			ErrorID: *roomErr.ErrorID,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetTotalMessagesSent(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	total, err := h.service.GetTotalMessagesSent(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}

	return map[string]interface{}{
		"total": total,
	}, nil
}

func (h *HTTP) GetTotalMessagesSentInARoom(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	roomID := chi.URLParam(r, "roomId")

	total, err := h.service.GetTotalMessagesSentInARoom(r.Context(), roomID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}

	return map[string]interface{}{
		"total": total,
	}, nil
}

func (h *HTTP) GetUsersWhoSentMessagesInTheLastDays(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	days := r.URL.Query().Get("days")

	if days == "" {
		days = "30"
	}

	daysInt, err := strconv.Atoi(days)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}

	result, err := h.service.GetUsersWhoSentMessagesInTheLastDays(r.Context(), GetUsersWhoSentMessagesInTheLastDaysQuery{
		PageStr:  pageStr,
		LimitStr: limitStr,
		Days:     daysInt,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}

	return result, nil
}

func (h *HTTP) GetUserContacts(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ID := chi.URLParam(r, "id")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	result, err := h.service.GetUserContacts(r.Context(), GetUserContactsQuery{
		ID:       ID,
		PageStr:  pageStr,
		LimitStr: limitStr,
	})
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
