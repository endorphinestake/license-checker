package main

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/goldsheva/discord-story-bot/internal/configs"
	"github.com/goldsheva/discord-story-bot/internal/workers"
	"github.com/joho/godotenv"

	"github.com/sirupsen/logrus"
)

//go:embed config.example.env
var embeddedExampleEnv []byte

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Load environment variables from config.env file
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Fatal("Cannot determine working directory: ", err)
	}
	envPath := filepath.Join(cwd, "config.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		logrus.Info("Loading embedded config.example.env")
		parsedEnv, err := godotenv.Parse(bytes.NewReader(embeddedExampleEnv))
		if err != nil {
			logrus.Fatal("Failed to parse embedded config.example.env: ", err)
		}
		for k, v := range parsedEnv {
			os.Setenv(k, v)
		}
	} else {
		logrus.Info("Loading config.env ...")

		if err := godotenv.Overload(envPath); err != nil {
			logrus.Fatal("Failed to load config.env: ", err)
		}
	}

	config := configs.GetEnvConfig()

	logrus.SetLevel(config.LogLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	wg.Add(1)
	go workers.GoDiscordBot(ctx, wg)

	// Handle sigterm and await termChan signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	<-termChan
	logrus.WithFields(logrus.Fields{"gopher": "main"}).Warn("Initiating shutdown...")
	cancelFunc()

	wg.Wait()
	logrus.WithFields(logrus.Fields{"gopher": "main"}).Warn("Shutdown complete. All processes stopped!")
}
