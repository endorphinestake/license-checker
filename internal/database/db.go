package database

import (
	"github.com/goldsheva/discord-story-bot/internal/configs"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB
)

func InitDB() *gorm.DB {
	config := configs.GetEnvConfig()

	switch config.DB_DRIVER {
	case "sqlite3":
		DB = InitSQLite()
	case "mysql":
		DB = InitMySQL()
	default:
		DB = InitSQLite()
	}

	logrus.WithFields(logrus.Fields{"gopher": "main", "driver": config.DB_DRIVER}).Info("Database connection established")

	return DB
}

func GetDB() *gorm.DB {
	sqlDB, _ := DB.DB()
	if err := sqlDB.Ping(); err != nil {
		DB = InitDB()
	}
	return DB
}
