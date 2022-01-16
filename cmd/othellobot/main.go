package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/othellobot"
)

func main() {
	token := os.Getenv("OTHELLO_TOKEN")
	mongodbURI := os.Getenv("OTHELLO_MONGODB_URI")
	if token == "" || mongodbURI == "" {
		log.Fatalln("OTHELLO_TOKEN or OTHELLO_MONGODB_URI environment variable is not set.")
	}

	rand.Seed(time.Now().UnixNano())

	bot := othellobot.New(token, mongodbURI)
	bot.Run()
}
