package main

import (
	"log"
	"os"

	"github.com/ArminGh02/othello-bot/pkg/othellobot"
)

func main() {
	token := os.Getenv("OTHELLO_TOKEN")
	if token == "" {
		log.Fatalln("OTHELLO_TOKEN environment variable is not set.")
	}
	bot := othellobot.New(token)
	bot.Run()
}
