package middleware

import (
	"context"
	"fmt"
	"mini-project/internal/utils"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			utils.WriteError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 1. Parse the token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is what we expect (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		})

		// 2. Check if valid
		if err != nil || !token.Valid {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// 3. Inject UserID into Context
		// This is the "Secret Sauce": now every handler knows WHO is calling
		userID, ok := claims["user_id"].(string)
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}