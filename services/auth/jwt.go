package auth

import (
	"expense-tracker/config"
	"expense-tracker/types"
	"expense-tracker/utils"
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

func ValidateJWT(c *gin.Context) error {
	token, err := extractToken(c)

	if err != nil || !token.Valid {
		return types.ErrInvalidToken
	}

	return nil
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := ValidateJWT(c)
		if err != nil {
			utils.WriteJSON(c, http.StatusUnauthorized, "Unauthorized")
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

	claims := token.Claims.(jwt.MapClaims)
	claim := claims[key].(string)

	return claim, nil
}

func extractToken(c *gin.Context) (*jwt.Token, error) {
	bearerToken := c.Request.Header.Get("Authorization")
	if bearerToken == "" {
		return nil, types.ErrInvalidToken
	}
	tks := strings.Split(bearerToken, " ")
	if len(tks) != 2 {
		return nil, types.ErrInvalidToken
	}
	tokenStr := tks[1]

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
