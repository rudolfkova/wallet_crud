// Package port ...
package port

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	walleterror "wallet/internal/error"
	"wallet/internal/port/middleware"
)

type ServerAPI struct {
	logger *slog.Logger
}

// Logger ...
func (s *ServerAPI) Logger() *slog.Logger {
	return s.logger
}

// NewServer ...
func NewServer(logger *slog.Logger) *ServerAPI {
	return &ServerAPI{logger: logger}
}

// ErrorResponse ...
type ErrorResponse struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// Respond ...
func (s *ServerAPI) Respond(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	const op = "ServerAPI.Respond"

	w.WriteHeader(code)

	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log := s.Logger().With(
				slog.String("op:", op),
				slog.String("requestID:", middleware.GetRequestIDFromRequest(r)),
			)
			log.Warn(err.Error())
		}
	}
}

// Error ...
func (s *ServerAPI) Error(w http.ResponseWriter, r *http.Request, op string, err error) {
	var resp ErrorResponse
	log := s.Logger().With(
		slog.String("op", op),
		slog.String("requestID:", middleware.GetRequestIDFromRequest(r)),
	)
	if err != nil {
		log.Warn(err.Error())
	}

	var code int
	switch {
	case errors.Is(err, walleterror.ErrWalletNotFound):
		code = http.StatusNotFound
		resp = ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "wallet not found",
		}
	case errors.Is(err, walleterror.ErrInsufficientFunds):
		code = http.StatusConflict
		resp = ErrorResponse{
			Code:    "CONFLICT",
			Message: "insufficient funds",
		}
	case errors.Is(err, walleterror.ErrInvalidOperationType):
		code = http.StatusBadRequest
		resp = ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "invalid operation type",
		}
	case errors.Is(err, walleterror.ErrTypeNotSpecified):
		code = http.StatusBadRequest
		resp = ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "type operation not specified",
		}
	case errors.Is(err, walleterror.ErrInvalidAmount):
		code = http.StatusBadRequest
		resp = ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "invalid amount",
		}
	case errors.Is(err, walleterror.ErrInvalidValletID):
		code = http.StatusBadRequest
		resp = ErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "invalid vallet id",
		}
	default:
		code = http.StatusInternalServerError
		resp = ErrorResponse{
			Message: "Internal server error",
		}
	}

	s.Respond(w, r, code, resp)
}
