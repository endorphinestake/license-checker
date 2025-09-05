package database

import (
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/goldsheva/discord-story-bot/internal/configs"
	gorm_logrus "github.com/onrik/gorm-logrus"
	"github.com/sirupsen/logrus"
)

func InitSQLite() *gorm.DB {
	config := configs.GetEnvConfig()

	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	dir := filepath.Dir(exePath)
	dbPath := filepath.Join(dir, config.DB_NAME)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger:                                   gorm_logrus.New(),
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		logrus.Fatal("Can't init SQLite connection: ", err)
	}

	return db
}
