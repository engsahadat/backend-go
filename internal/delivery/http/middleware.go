package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/your-org/ai-employee-platform/internal/service"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware validates the JWT in the Authorization header
// and injects the user ID into the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		userID, err := service.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext extracts the authenticated user ID from the context.
func GetUserIDFromContext(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}
