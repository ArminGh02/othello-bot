package othellobot

import (
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type Bot struct {
	token                        string
	api                          *tgbotapi.BotAPI
	db                           *database.DBHandler
	scoreboard                   util.Scoreboard
	inlineMessageIDsToUsers      map[string]*tgbotapi.User
	inlineMessageIDsToUsersMutex sync.Mutex
	gamesToInlineMessageIDs      map[*othellogame.Game]string
	gamesToInlineMessageIDsMutex sync.Mutex
	usersToCurrentGames          map[tgbotapi.User]*othellogame.Game
	usersToCurrentGamesMutex     sync.Mutex
	waitingPlayer                chan *tgbotapi.User
}

func New(token string, mongodbURI string) *Bot {
	db := database.New(mongodbURI)
	return &Bot{
		token:                   token,
		db:                      db,
		scoreboard:              util.NewScoreboard(db.GetAllPlayers()),
		inlineMessageIDsToUsers: make(map[string]*tgbotapi.User),
		gamesToInlineMessageIDs: make(map[*othellogame.Game]string),
		usersToCurrentGames:     make(map[tgbotapi.User]*othellogame.Game),
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

	defer bot.db.Disconnect()

	log.Println("Bot started.")

	for update := range updates {
		go bot.handleUpdate(update)
	}
}

func (bot *Bot) handleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		bot.handleMessage(update.Message)
	case update.CallbackQuery != nil:
		bot.handleCallbackQuery(update.CallbackQuery)
	case update.InlineQuery != nil:
		bot.handleInlineQuery(update.InlineQuery)
	case update.ChosenInlineResult != nil:
		bot.handleChosenInlineResult(update.ChosenInlineResult)
	}
}

func (bot *Bot) handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		bot.handleCommand(message)
		return
	}
	switch message.Text {
	case NEW_GAME_BUTTON_TEXT:
		bot.askGameMode(message)
	case SCOREBOARD_BUTTON_TEXT:
		bot.showScoreboard(message)
	case PROFILE_BUTTON_TEXT:
		bot.showProfile(message)
	case HELP_BUTTON_TEXT:
		bot.showHelp(message)
	}
}

func (bot *Bot) handleCommand(message *tgbotapi.Message) {
	switch command := message.Command(); command {
	case "start":
		user := message.From

		msgText := fmt.Sprintf("Hi %s\\!\n"+
			"I am *Othello Bot*\\.\n"+
			"Have fun playing Othello strategic board game,\n"+
			"with your friends or opponents around the world\\!", user.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
		msg.ReplyMarkup = buildMainKeyboard()
		msg.ParseMode = "MarkdownV2"
		bot.api.Send(msg)

		if bot.db.AddPlayer(user.ID, getFullNameOf(user)) {
			bot.scoreboard.Insert(bot.db.Find(user.ID))
		}
	default:
		msgText := fmt.Sprintf("Sorry! %s is not recognized as a command.", command)
		bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, msgText))
	}
}

func (bot *Bot) askGameMode(message *tgbotapi.Message) {
	msgText := "You can play Othello with opponents around the world,\n" +
		"or play with your friends in chats!"
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ReplyMarkup = buildGameModeKeyboard()
	bot.api.Send(msg)
}

func (bot *Bot) showScoreboard(message *tgbotapi.Message) {
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, bot.scoreboard.String()))
}

func (bot *Bot) showProfile(message *tgbotapi.Message) {
	msg := bot.db.Find(message.From.ID).String(bot.scoreboard.RankOf(message.From.ID))
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, msg))
}

func (bot *Bot) showHelp(message *tgbotapi.Message) {
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, HELP_MSG))
}

func (bot *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	data := query.Data

	if match, _ := regexp.MatchString("^\\d+_\\d+$", data); match {
		bot.placeDisk(query)
		return
	}

	switch data {
	case "join":
		bot.startNewGameWithFriend(query)
	case "playWithRandomOpponent":
		bot.playWithRandomOpponent(query)
	case "toggleShowingLegalMoves":
		bot.toggleShowingLegalMoves(query)
	case "whiteProfile":
		bot.alertProfile(true, query)
	case "blackProfile":
		bot.alertProfile(false, query)
	case "surrender":
		bot.handleSurrender(query)
	}
}

func (bot *Bot) placeDisk(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", user)
	}

	var where coord.Coord
	fmt.Sscanf(query.Data, "%d_%d", &where.X, &where.Y)

	err := game.PlaceDisk(where, user)
	if err != nil {
		bot.api.Request(tgbotapi.NewCallback(query.ID, err.Error()))
	} else if game.IsEnded() {
		bot.handleGameEnd(game, query)
	} else {
		bot.api.Send(getEditedMsgOfGame(game, query, user.ID, bot.db.LegalMovesAreShown(user.ID)))
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Disk placed!"))
	}
}

func (bot *Bot) handleGameEnd(game *othellogame.Game, query *tgbotapi.CallbackQuery) {
	winner, loser := game.Winner(), game.Loser()
	if winner == nil {
		bot.db.IncrementDraws(game.WhiteUser().ID)
		bot.db.IncrementDraws(game.BlackUser().ID)
	} else {
		bot.db.IncrementWins(winner.ID)
		bot.db.IncrementLosses(loser.ID)
		bot.scoreboard.UpdateRankOf(winner.ID, 1, 0)
		bot.scoreboard.UpdateRankOf(loser.ID, 0, 1)
	}

	bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is over!"))

	bot.cleanUp(game, query)
}

func (bot *Bot) cleanUp(game *othellogame.Game, query *tgbotapi.CallbackQuery) {
	delete(bot.usersToCurrentGames, *game.WhiteUser())
	delete(bot.usersToCurrentGames, *game.BlackUser())

	if query.InlineMessageID != "" {
		bot.inlineMessageIDsToUsersMutex.Lock()
		delete(bot.inlineMessageIDsToUsers, query.InlineMessageID)
		bot.inlineMessageIDsToUsersMutex.Unlock()

		bot.gamesToInlineMessageIDsMutex.Lock()
		delete(bot.gamesToInlineMessageIDs, game)
		bot.gamesToInlineMessageIDsMutex.Unlock()
	}
}

func (bot *Bot) startNewGameWithFriend(query *tgbotapi.CallbackQuery) {
	bot.inlineMessageIDsToUsersMutex.Lock()
	user1, ok := bot.inlineMessageIDsToUsers[query.InlineMessageID]
	if !ok {
		log.Panicf("Invalid state: inlineMessageIDsToUsers does not contain %v\n", query.InlineMessageID)
	}
	bot.inlineMessageIDsToUsersMutex.Unlock()

	user2 := query.From

	if bot.db.AddPlayer(user2.ID, getFullNameOf(user2)) {
		bot.scoreboard.Insert(bot.db.Find(user2.ID))
	}

	game := othellogame.New(user1, user2)

	log.Printf("Started %s\n", game)

	bot.gamesToInlineMessageIDsMutex.Lock()
	bot.gamesToInlineMessageIDs[game] = query.InlineMessageID
	bot.gamesToInlineMessageIDsMutex.Unlock()

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game

	bot.api.Send(getEditedMsgOfGameInline(
		game,
		query.InlineMessageID,
		bot.db.LegalMovesAreShown(game.ActiveUser().ID),
	))
	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})
}

func (bot *Bot) playWithRandomOpponent(query *tgbotapi.CallbackQuery) {
	defer bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})

	if len(bot.waitingPlayer) == 0 {
		bot.waitingPlayer <- query.From
		return
	}

	user1 := <-bot.waitingPlayer
	user2 := query.From

	game := othellogame.New(user1, user2)

	log.Printf("Started %s\n", game)

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game

	msg := tgbotapi.NewMessage(query.Message.Chat.ID, getGameMsg(game))
	msg.ReplyMarkup = buildGameKeyboard(game, bot.db.LegalMovesAreShown(game.ActiveUser().ID), false)

	bot.api.Send(msg)
}

func (bot *Bot) toggleShowingLegalMoves(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", user)
	}

	bot.db.ToggleLegalMovesAreShown(user.ID)

	if *user == *game.ActiveUser() {
		bot.api.Send(getEditedMsgOfGame(game, query, user.ID, bot.db.LegalMovesAreShown(user.ID)))
	}

	bot.api.Request(tgbotapi.NewCallback(query.ID, "Toggled for you!"))
}

func (bot *Bot) alertProfile(white bool, query *tgbotapi.CallbackQuery) {
	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game := bot.usersToCurrentGames[*query.From]

	var userID int64
	if white {
		userID = game.WhiteUser().ID
	} else {
		userID = game.BlackUser().ID
	}

	rank := bot.scoreboard.RankOf(userID)

	bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, bot.db.Find(userID).String(rank)))
}

func (bot *Bot) handleSurrender(query *tgbotapi.CallbackQuery) {
	loser := query.From

	bot.usersToCurrentGamesMutex.Lock()

	game, ok := bot.usersToCurrentGames[*loser]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", loser)
	}

	winner := game.OpponentOf(loser)

	msgText := fmt.Sprintf("%s surrendered to %s!\n", loser.FirstName, winner.FirstName)
	var msg tgbotapi.Chattable
	if query.InlineMessageID != "" {
		msg = tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				InlineMessageID: query.InlineMessageID,
			},
			Text: msgText,
		}
	} else {
		msg = tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				ChatID:    query.Message.Chat.ID,
				MessageID: query.Message.MessageID,
			},
			Text: msgText,
		}
	}
	bot.api.Send(msg)

	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})

	bot.cleanUp(game, query)

	bot.usersToCurrentGamesMutex.Unlock()

	bot.db.IncrementWins(winner.ID)
	bot.db.IncrementLosses(loser.ID)
	bot.scoreboard.UpdateRankOf(winner.ID, 1, 0)
	bot.scoreboard.UpdateRankOf(loser.ID, 0, 1)
}

func (bot *Bot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	if inlineQuery.Query == "#Resend" {
		bot.resendGame(inlineQuery)
	}

	user := inlineQuery.From

	if bot.db.AddPlayer(user.ID, getFullNameOf(user)) {
		bot.scoreboard.Insert(bot.db.Find(user.ID))
	}

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
		CacheTime:     0,
	})
}

func (bot *Bot) resendGame(inlineQuery *tgbotapi.InlineQuery) {
	user := inlineQuery.From

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", user)
	}

	msg := tgbotapi.NewInlineQueryResultArticle(
		uuid.NewString(),
		"Send down your current game",
		getGameMsg(game),
	)
	msg.ReplyMarkup = buildGameKeyboard(game, bot.db.LegalMovesAreShown(game.ActiveUser().ID), true)

	bot.api.Request(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.ID,
		Results:       []interface{}{msg},
		CacheTime:     0,
	})
}

func (bot *Bot) handleChosenInlineResult(chosenInlineResult *tgbotapi.ChosenInlineResult) {
	user := chosenInlineResult.From
	newID := chosenInlineResult.InlineMessageID

	if chosenInlineResult.Query != "#Resend" {
		bot.inlineMessageIDsToUsersMutex.Lock()
		bot.inlineMessageIDsToUsers[newID] = user
		bot.inlineMessageIDsToUsersMutex.Unlock()
		return
	}

	bot.usersToCurrentGamesMutex.Lock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v\n", user)
	}

	bot.gamesToInlineMessageIDsMutex.Lock()
	oldID, ok := bot.gamesToInlineMessageIDs[game]
	if !ok {
		log.Panicf("Invalid state: gamesToInlineMessageIDs does not contain %v\n", game)
	}
	bot.gamesToInlineMessageIDs[game] = newID
	bot.gamesToInlineMessageIDsMutex.Unlock()

	bot.api.Send(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: oldID,
		},
		Text: fmt.Sprintf("%v has been moved down 🔽", game),
	})

	bot.usersToCurrentGamesMutex.Unlock()

	bot.inlineMessageIDsToUsersMutex.Lock()
	bot.inlineMessageIDsToUsers[newID] = user
	delete(bot.inlineMessageIDsToUsers, oldID)
	bot.inlineMessageIDsToUsersMutex.Unlock()
}
