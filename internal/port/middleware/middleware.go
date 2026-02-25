// Package middleware ...
package middleware

import "net/http"

// Middleware ...
type Middleware func(http.Handler) http.Handler

var chain []Middleware

// Use ...
func Use(mw Middleware) {
	chain = append(chain, mw)
}

// Apply ...
func Apply(next http.Handler) http.Handler {
	for i := len(chain) - 1; i >= 0; i-- {
		next = chain[i](next)
	}
	return next
}
