package handler

import (
	"encoding/json"
	"net/http"

	"github.com/vit0rr/chat/pkg/log"
)

// Handler is a type to allow functions to act as Handlers.
// Ideally, all of our HTTP handlers should be wrapped with this type to have a common
// way for error handling, logging, etc.
type Handler func(http.ResponseWriter, *http.Request) (interface{}, error)

func handleError(r *http.Request, err error, w http.ResponseWriter) {}

// ServeHTTP executes the handler function and handles potential errors as well as writing potential responses to http.ResponseWriter
func (fn Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := fn(w, r)
	if err != nil {
		handleError(r, err, w)
		return
	}

	if resp != nil {
		w.WriteHeader(http.StatusOK)
		res, err := json.Marshal(resp)
		if err != nil {
			log.Error(r.Context(), "Handler: failed to marshal response body", log.ErrAttr(err))
		}

		_, err = w.Write(res)
		if err != nil {
			log.Error(r.Context(), "Handler: failed to write response body", log.ErrAttr(err))
		}
	}
}
