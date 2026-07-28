package controllers

import (
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/services"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	errUserNotFound = "User tidak ditemukan"
	errHashPassword = "Gagal mengenkripsi password"
)

var allowedRoles = map[string]bool{
	"superadmin": true,
	"admin":      true,
	"staff":      true,
}

type StorePushTokenRequest struct {
	Token string `json:"token"`
}

func hashPassword(pass string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(hashed), err
}

// HELPER REFACTOR COGITIVE COMPLEXITY - REFACTOR THESIS
func handlePasswordUpdate(user models.User,
	oldPassword string,
	newPassword string,
	updates map[string]interface{},
) error {
	if oldPassword == "" && newPassword == "" {
		return nil
	}

	if oldPassword == "" || newPassword == "" {
		return fmt.Errorf("Password lama dan baru harus diisi")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("Password lama salah")
	}

	if len(newPassword) < 6 {
		return fmt.Errorf("Password baru minimal 6 karakter")
	}

	hashed, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf(errHashPassword)
	}

	updates["password"] = hashed

	return nil
}

// CREATE USERS
func CreateUserWithRole(c *gin.Context, role string) {
	if !allowedRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ID = uuid.NewString()
	input.Role = role

	hashed, err := hashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errHashPassword})
		return
	}
	input.Password = hashed

	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user"})
		return
	}

	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"create",
		"Menambahkan user baru: "+input.Name+" sebagai "+role,
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User berhasil dibuat",
		"user":    input,
	})
}

func CreateSuperAdmin(c *gin.Context) { CreateUserWithRole(c, "superadmin") }

// INJECT SUPERADMIN
// !!! DELETE AFTER DEPLOYMENT !!!!
// func CreateSuperAdmin(c *gin.Context) {
// 	var count int64
// 	config.DB.Model(&models.User{}).
// 		Where("role = ?", "superadmin").
// 		Count(&count)

// 	if count > 0 {
// 		c.JSON(http.StatusForbidden, gin.H{
// 			"error": "Superadmin sudah ada",
// 		})
// 		return
// 	}

// 	var input models.User
// 	if err := c.ShouldBindJSON(&input); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	input.ID = uuid.NewString()
// 	input.Role = "superadmin"

// 	hashed, err := hashPassword(input.Password)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash password"})
// 		return
// 	}
// 	input.Password = hashed

// 	if err := config.DB.Create(&input).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat superadmin"})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"message": "Superadmin pertama berhasil dibuat",
// 	})
// }

func CreateAdmin(c *gin.Context) { CreateUserWithRole(c, "admin") }
func CreateStaff(c *gin.Context) { CreateUserWithRole(c, "staff") }

// READ ALL USERS
func GetUsers(c *gin.Context) {
	var users []models.User

	result := config.DB.Select("id", "name", "username", "role", "created_at", "updated_at", "photo_url").
		Order("created_at DESC").
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data users: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// READ BY ID
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.Select("id", "name", "username", "role", "created_at", "updated_at").
		Where(idQuery, id).First(&user).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GET USERS FOR FILTER
func GetUsersForFilter(c *gin.Context) {
	var users []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	result := config.DB.Model(&models.User{}).
		Select("id", "name").
		Where("role IN ?", []string{"staff", "admin", "superadmin"}). // Sesuaikan dengan role yang diinginkan
		Order("name ASC").
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data users: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GET ME
func GetMe(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	user := userRaw.(models.User)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"name":      user.Name,
			"username":  user.Username,
			"role":      user.Role,
			"photo_url": user.PhotoURL,
		},
	})
}

// UPDATE USERS
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.Where(idQuery, id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
		return
	}

	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")

	var input struct {
		Name     string `form:"name"`
		Username string `form:"username"`
		Role     string `form:"role"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid: " + err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}

	if input.Username != "" {
		var count int64
		config.DB.Model(&models.User{}).
			Where("username = ? AND id != ?", input.Username, id).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah digunakan"})
			return
		}

		updates["username"] = input.Username
	}

	if input.Role != "" {
		updates["role"] = input.Role
	}

	// Password logic - REFACTOR COGNITIVE COMPLEXITY
	if err := handlePasswordUpdate(
		user,
		oldPassword,
		newPassword,
		updates,
	); err != nil {

		switch err.Error() {

		case "Password lama dan baru harus diisi":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		case "Password lama salah":
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})

		case "Password baru minimal 6 karakter":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		return
	}

	// END REFACTOR COGNITIVE COMPLEXITY

	// Photo upload
	file, err := c.FormFile("photo")
	if err == nil {
		if file.Size > 5<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 5MB"})
			return
		}

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
			return
		}
		defer f.Close()

		userID := user.ID
		originalFileName := file.Filename

		ext := ""
		if dot := strings.LastIndex(originalFileName, "."); dot != -1 {
			ext = originalFileName[dot:]
		}

		timestamp := time.Now().Unix()
		uniqueFileName := fmt.Sprintf("user-%s-%d%s", userID[:8], timestamp, ext)

		uploadRes, err := config.UploadToCloudinary(f, uniqueFileName, "users", "image")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto: " + err.Error()})
			return
		}

		if user.PhotoID != nil && *user.PhotoID != "" {
			if *user.PhotoID != uploadRes.PublicID {
				config.DeleteFromCloudinary(*user.PhotoID, "image")
			}
		}

		updates["photo_url"] = uploadRes.SecureURL
		updates["photo_id"] = uploadRes.PublicID
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update user"})
			return
		}
	}

	config.DB.Where(idQuery, id).First(&user)

	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"update",
		"Mengupdate user: "+user.Name,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "User berhasil diperbarui",
		"user":    user,
	})
}

// RESET PASSWORD (only superadmin)
func ResetPassword(c *gin.Context) {
	id := c.Param("id")

	currentUser := c.MustGet("user").(models.User)
	if currentUser.Role != "superadmin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya superadmin yang dapat reset password"})
		return
	}

	var user models.User
	if err := config.DB.Where(idQuery, id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
		return
	}

	defaultPassword := "123456"
	hashed, err := hashPassword(defaultPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errHashPassword})
		return
	}

	if err := config.DB.Model(&user).Update("password", hashed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal reset password"})
		return
	}

	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"update",
		"Reset password user: "+user.Name+" ke default",
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Password berhasil direset ke default (123456)",
	})
}

// DELETE USERS
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := config.DB.Where(idQuery, id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errUserNotFound})
		return
	}

	if err := config.DB.Delete(&models.User{}, idQuery, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus user"})
		return
	}

	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"delete",
		"Menghapus user: "+user.Name,
	)

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
}

// PUSH TOKEN
func StorePushToken(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		log.Println("[PushToken] Unauthorized user")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user := userRaw.(models.User)
	log.Println("[PushToken] User:", user.ID)

	var req StorePushTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("[PushToken] Invalid JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	log.Println("[PushToken] Received token:", req.Token)

	if req.Token == "" {
		log.Println("[PushToken] Empty token received")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	if err := config.DB.Model(&models.User{}).
		Where(idQuery, user.ID).
		Update("push_token", req.Token).Error; err != nil {
		log.Println("[PushToken] DB update error:", err)
		return
	}

	log.Println("[PushToken] Token saved SUCCESSFULLY for user:", user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Push token stored successfully",
	})
}
