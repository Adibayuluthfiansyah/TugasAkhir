package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/middleware"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/routes"
	"dinsos_kuburaya/utils"

	ws "dinsos_kuburaya/websocket"
)

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 100 << 20

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "Server and DB are up and running securely!",
		})
	})

	config.ConnectDatabase()
	utils.StartActivityLogCleaner()
	utils.StartNotificationCleaner()

	if err := config.DB.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.SecretToken{},
		&models.DocumentStaff{},
		&models.Notification{},
		&models.ActivityLog{},
	); err != nil {
		log.Fatal("Gagal migrasi tabel:", err)
	}

	r.Use(middleware.RateLimiter())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.XSSBlocker())

	websocketHub := ws.NewHub()
	go websocketHub.Run()

	ws.HubInstance = websocketHub

	api := r.Group("/api")
	{
		routes.WebSocketRoutes(api, websocketHub)
		routes.LoginRoutes(api)
		routes.LogoutRoutes(api)
		routes.UserRoutes(api)
		routes.DocumentRoutes(api)
		routes.DocumentStaffRoutes(api)
		routes.NotificationRoutes(api)
		routes.ActivityLogRoutes(api)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // fallback lokal
	}

	log.Println("✅ Server berjalan di port", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Gagal menjalankan server:", err)
	}
}
