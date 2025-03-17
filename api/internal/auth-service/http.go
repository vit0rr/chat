package authservice

import (
	"fmt"
	"net/http"

	"github.com/vit0rr/chat/pkg/deps"
	"go.mongodb.org/mongo-driver/mongo"
)

type HTTP struct {
	service *Service
}

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func NewHTTP(deps *deps.Deps, db *mongo.Database) *HTTP {
	return &HTTP{
		service: NewService(deps, db),
	}
}

func (h *HTTP) Register(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	result, err := h.service.Register(r.Context(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}

func (h *HTTP) Login(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	result, err := h.service.Login(r.Context(), r.Body)

	authHeader := r.Header.Get("Authorization")
	fmt.Println("authHeader", authHeader)
	fmt.Println("h.service.deps.Config.APIKey", h.service.deps.Config.APIKey)
	if authHeader == "" || authHeader != fmt.Sprintf("Bearer %s", h.service.deps.Config.APIKey) {
		w.WriteHeader(http.StatusUnauthorized)
		return ErrorResponse{
			Error: "Authorization header required",
			Code:  http.StatusUnauthorized,
		}, nil
	}

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusUnauthorized,
		}, nil
	}
	return result, nil
}

func (h *HTTP) DeleteUser(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	result, err := h.service.DeleteUser(r.Context(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return ErrorResponse{
			Error: err.Error(),
			Code:  http.StatusBadRequest,
		}, nil
	}
	return result, nil
}
