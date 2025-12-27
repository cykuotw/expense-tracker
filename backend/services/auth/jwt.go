package auth

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func CreateJWT(secret []byte, userID uuid.UUID) (string, error) {
	expiration := time.Second * time.Duration(config.Envs.JWTExpirationInSeconds)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID":    userID.String(),
		"expiredAt": time.Now().Add(expiration).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateJWT(c *gin.Context) error {
	token, err := extractToken(c)

	if err != nil || !token.Valid {
		return types.ErrInvalidToken
	}

	return nil
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := validateJWT(c)
		if err != nil {
			utils.WriteJSON(c, http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func ExtractJWTClaim(c *gin.Context, key string) (string, error) {
	token, err := extractToken(c)
	if err != nil || !token.Valid {
		return "", types.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	claim, exists := claims[key]
	if !exists {
		return "", fmt.Errorf("claim '%s' not found", key)
	}

	claimStr, ok := claim.(string)
	if !ok {
		return "", fmt.Errorf("claim '%s' is not a string", key)
	}

	return claimStr, nil
}

func extractToken(c *gin.Context) (*jwt.Token, error) {
	var tokenStr string

	if cookie, err := c.Cookie("access_token"); err == nil {
		tokenStr = cookie
	} else {
		bearerToken := c.Request.Header.Get("Authorization")
		if bearerToken == "" {
			return nil, types.ErrInvalidToken
		}
		tks := strings.Split(bearerToken, " ")
		if len(tks) != 2 {
			return nil, types.ErrInvalidToken
		}
		tokenStr = tks[1]
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Envs.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
