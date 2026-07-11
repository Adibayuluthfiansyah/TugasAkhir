package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	_ = godotenv.Load()

	rawURL := os.Getenv("MYSQL_URL")
	if rawURL == "" {
		log.Fatal("❌ MYSQL_URL tidak ditemukan")
	}

	// mysql://user:pass@host:port/db
	dsn := strings.Replace(rawURL, "mysql://", "", 1)
	dsn = strings.Replace(dsn, "@", "@tcp(", 1)
	dsn = strings.Replace(dsn, "/", ")/", 1)

	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Gagal koneksi database:", err)
	}

	DB = database

	log.Println("✅ Database MySQL terkoneksi")
}

func RegisterQueryProtector() {
	registerQueryProtector(DB)
	log.Println("✅ SQL Query Protector aktif")
}
