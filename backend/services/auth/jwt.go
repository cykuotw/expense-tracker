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

type Claims struct {
	UserID string `json:"userID"`
	jwt.RegisteredClaims
}

func CreateJWT(secret []byte, userID uuid.UUID) (string, error) {
	expiration := time.Second * time.Duration(config.Envs.JWTExpirationInSeconds)
	now := time.Now()
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func validateJWT(c *gin.Context) error {
	_, err := extractToken(c)
	if err != nil {
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
	claims, err := extractToken(c)
	if err != nil {
		return "", types.ErrInvalidToken
	}

	switch key {
	case "userID":
		if claims.UserID == "" {
			return "", fmt.Errorf("claim '%s' not found", key)
		}
		return claims.UserID, nil
	default:
		return "", fmt.Errorf("claim '%s' not found", key)
	}
}

func extractToken(c *gin.Context) (*Claims, error) {
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

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.Envs.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, types.ErrInvalidToken
	}

	return claims, nil
}
