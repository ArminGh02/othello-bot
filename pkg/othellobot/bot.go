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

		bot.db.AddPlayer(user.ID, getFullNameOf(user))
		bot.scoreboard.Insert(bot.db.Find(user.ID))
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
	// TODO: implement
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, "Not implemented yet!"))
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
}

func (bot *Bot) startNewGameWithFriend(query *tgbotapi.CallbackQuery) {
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
	msg.ReplyMarkup = buildGameKeyboard(game, bot.db.LegalMovesAreShown(game.ActiveUser().ID))

	bot.api.Send(msg)
	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})
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

	bot.api.Send(getEditedMsgOfGame(game, query, user.ID, bot.db.LegalMovesAreShown(user.ID)))
	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})
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
		CacheTime:     0,
	})
}

func (bot *Bot) handleChosenInlineResult(chosenInlineResult *tgbotapi.ChosenInlineResult) {
	user := chosenInlineResult.From
	id := chosenInlineResult.InlineMessageID

	bot.inlineMessageIDsToUsersMutex.Lock()
	bot.inlineMessageIDsToUsers[id] = user
	bot.inlineMessageIDsToUsersMutex.Unlock()
}
