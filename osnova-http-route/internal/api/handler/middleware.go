package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey string

const userDataKey contextKey = "userData"

func (h Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		const prefix = "Bearer "

		if !strings.HasPrefix(authHeader, prefix) {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, prefix)
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			http.Error(w, "Invalid JWT format", http.StatusUnauthorized)
			return
		}

		// Decode the payload (2nd part of the token)
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			http.Error(w, "Failed to decode token payload", http.StatusUnauthorized)
			return
		}

		// Parse the JSON payload into a map
		var claims map[string]any
		if err := json.Unmarshal(payload, &claims); err != nil {
			http.Error(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), userDataKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
