package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"fmt"

	"github.com/goldsheva/discord-story-bot/internal/configs"
	gorm_logrus "github.com/onrik/gorm-logrus"
	"github.com/sirupsen/logrus"
)

func InitMySQL() *gorm.DB {
	config := configs.GetEnvConfig()

	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC&interpolateParams=true",
		config.DB_USER,
		config.DB_PASSWORD,
		"localhost",
		3306,
		config.DB_NAME,
	)), &gorm.Config{
		Logger:                                   gorm_logrus.New(),
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		logrus.Fatal("Can't init MySQL connection: ", err)
	}

	return db
}
