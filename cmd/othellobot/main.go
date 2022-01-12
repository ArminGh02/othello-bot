package main

import (
	"log"
	"os"

	"github.com/ArminGh02/othello-bot/pkg/bot"
)

func main() {
	apiToken := os.Getenv("OTHELLO_TOKEN")
	if apiToken == "" {
		log.Fatalln("OTHELLO_TOKEN environment variable is not set.")
	}
	bot.Run(apiToken)
}