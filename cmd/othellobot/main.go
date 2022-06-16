package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/logging"
	"github.com/ArminGh02/othello-bot/pkg/othellobot"
	"github.com/joho/godotenv"
)

func main() {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		log.Fatalln("Error loading location:", err)
	}

	log.SetFlags(0)
	log.SetOutput(&logging.Writer{Loc: loc})

	err = godotenv.Load()
	if err != nil {
		log.Fatalln("Error loading .env file:", err)
	}

	token := os.Getenv("OTHELLO_TOKEN")
	mongodbURI := os.Getenv("OTHELLO_MONGODB_URI")
	if token == "" || mongodbURI == "" {
		log.Fatalln("OTHELLO_TOKEN or OTHELLO_MONGODB_URI environment variable is not set.")
	}

	rand.Seed(time.Now().UnixNano())

	bot := othellobot.New(token, mongodbURI)
	bot.Run()
}
