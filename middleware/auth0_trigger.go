package middleware

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateTriggerToken validates a JWT signed with AUTH0_TRIGGER_SECRET (HMAC).
// Returns nil if valid, or an error describing the failure.
func ValidateTriggerToken(authHeader string) error {
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}

	tokenString := authHeader
	// Support both raw token and "Bearer <token>" format
	if parts := strings.SplitN(authHeader, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		tokenString = parts[1]
	}

	secret := os.Getenv("AUTH0_TRIGGER_SECRET")
	if secret == "" {
		return fmt.Errorf("server misconfiguration: trigger secret not set")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return fmt.Errorf("invalid or expired trigger token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("invalid trigger token")
	}

	return nil
}
