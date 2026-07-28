package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Input tidak valid"})
		return
	}

	db := config.DB

	var user models.User
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Username atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Username atau password salah"})
		return
	}

	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "default_secret"
	}

	// REFACTOR PERFORMANCE: tambah jti (UUID) agar setiap JWT unik
	// Sebelumnya: tanpa jti -> 2 request di detik yang sama menghasilkan JWT identical
	// -> hash SHA256 identical -> Error 1062 duplicate entry di tabel secret_tokens
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(14 * 24 * time.Hour).Unix(),
		"role":    user.Role,
		"jti":     uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat token"})
		return
	}

	hash := sha256.Sum256([]byte(signedToken))
	hashedToken := hex.EncodeToString(hash[:])

	device := c.GetHeader("X-Device")
	if device == "" {
		device = "unknown"
	}

	// REFACTOR PERFORMANCE: atomic transaction untuk mencegah race condition
	// Sebelumnya: DELETE + INSERT sebagai statement terpisah -> goroutine A hapus,
	// goroutine B hapus token yang sama, lalu INSERT bareng -> salah satu gagal
	// Sekarang: semua operasi dalam 1 transaksi, kalo gagal di-rollback.
	secretToken := models.SecretToken{
		JwtToken:  hashedToken,
		UserID:    user.ID,
		Device:    device,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		// Hapus token expired
		tx.Where("expires_at < ?", time.Now()).Delete(&models.SecretToken{})

		// Hapus token lama device yang sama (1 device 1 session)
		tx.Where("user_id = ? AND device = ?", user.ID, device).Delete(&models.SecretToken{})

		// Batasi maksimal 2 token per user (hapus paling tua)
		var userTokens []models.SecretToken
		tx.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&userTokens)
		if len(userTokens) >= 2 {
			oldest := userTokens[len(userTokens)-1]
			tx.Delete(&oldest)
		}

		// Insert token baru
		return tx.Create(&secretToken).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login berhasil",
		"token":    signedToken,
		"token_id": secretToken.ID,
		"user": gin.H{
			"id":        user.ID,
			"name":      user.Name,
			"username":  user.Username,
			"role":      user.Role,
			"photo_url": user.PhotoURL,
		},
	})
}
