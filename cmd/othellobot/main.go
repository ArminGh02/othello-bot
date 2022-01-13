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
	if token == "" {
		log.Fatalln("OTHELLO_TOKEN environment variable is not set.")
	}

	rand.Seed(time.Now().UnixNano())

	bot := othellobot.New(token)
	bot.Run()
}
