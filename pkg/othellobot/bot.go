package othellobot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	token     string
	api       *tgbotapi.BotAPI
	isStarted bool
}

func New(token string) *Bot {
	return &Bot{
		token: token,
	}
}

func (bot *Bot) Run() {
	botapi, err := tgbotapi.NewBotAPI(bot.token)
	if err != nil {
		log.Panic(err)
	}

	bot.api = botapi

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := botapi.GetUpdatesChan(updateConfig)
	for update := range updates {
		go bot.handleUpdate(update)
	}
}

func (bot *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		bot.handleMessage(update)
	} else if update.CallbackQuery != nil {
		bot.handleCallbackQuery(update)
	}
}

func (bot *Bot) handleMessage(update tgbotapi.Update) {
	if update.Message.IsCommand() {
		bot.handleCommand(update)
	}
}

func (bot *Bot) handleCommand(update tgbotapi.Update) {
	switch update.Message.Command() {
	case "start":
		msg := fmt.Sprintf("Hi %s!\n"+
			"I am Othello Bot.\n"+
			"Have fun playing Othello strategic board game,\n"+
			"with your friends or opponents around the world!", update.SentFrom().FirstName)
		bot.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
	}
}

func (bot *Bot) handleCallbackQuery(update tgbotapi.Update) {

}
