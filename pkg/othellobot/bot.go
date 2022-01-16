package othellobot

import (
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type Bot struct {
	token                        string
	api                          *tgbotapi.BotAPI
	db                           *database.DBHandler
	inlineMessageIDsToUsers      map[string]*tgbotapi.User
	inlineMessageIDsToUsersMutex sync.Mutex
	usersToCurrentGames          map[tgbotapi.User]*othellogame.Game
	usersToCurrentGamesMutex     sync.Mutex
	waitingPlayer                chan *tgbotapi.User
}

func New(token string, mongodbURI string) *Bot {
	return &Bot{
		db:                      database.New(mongodbURI),
		token:                   token,
		usersToCurrentGames:     make(map[tgbotapi.User]*othellogame.Game),
		inlineMessageIDsToUsers: make(map[string]*tgbotapi.User),
		waitingPlayer:           make(chan *tgbotapi.User, 1),
	}
}

func (bot *Bot) Run() {
	var err error
	bot.api, err = tgbotapi.NewBotAPI(bot.token)
	if err != nil {
		log.Panicln(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.api.GetUpdatesChan(updateConfig)

	log.Println("Bot started.")

	for update := range updates {
		go bot.handleUpdate(update)
	}
}

func (bot *Bot) handleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		bot.handleMessage(update)
	case update.CallbackQuery != nil:
		bot.handleCallbackQuery(update)
	case update.InlineQuery != nil:
		bot.handleInlineQuery(update.InlineQuery)
	case update.ChosenInlineResult != nil:
		bot.handleChosenInlineResult(update.ChosenInlineResult)
	}
}

func (bot *Bot) handleMessage(update tgbotapi.Update) {
	if update.Message.IsCommand() {
		bot.handleCommand(update)
		return
	}
	switch update.Message.Text {
	case NEW_GAME_BUTTON_TEXT:
		bot.askGameMode(update)
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
		msgText := fmt.Sprintf("Hi %s\\!\n"+
			"I am *Othello Bot*\\.\n"+
			"Have fun playing Othello strategic board game,\n"+
			"with your friends or opponents around the world\\!", update.SentFrom().FirstName)
		msg := tgbotapi.NewMessage(update.FromChat().ID, msgText)
		msg.ReplyMarkup = buildMainKeyboard()
		msg.ParseMode = "MarkdownV2"
		bot.api.Send(msg)
	default:
		msgText := fmt.Sprintf("Sorry! %s is not recognized as a command.", command)
		bot.api.Send(tgbotapi.NewMessage(update.FromChat().ID, msgText))
	}
}

func (bot *Bot) askGameMode(update tgbotapi.Update) {
	msgText := "You can play Othello with opponents around the world,\n" +
		"or play with your friends in chats!"
	msg := tgbotapi.NewMessage(update.FromChat().ID, msgText)
	msg.ReplyMarkup = buildGameModeKeyboard()
	bot.api.Send(msg)
}

func (bot *Bot) showScoreboard(update tgbotapi.Update) {
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(update.FromChat().ID, "Not implemented yet!"))
}

func (bot *Bot) showProfile(update tgbotapi.Update) {
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(update.FromChat().ID, "Not implemented yet!"))
}

func (bot *Bot) showHelp(update tgbotapi.Update) {
	bot.api.Send(tgbotapi.NewMessage(update.FromChat().ID, HELP_MSG))
}

func (bot *Bot) handleCallbackQuery(update tgbotapi.Update) {
	query := update.CallbackQuery
	data := query.Data

	if match, _ := regexp.MatchString("^\\d+_\\d+$", data); match {
		bot.placeDisk(update)
		return
	}

	switch data {
	case "playWithRandomOpponent":
		bot.playWithRandomOpponent(update)
	case "join":
		bot.startNewGameWithFriend(update)
	}

	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})
}

func (bot *Bot) placeDisk(update tgbotapi.Update) {
	query := update.CallbackQuery
	user := query.From

	bot.usersToCurrentGamesMutex.Lock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", user)
	}

	var where coord.Coord
	fmt.Sscanf(query.Data, "%d_%d", &where.X, &where.Y)

	err := game.PlaceDisk(where, user)

	bot.usersToCurrentGamesMutex.Unlock()

	if err != nil {
		bot.api.Request(tgbotapi.NewCallback(query.ID, err.Error()))
	} else if game.IsEnded() {
		// TODO
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is over!"))
	} else {
		if query.InlineMessageID != "" {
			bot.api.Send(getEditedMsgOfGameInline(game, query.InlineMessageID))
		} else {
			bot.api.Send(getEditedMsgOfGame(game, update.FromChat().ID, query.Message.MessageID))
		}
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Disk placed!"))
	}
}

func (bot *Bot) startNewGameWithFriend(update tgbotapi.Update) {
	query := update.CallbackQuery

	bot.inlineMessageIDsToUsersMutex.Lock()
	user1, ok := bot.inlineMessageIDsToUsers[query.InlineMessageID]
	if !ok {
		log.Panicf("Invalid state: inlineMessageIDsToUsers does not contain %v\n", query.InlineMessageID)
	}
	bot.inlineMessageIDsToUsersMutex.Unlock()

	user2 := query.From

	game := othellogame.New(user1, user2)

	log.Printf("Started %s\n", game)

	bot.usersToCurrentGamesMutex.Lock()
	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game
	bot.usersToCurrentGamesMutex.Unlock()

	bot.api.Send(getEditedMsgOfGameInline(game, query.InlineMessageID))
}

func (bot *Bot) playWithRandomOpponent(update tgbotapi.Update) {
	if len(bot.waitingPlayer) == 0 {
		bot.waitingPlayer <- update.SentFrom()
		return
	}

	user1 := <-bot.waitingPlayer
	user2 := update.SentFrom()

	game := othellogame.New(user1, user2)

	log.Printf("Started %s\n", game)

	bot.usersToCurrentGamesMutex.Lock()
	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game
	bot.usersToCurrentGamesMutex.Unlock()

	msg := tgbotapi.NewMessage(update.FromChat().ID, getGameMsg(game))
	msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: game.InlineKeyboard(true)} //TODO

	bot.api.Send(msg)
}

func (bot *Bot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	game := tgbotapi.NewInlineQueryResultArticleMarkdownV2(
		uuid.NewString(),
		"Othello",
		fmt.Sprintf("Let's Play Othello\\! [🎯](%s)", BOT_PIC),
	)
	game.Description = HELP_MSG
	game.ReplyMarkup = buildJoinToGameKeyboard()
	game.ThumbURL = BOT_PIC
	game.ThumbWidth = 330
	game.ThumbHeight = 280

	bot.api.Request(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.ID,
		Results:       []interface{}{game},
	})
}

func (bot *Bot) handleChosenInlineResult(chosenInlineResult *tgbotapi.ChosenInlineResult) {
	user := chosenInlineResult.From
	id := chosenInlineResult.InlineMessageID

	bot.inlineMessageIDsToUsersMutex.Lock()
	bot.inlineMessageIDsToUsers[id] = user
	bot.inlineMessageIDsToUsersMutex.Unlock()
}
