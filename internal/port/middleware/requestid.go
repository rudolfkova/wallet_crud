// Package middleware ...
package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey struct{}

var requestIDKey = ctxKey{}

// RequestID ...
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, reqID)

		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID ...
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// GetRequestIDFromRequest ...
func GetRequestIDFromRequest(r *http.Request) string {
	return GetRequestID(r.Context())
}
