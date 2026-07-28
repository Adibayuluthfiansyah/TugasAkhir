package middleware

import (
	"crypto/sha256"
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtTokenQuery = "jwt_token = ?"
)

// HELPER REFACTOR COGITIVE COMPLEXITY - REFACTOR THESIS
func abortUnauthorized(c *gin.Context, hashedToken string, message string, deleteToken bool) {
	if deleteToken {
		config.DB.Where(jwtTokenQuery, hashedToken).Delete(&models.SecretToken{})
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})

	c.Abort()
}

func parseJWT(tokenString, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
}

// END HELPER

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//  Ambil header Authorization
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		// HASH token yang diterima untuk dicocokkan dengan database
		hash := sha256.Sum256([]byte(tokenString))
		hashedToken := hex.EncodeToString(hash[:])

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default_secret"
		}

		// REFACTOR START

		token, err := parseJWT(tokenString, secret)
		if err != nil {
			abortUnauthorized(c, hashedToken, "token invalid", true)
			return
		}

		if !token.Valid {
			abortUnauthorized(c, hashedToken, "token invalid", true)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			abortUnauthorized(c, hashedToken, "invalid token claims", true)
			return
		}

		// Validasi exp claim
		exp, ok := claims["exp"].(float64)
		if !ok {
			abortUnauthorized(c, hashedToken, "invalid exp claim", true)
			return
		}

		if time.Now().Unix() > int64(exp) {
			abortUnauthorized(c, hashedToken, "token expired", true)
			return
		}

		// Validasi user_id claim
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			abortUnauthorized(c, hashedToken, "invalid user_id claim", true)
			return
		}

		//  Cek hashed token di database
		var st models.SecretToken
		if err := config.DB.Preload("User").
			Where(jwtTokenQuery, hashedToken).
			First(&st).Error; err != nil {

			abortUnauthorized(c, hashedToken, "session expired", false)
			return
		}

		//  Cek expiration di database
		if time.Now().After(st.ExpiresAt) {
			config.DB.Where(jwtTokenQuery, hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		//  Cek user masih ada
		if st.User.ID == "" {
			abortUnauthorized(c, hashedToken, "user not found", true)
			return
		}

		c.Set("user", st.User)
		c.Next()
	}
}
