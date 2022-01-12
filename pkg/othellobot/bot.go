package othellobot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	token string
	api   *tgbotapi.BotAPI
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
		return
	}
	switch update.Message.Text {
	case NEW_GAME_BUTTON_TEXT:
		bot.startNewGame(update)
	case SCOREBOARD_BUTTON_TEXT:
		bot.showScoreboard(update)
	case PROFILE_BUTTON_TEXT:
		bot.showProfile(update)
	case HELP_BUTTON_TEXT:
		bot.showHelp(update)
	}
}

func (bot *Bot) handleCommand(update tgbotapi.Update) {
	switch command := update.Message.Command(); command {
	case "start":
		msgText := fmt.Sprintf("Hi %s!\n"+
			"I am Othello Bot.\n"+
			"Have fun playing Othello strategic board game,\n"+
			"with your friends or opponents around the world!", update.SentFrom().FirstName)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		msg.ReplyMarkup = buildMainKeyboard()
		bot.api.Send(msg)
	default:
		msgText := fmt.Sprintf("Sorry! %s is not recognized as a command.", command)
		bot.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msgText))
	}
}

func (bot *Bot) startNewGame(update tgbotapi.Update) {
	msgText := "You can play Othello with opponents around the world,\n" +
		"or play with your friends in chats or groups!"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
	msg.ReplyMarkup = buildGameModeKeyboard()
	bot.api.Send(msg)
}

func (bot *Bot) showScoreboard(update tgbotapi.Update) {
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Not implemented yet!"))
}

func (bot *Bot) showProfile(update tgbotapi.Update) {
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Not implemented yet!"))
}

func (bot *Bot) showHelp(update tgbotapi.Update) {
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Not implemented yet!"))
}

func (bot *Bot) handleCallbackQuery(update tgbotapi.Update) {

}
