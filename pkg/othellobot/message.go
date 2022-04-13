package othellobot

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"

	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (bot *Bot) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		bot.handleCommand(message)
		return
	}

	if strings.HasPrefix(message.Text, "End chat with") {
		bot.userIDToChatBuddyMutex.Lock()
		delete(bot.userIDToChatBuddy, message.From.ID)
		bot.userIDToChatBuddyMutex.Unlock()

		msg := tgbotapi.NewMessage(message.From.ID, "Chat ended.")
		msg.ReplyMarkup = buildMainKeyboard()
		bot.api.Send(msg)
		return
	}

	switch message.Text {
	case newGameButtonText:
		bot.askGameMode(message)
	case scoreboardButtonText:
		bot.showScoreboard(message)
	case profileButtonText:
		bot.showProfile(message)
	case helpButtonText:
		bot.showHelp(message)
	default:
		user1 := message.From

		bot.userIDToChatBuddyMutex.Lock()
		user2, ok := bot.userIDToChatBuddy[user1.ID]
		bot.userIDToChatBuddyMutex.Unlock()
		if !ok {
			break
		}

		msg := fmt.Sprintf(
			"📬 Message from %s:\n\n%s",
			util.FirstNameElseLastName(user1),
			message.Text,
		)
		bot.api.Send(tgbotapi.NewMessage(user2.ID, msg))
	}
}

func (bot *Bot) handleCommand(message *tgbotapi.Message) {
	switch command := message.Command(); command {
	case "start":
		bot.handleStartCommand(message)
	case "stats":
		msgText := fmt.Sprintf("⚪️ Games played today: %d\n"+
			"⚫️ Users joined today: %d\n"+
			"🔴 All players: %d",
			atomic.LoadUint64(&bot.gamesPlayedToday),
			atomic.LoadUint64(&bot.usersJoinedToday),
			bot.db.UsersCount(),
		)
		bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, msgText))
	default:
		msgText := fmt.Sprintf("Sorry! %s is not recognized as a command.", command)
		bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, msgText))
	}
}

func (bot *Bot) handleStartCommand(message *tgbotapi.Message) {
	user := message.From

	if arg := message.CommandArguments(); strings.HasPrefix(arg, "replay") {
		if err := bot.sendGameReplay(user, arg); err != nil {
			bot.api.Send(tgbotapi.NewMessage(user.ID, err.Error()))
		}
		return
	}

	msgText := fmt.Sprintf("Hi %s\\!\n"+
		"I am *Othello Bot*\\.\n"+
		"Have fun playing Othello strategic board game,\n"+
		"with your friends or opponents around the world\\!", user.FirstName)
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ReplyMarkup = buildMainKeyboard()
	msg.ParseMode = "MarkdownV2"
	bot.api.Send(msg)

	if bot.db.AddPlayer(user.ID, util.FullNameOf(user)) {
		bot.scoreboard.Insert(bot.db.Find(user.ID))
		atomic.AddUint64(&bot.usersJoinedToday, 1)
	}

	log.Printf("Bot started by %v.", user)
}

func (bot *Bot) askGameMode(message *tgbotapi.Message) {
	bot.userIDToCurrentGameMutex.Lock()
	_, ok := bot.userIDToCurrentGame[message.From.ID]
	bot.userIDToCurrentGameMutex.Unlock()
	if ok {
		bot.api.Send(
			tgbotapi.NewMessage(
				message.Chat.ID,
				"You can't play more than one games at the same time.",
			),
		)
		return
	}

	msgText := "You can play Othello with opponents around the world,\n" +
		"or play with your friends in chats!"
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ReplyMarkup = buildGameModeKeyboard()
	_, err := bot.api.Send(msg)
	if err != nil {
		log.Panicln(err)
	}
}

func (bot *Bot) showScoreboard(message *tgbotapi.Message) {
	bot.api.Send(
		tgbotapi.NewMessage(
			message.Chat.ID,
			bot.scoreboard.String(message.From.ID),
		),
	)
}

func (bot *Bot) showProfile(message *tgbotapi.Message) {
	msg := bot.db.Find(message.From.ID).String(bot.scoreboard.RankOf(message.From.ID))
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, msg))
}

func (bot *Bot) showHelp(message *tgbotapi.Message) {
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
}
