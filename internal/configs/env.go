package configs

import (
	"os"
	"sync"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/sirupsen/logrus"
)

var config *Config
var once sync.Once

type Config struct {
	LogLevel                 logrus.Level
	LOCALE                   string
	DB_DRIVER                string
	DB_NAME                  string
	DB_USER                  string
	DB_PASSWORD              string
	BOT_TOKEN                string
	STORY_API_KEY            string
	STORY_API_BASE_URL       string
	STORY_BUTTON_TIMEOUT_SEC int
}

func GetEnvConfig() *Config {
	once.Do(func() {
		config = &Config{
			STORY_BUTTON_TIMEOUT_SEC: 300,
		}

		switch os.Getenv("LOG_LEVEL") {
		case "debug":
			config.LogLevel = logrus.DebugLevel
		case "info":
			config.LogLevel = logrus.InfoLevel
		case "warn":
			config.LogLevel = logrus.WarnLevel
		case "error":
			config.LogLevel = logrus.ErrorLevel
		case "fatal":
			config.LogLevel = logrus.FatalLevel
		case "panic":
			config.LogLevel = logrus.PanicLevel
		default:
			config.LogLevel = logrus.InfoLevel
		}

		config.LOCALE = os.Getenv("LOCALE")
		config.DB_DRIVER = os.Getenv("DB_DRIVER")
		config.DB_NAME = os.Getenv("DB_NAME")
		config.DB_USER = os.Getenv("DB_USER")
		config.DB_PASSWORD = os.Getenv("DB_PASSWORD")
		config.BOT_TOKEN = os.Getenv("BOT_TOKEN")
		config.STORY_API_KEY = os.Getenv("STORY_API_KEY")
		config.STORY_API_BASE_URL = os.Getenv("STORY_API_BASE_URL")

		if err := validation.ValidateStruct(config,
			validation.Field(&config.LOCALE, validation.Required, validation.In("en", "ru")),
			validation.Field(&config.DB_DRIVER, validation.Required, validation.In("sqlite3", "mysql")),
			validation.Field(&config.DB_NAME, validation.Required, validation.Length(4, 64)),
			validation.Field(&config.DB_USER, validation.Length(0, 64)),
			validation.Field(&config.DB_PASSWORD, validation.Length(0, 64)),
			validation.Field(&config.BOT_TOKEN, validation.Required, validation.Length(1, 256)),
			validation.Field(&config.STORY_API_KEY, validation.Required, validation.Length(1, 256)),
			validation.Field(&config.STORY_API_BASE_URL, validation.Required, validation.Length(1, 256)),
		); err != nil {
			logrus.Fatalf("Can't parse .env: %v", err)
		}
	})

	return config
}
