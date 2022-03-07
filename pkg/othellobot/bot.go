package othellobot

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/gifmaker"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	cron "github.com/robfig/cron/v3"
)

type Bot struct {
	token                        string
	api                          *tgbotapi.BotAPI
	db                           *database.Handler
	scoreboard                   util.Scoreboard
	waitingPlayer                chan *tgbotapi.User
	inlineMessageIDToUser        map[string]*tgbotapi.User
	gameIDToMovesSequence        map[string][]coord.Coord
	gameIDToInlineMessageID      map[string]string
	userIDToCurrentGame          map[int64]*othellogame.Game
	userIDToLastTimeActive       map[int64]time.Time
	userIDToMessageID            map[int64]int
	userIDToChatBuddy            map[int64]*tgbotapi.User
	userIDToUser                 map[int64]*tgbotapi.User
	userIDToRematchGameID        map[int64]string
	inlineMessageIDToUserMutex   sync.Mutex
	gameIDToMovesSequenceMutex   sync.Mutex
	gameIDToInlineMessageIDMutex sync.Mutex
	userIDToCurrentGameMutex     sync.Mutex
	userIDToLastTimeActiveMutex  sync.Mutex
	userIDToMessageIDMutex       sync.Mutex
	userIDToChatBuddyMutex       sync.Mutex
	userIDToUserMutex            sync.Mutex
	userIDToRematchGameIDMutex   sync.Mutex

	gamesPlayedToday uint64
	usersJoinedToday uint64
}

func New(token, mongodbURI string) *Bot {
	db := database.New(mongodbURI)
	return &Bot{
		token:                   token,
		db:                      db,
		scoreboard:              util.NewScoreboard(db.GetAllPlayers()),
		waitingPlayer:           make(chan *tgbotapi.User, 1),
		inlineMessageIDToUser:   make(map[string]*tgbotapi.User),
		gameIDToMovesSequence:   make(map[string][]coord.Coord),
		gameIDToInlineMessageID: make(map[string]string),
		userIDToCurrentGame:     make(map[int64]*othellogame.Game),
		userIDToLastTimeActive:  make(map[int64]time.Time),
		userIDToMessageID:       make(map[int64]int),
		userIDToChatBuddy:       make(map[int64]*tgbotapi.User),
		userIDToUser:            make(map[int64]*tgbotapi.User),
		userIDToRematchGameID:   make(map[int64]string),
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

	c := cron.New()
	c.AddFunc("@daily", func() {
		atomic.SwapUint64(&bot.gamesPlayedToday, 0)
		atomic.SwapUint64(&bot.usersJoinedToday, 0)
	})
	c.Start()

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
	bot.api.Send(msg)
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

func (bot *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	switch data := query.Data; data {
	case "join":
		bot.startGameOfFriends(query)
	case "playWithRandomOpponent":
		bot.playWithRandomOpponent(query)
	case "cancel":
		bot.handleCanceledGame(query)
	case "toggleShowingLegalMoves":
		bot.toggleShowingLegalMoves(query)
	case "surrender":
		bot.handleSurrender(query)
	case "end":
		bot.handleEndEarly(query)
	case "chat":
		bot.startChatBetweenOpponents(query)
	case "gameOver":
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is over!"))
	default:
		match, _ := regexp.MatchString(`^\d+_\d+$`, data)
		switch {
		case match:
			bot.placeDisk(query)
		case strings.HasPrefix(data, "replay"):
			text := ""
			if err := bot.sendGameReplay(query.From, query.Data); err != nil {
				text = err.Error()
			}
			bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		case strings.HasPrefix(data, "profile"):
			bot.alertProfile(query)
		case strings.HasPrefix(data, "rematch"):
			bot.handleRematch(query)
		case strings.HasPrefix(data, "accept"):
			bot.handleAcceptedRematch(query)
		case strings.HasPrefix(data, "reject"):
			bot.handleRejectedRematch(query)
		}
	}
}

func (bot *Bot) sendGameReplay(user *tgbotapi.User, data string) error {
	data = strings.TrimPrefix(data, "replay")
	gameID := data[1:]

	bot.gameIDToMovesSequenceMutex.Lock()
	movesSequence, ok := bot.gameIDToMovesSequence[gameID]
	bot.gameIDToMovesSequenceMutex.Unlock()
	if !ok {
		return fmt.Errorf("Game is too old!")
	}

	whiteStarted := data[0] == 'w'

	gifFilename := gameID + ".gif"
	gifmaker.Make(gifFilename, movesSequence, whiteStarted)
	bot.api.Send(tgbotapi.NewAnimation(user.ID, tgbotapi.FilePath(gifFilename)))

	err := os.Remove(gifFilename)
	if err != nil {
		log.Panicln(err)
	}

	return nil
}

func (bot *Bot) placeDisk(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Game is too old!"))
		return
	}

	var where coord.Coord
	fmt.Sscanf(query.Data, "%d_%d", &where.X, &where.Y)

	err := game.PlaceDisk(where, user)
	if err != nil {
		bot.api.Request(tgbotapi.NewCallback(query.ID, err.Error()))
	} else if game.IsEnded() {
		bot.handleGameEnd(game, query)
	} else {
		bot.userIDToLastTimeActiveMutex.Lock()
		bot.userIDToLastTimeActive[user.ID] = time.Now()
		bot.userIDToLastTimeActiveMutex.Unlock()

		msg, replyMarkup := getRunningGameMsgAndReplyMarkup(
			game,
			bot.db.LegalMovesAreShown(game.ActiveUser().ID),
			query.InlineMessageID != "",
		)
		bot.sendEditMessageTextForGame(
			msg,
			replyMarkup,
			game.WhiteUser(),
			game.BlackUser(),
			query.InlineMessageID,
		)
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

	bot.gameIDToMovesSequenceMutex.Lock()
	bot.gameIDToMovesSequence[game.ID()] = game.MovesSequence()
	bot.gameIDToMovesSequenceMutex.Unlock()

	msg, replyMarkup := getGameOverMsgAndReplyMarkup(
		game,
		bot.api.Self.UserName,
		query.InlineMessageID != "",
	)
	bot.sendEditMessageTextForGame(
		msg,
		replyMarkup,
		game.WhiteUser(),
		game.BlackUser(),
		query.InlineMessageID,
	)

	bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is over!"))

	bot.cleanUp(game, query)
	log.Println(game, "is over.")
	atomic.AddUint64(&bot.gamesPlayedToday, 1)
}

func (bot *Bot) cleanUp(game *othellogame.Game, query *tgbotapi.CallbackQuery) {
	user1 := game.WhiteUser()
	user2 := game.BlackUser()

	delete(bot.userIDToCurrentGame, user1.ID)
	delete(bot.userIDToCurrentGame, user2.ID)

	if query.InlineMessageID != "" {
		bot.inlineMessageIDToUserMutex.Lock()
		delete(bot.inlineMessageIDToUser, query.InlineMessageID)
		bot.inlineMessageIDToUserMutex.Unlock()

		bot.gameIDToInlineMessageIDMutex.Lock()
		delete(bot.gameIDToInlineMessageID, game.ID())
		bot.gameIDToInlineMessageIDMutex.Unlock()
	} else {
		bot.userIDToMessageIDMutex.Lock()
		delete(bot.userIDToMessageID, user1.ID)
		delete(bot.userIDToMessageID, user2.ID)
		bot.userIDToMessageIDMutex.Unlock()
	}
}

func (bot *Bot) startGameOfFriends(query *tgbotapi.CallbackQuery) {
	bot.inlineMessageIDToUserMutex.Lock()
	user1, ok := bot.inlineMessageIDToUser[query.InlineMessageID]
	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Invitation is too old!"))
		return
	}
	bot.inlineMessageIDToUserMutex.Unlock()

	user2 := query.From

	if *user1 == *user2 {
		text := "You can't play with yourself!"
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		return
	}

	if _, ok := bot.userIDToCurrentGame[user1.ID]; ok {
		text := util.FirstNameElseLastName(user1) + " is playing another game"
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		return
	}
	if _, ok := bot.userIDToCurrentGame[user2.ID]; ok {
		text := util.FirstNameElseLastName(user2) + " is playing another game"
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		return
	}

	if bot.db.AddPlayer(user2.ID, util.FullNameOf(user2)) {
		bot.scoreboard.Insert(bot.db.Find(user2.ID))
		atomic.AddUint64(&bot.usersJoinedToday, 1)
	}

	game := othellogame.New(user1, user2)

	log.Printf("Started %v.\n", game)

	bot.gameIDToInlineMessageIDMutex.Lock()
	bot.gameIDToInlineMessageID[game.ID()] = query.InlineMessageID
	bot.gameIDToInlineMessageIDMutex.Unlock()

	now := time.Now()
	bot.userIDToLastTimeActiveMutex.Lock()
	bot.userIDToLastTimeActive[user1.ID] = now
	bot.userIDToLastTimeActive[user2.ID] = now
	bot.userIDToLastTimeActiveMutex.Unlock()

	bot.userIDToUserMutex.Lock()
	bot.userIDToUser[user1.ID] = user1
	bot.userIDToUser[user2.ID] = user2
	bot.userIDToUserMutex.Unlock()

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	bot.userIDToCurrentGame[user1.ID] = game
	bot.userIDToCurrentGame[user2.ID] = game

	msg, replyMarkup := getRunningGameMsgAndReplyMarkup(
		game,
		bot.db.LegalMovesAreShown(game.ActiveUser().ID),
		query.InlineMessageID != "",
	)
	bot.sendEditMessageTextForGame(msg, replyMarkup, user1, user2, query.InlineMessageID)

	bot.api.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	})
}

func (bot *Bot) playWithRandomOpponent(query *tgbotapi.CallbackQuery) {
	user1 := query.From

	if len(bot.waitingPlayer) == 0 {
		bot.waitingPlayer <- user1

		msg := tgbotapi.NewMessage(user1.ID, "Wait until another player joins the game.")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Cancel", "cancel"),
			),
		)
		bot.api.Send(msg)

		bot.api.Request(tgbotapi.CallbackConfig{
			CallbackQueryID: query.ID,
		})
		return
	}

	user2 := <-bot.waitingPlayer

	if *user1 == *user2 {
		text := "You can't play with yourself!"
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		bot.waitingPlayer <- user2
		return
	}

	text := ""
	if err := bot.startGameOfRandomOpponents(user1, user2); err != nil {
		text = err.Error()
	}
	bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
}

func (bot *Bot) startGameOfRandomOpponents(user1, user2 *tgbotapi.User) error {
	if _, ok := bot.userIDToCurrentGame[user1.ID]; ok {
		return fmt.Errorf("%s is playing another game", util.FirstNameElseLastName(user1))
	}
	if _, ok := bot.userIDToCurrentGame[user2.ID]; ok {
		return fmt.Errorf("%s is playing another game", util.FirstNameElseLastName(user2))
	}

	game := othellogame.New(user1, user2)

	log.Printf("Started %s.\n", game)

	now := time.Now()
	bot.userIDToLastTimeActiveMutex.Lock()
	bot.userIDToLastTimeActive[user1.ID] = now
	bot.userIDToLastTimeActive[user2.ID] = now
	bot.userIDToLastTimeActiveMutex.Unlock()

	bot.userIDToUserMutex.Lock()
	bot.userIDToUser[user1.ID] = user1
	bot.userIDToUser[user2.ID] = user2
	bot.userIDToUserMutex.Unlock()

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	bot.userIDToCurrentGame[user1.ID] = game
	bot.userIDToCurrentGame[user2.ID] = game

	msgText, replyMarkup := getRunningGameMsgAndReplyMarkup(
		game, bot.db.LegalMovesAreShown(game.ActiveUser().ID), false)
	msg1 := tgbotapi.NewMessage(user1.ID, msgText)
	msg2 := tgbotapi.NewMessage(user2.ID, msgText)
	msg1.ReplyMarkup = replyMarkup
	msg2.ReplyMarkup = replyMarkup

	msg, _ := bot.api.Send(msg1)
	bot.userIDToMessageIDMutex.Lock()
	bot.userIDToMessageID[user1.ID] = msg.MessageID
	bot.userIDToMessageIDMutex.Unlock()

	msg, _ = bot.api.Send(msg2)
	bot.userIDToMessageIDMutex.Lock()
	bot.userIDToMessageID[user2.ID] = msg.MessageID
	bot.userIDToMessageIDMutex.Unlock()

	return nil
}

func (bot *Bot) handleCanceledGame(query *tgbotapi.CallbackQuery) {
	defer bot.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: query.ID})

	if len(bot.waitingPlayer) == 0 {
		return
	}

	waitingPlayer := <-bot.waitingPlayer
	if *waitingPlayer == *query.From {
		bot.api.Send(
			tgbotapi.NewEditMessageTextAndMarkup(
				query.From.ID,
				query.Message.MessageID,
				"Request was canceled.",
				util.RemoveInlineKeyboardMarkup(),
			),
		)
	} else {
		bot.waitingPlayer <- waitingPlayer
	}
}

func (bot *Bot) toggleShowingLegalMoves(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Game is too old!"))
		return
	}

	bot.db.ToggleLegalMovesAreShown(user.ID)

	if game.IsTurnOf(user) {
		msg, replyMarkup := getRunningGameMsgAndReplyMarkup(
			game,
			bot.db.LegalMovesAreShown(user.ID),
			query.InlineMessageID != "",
		)
		bot.sendEditMessageTextForGame(
			msg,
			replyMarkup,
			game.WhiteUser(),
			game.BlackUser(),
			query.InlineMessageID,
		)
	}

	bot.api.Request(tgbotapi.NewCallback(query.ID, "Toggled for you!"))
}

func (bot *Bot) alertProfile(query *tgbotapi.CallbackQuery) {
	userID, _ := strconv.ParseInt(strings.TrimPrefix(query.Data, "profile"), 10, 64)
	rank := bot.scoreboard.RankOf(userID)
	bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, bot.db.Find(userID).String(rank)))
}

func (bot *Bot) handleSurrender(query *tgbotapi.CallbackQuery) {
	loser := query.From

	bot.userIDToCurrentGameMutex.Lock()

	game, ok := bot.userIDToCurrentGame[loser.ID]
	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Game is too old!"))
		return
	}

	bot.gameIDToMovesSequenceMutex.Lock()
	bot.gameIDToMovesSequence[game.ID()] = game.MovesSequence()
	bot.gameIDToMovesSequenceMutex.Unlock()

	winner := game.OpponentOf(loser)

	msg, replyMarkup := getSurrenderMsgAndReplyMarkup(
		game,
		winner,
		loser,
		bot.api.Self.UserName,
		query.InlineMessageID != "",
	)
	bot.sendEditMessageTextForGame(msg, replyMarkup, winner, loser, query.InlineMessageID)

	bot.api.Request(tgbotapi.NewCallback(query.ID, "You surrendered!"))

	bot.cleanUp(game, query)

	bot.userIDToCurrentGameMutex.Unlock()

	bot.db.IncrementWins(winner.ID)
	bot.db.IncrementLosses(loser.ID)
	bot.scoreboard.UpdateRankOf(winner.ID, 1, 0)
	bot.scoreboard.UpdateRankOf(loser.ID, 0, 1)

	log.Printf("%s surrendered in %v.\n", loser, game)
	atomic.AddUint64(&bot.gamesPlayedToday, 1)
}

func (bot *Bot) handleEndEarly(query *tgbotapi.CallbackQuery) {
	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	user1 := query.From

	game, ok := bot.userIDToCurrentGame[user1.ID]
	if !ok {
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is too old!"))
		return
	}

	if game.IsTurnOf(user1) {
		bot.api.Request(
			tgbotapi.NewCallback(query.ID, "You can't end the game in your turn."),
		)
		return
	}

	user2 := game.OpponentOf(user1)

	bot.userIDToLastTimeActiveMutex.Lock()
	lastActiveTime := bot.userIDToLastTimeActive[user2.ID]
	bot.userIDToLastTimeActiveMutex.Unlock()

	secondsSinceLastActive := time.Since(lastActiveTime).Seconds()
	if secondsSinceLastActive > 90 {
		bot.gameIDToMovesSequenceMutex.Lock()
		bot.gameIDToMovesSequence[game.ID()] = game.MovesSequence()
		bot.gameIDToMovesSequenceMutex.Unlock()

		msg, replyMarkup := getEarlyEndMsgAndReplyMarkup(
			game,
			user2,
			bot.api.Self.UserName,
			query.InlineMessageID != "",
		)
		bot.sendEditMessageTextForGame(
			msg,
			replyMarkup,
			user1,
			user2,
			query.InlineMessageID,
		)

		bot.cleanUp(game, query)

		bot.db.IncrementWins(user1.ID)
		bot.db.IncrementLosses(user2.ID)
		bot.scoreboard.UpdateRankOf(user1.ID, 1, 0)
		bot.scoreboard.UpdateRankOf(user2.ID, 0, 1)

		bot.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: query.ID})
		atomic.AddUint64(&bot.gamesPlayedToday, 1)
		log.Printf("%s ended %v.\n", user1, game)
	} else {
		msg := fmt.Sprintf("You can end the game if your "+
			"opponent doesn't place a disk for %d seconds.",
			90-int(secondsSinceLastActive),
		)
		bot.api.Request(tgbotapi.NewCallback(query.ID, msg))
	}
}

func (bot *Bot) startChatBetweenOpponents(query *tgbotapi.CallbackQuery) {
	user1 := query.From
	user2, err := bot.opponentOf(user1)
	if err != nil {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, err.Error()))
		return
	}

	msg := tgbotapi.NewMessage(user1.ID, "Chat with your opponent:")
	buttonText := fmt.Sprint("End chat with ", util.FirstNameElseLastName(user2))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		),
	)
	bot.api.Send(msg)

	bot.userIDToChatBuddyMutex.Lock()
	bot.userIDToChatBuddy[user1.ID] = user2
	bot.userIDToChatBuddyMutex.Unlock()
}

func (bot *Bot) handleRematch(query *tgbotapi.CallbackQuery) {
	var user1ID, user2ID int64
	var gameID string
	fmt.Sscanf(query.Data, "rematch%d&%d:%s", &user1ID, &user2ID, &gameID)

	var otherUserID int64
	if query.From.ID == user1ID {
		otherUserID = user2ID
	} else {
		otherUserID = user1ID
	}

	bot.userIDToUserMutex.Lock()
	otherUser, ok := bot.userIDToUser[otherUserID]
	bot.userIDToUserMutex.Unlock()

	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Game is too old!"))
		return
	}

	bot.userIDToRematchGameIDMutex.Lock()
	otherUserGameID := bot.userIDToRematchGameID[otherUserID]
	bot.userIDToRematchGameIDMutex.Unlock()
	if otherUserGameID == gameID { // other user has also requested rematch; start the game
		bot.userIDToRematchGameIDMutex.Lock()
		delete(bot.userIDToRematchGameID, user1ID)
		delete(bot.userIDToRematchGameID, user2ID)
		bot.userIDToRematchGameIDMutex.Unlock()

		text := ""
		if err := bot.startGameOfRandomOpponents(query.From, otherUser); err != nil {
			text = err.Error()
		}
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
	} else {
		bot.userIDToRematchGameIDMutex.Lock()
		bot.userIDToRematchGameID[query.From.ID] = gameID
		bot.userIDToRematchGameIDMutex.Unlock()

		msgText := fmt.Sprintf(
			"%s wants to rematch", util.FirstNameElseLastName(query.From))
		msg := tgbotapi.NewMessage(otherUserID, msgText)
		id := strconv.FormatInt(query.From.ID, 10)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Accept", "accept"+id),
				tgbotapi.NewInlineKeyboardButtonData("Reject", "reject"+id),
			),
		)
		bot.api.Send(msg)

		text := "Wait for your opponent's response."
		bot.api.Request(tgbotapi.NewCallback(query.ID, text))
	}
}

func (bot *Bot) handleAcceptedRematch(query *tgbotapi.CallbackQuery) {
	otherUserID, _ := strconv.ParseInt(strings.TrimPrefix(query.Data, "accept"), 10, 64)

	bot.userIDToUserMutex.Lock()
	otherUser, ok := bot.userIDToUser[otherUserID]
	bot.userIDToUserMutex.Unlock()
	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Game is too old!"))
		return
	}

	bot.userIDToRematchGameIDMutex.Lock()
	delete(bot.userIDToRematchGameID, otherUserID)
	bot.userIDToRematchGameIDMutex.Unlock()

	bot.startGameOfRandomOpponents(query.From, otherUser)

	bot.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: query.ID})
}

func (bot *Bot) handleRejectedRematch(query *tgbotapi.CallbackQuery) {
	defer bot.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: query.ID})

	otherUserID, _ := strconv.ParseInt(strings.TrimPrefix(query.Data, "reject"), 10, 64)

	bot.userIDToRematchGameIDMutex.Lock()
	delete(bot.userIDToRematchGameID, otherUserID)
	bot.userIDToRematchGameIDMutex.Unlock()

	bot.userIDToUserMutex.Lock()
	otherUser, ok := bot.userIDToUser[otherUserID]
	bot.userIDToUserMutex.Unlock()

	msg := "Rematch request was rejected"

	bot.api.Send(tgbotapi.NewEditMessageText(query.From.ID, query.Message.MessageID, msg+"."))

	if ok {
		msg += " by " + util.FirstNameElseLastName(otherUser)
	}
	bot.api.Send(tgbotapi.NewMessage(otherUserID, msg))
}

func (bot *Bot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	if inlineQuery.Query == resendQuery {
		bot.resendGame(inlineQuery)
		return
	}

	user := inlineQuery.From

	bot.userIDToCurrentGameMutex.Lock()
	_, ok := bot.userIDToCurrentGame[user.ID]
	bot.userIDToCurrentGameMutex.Unlock()
	if ok {
		bot.api.Request(tgbotapi.InlineConfig{
			InlineQueryID:     inlineQuery.ID,
			Results:           []interface{}{},
			CacheTime:         0,
			SwitchPMText:      "Can't play two games at the same time!",
			SwitchPMParameter: "playingSimultaneously",
		})
		return
	}

	if bot.db.AddPlayer(user.ID, util.FullNameOf(user)) {
		bot.scoreboard.Insert(bot.db.Find(user.ID))
		atomic.AddUint64(&bot.usersJoinedToday, 1)
	}

	game := tgbotapi.NewInlineQueryResultArticleMarkdownV2(
		uuid.NewString(),
		"Othello",
		fmt.Sprintf("Let's Play Othello\\! [🎯](%s)", botPic),
	)
	game.Description = helpMsg
	game.ReplyMarkup = buildJoinToGameKeyboard()
	game.ThumbURL = botPic
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

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		bot.api.Request(tgbotapi.InlineConfig{
			InlineQueryID:     inlineQuery.ID,
			Results:           []interface{}{},
			CacheTime:         0,
			SwitchPMText:      "Game is too old!",
			SwitchPMParameter: "oldGame",
		})
		return
	}

	msgText, replyMarkup := getRunningGameMsgAndReplyMarkup(
		game,
		bot.db.LegalMovesAreShown(game.ActiveUser().ID),
		true,
	)
	msg := tgbotapi.NewInlineQueryResultArticle(
		uuid.NewString(),
		"Send down your current game",
		msgText,
	)
	msg.ReplyMarkup = replyMarkup

	bot.api.Request(tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.ID,
		Results:       []interface{}{msg},
		CacheTime:     0,
	})
}

func (bot *Bot) handleChosenInlineResult(chosenInlineResult *tgbotapi.ChosenInlineResult) {
	user := chosenInlineResult.From
	newID := chosenInlineResult.InlineMessageID

	if chosenInlineResult.Query != resendQuery {
		bot.inlineMessageIDToUserMutex.Lock()
		bot.inlineMessageIDToUser[newID] = user
		bot.inlineMessageIDToUserMutex.Unlock()
		return
	}

	bot.userIDToCurrentGameMutex.Lock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
	}

	bot.gameIDToInlineMessageIDMutex.Lock()
	oldID, ok := bot.gameIDToInlineMessageID[game.ID()]
	if !ok {
		log.Panicf("Invalid state: gamesToInlineMessageIDs does not contain %v.\n", game)
	}
	bot.gameIDToInlineMessageID[game.ID()] = newID
	bot.gameIDToInlineMessageIDMutex.Unlock()

	bot.api.Send(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: oldID,
		},
		Text: fmt.Sprintf("%v has been moved down 🔽", game),
	})

	bot.userIDToCurrentGameMutex.Unlock()

	bot.inlineMessageIDToUserMutex.Lock()
	bot.inlineMessageIDToUser[newID] = user
	delete(bot.inlineMessageIDToUser, oldID)
	bot.inlineMessageIDToUserMutex.Unlock()
}

func (bot *Bot) sendEditMessageTextForGame(
	msgText string,
	replyMarkup *tgbotapi.InlineKeyboardMarkup,
	user1, user2 *tgbotapi.User,
	inlineMessageID string,
) {
	if inlineMessageID != "" {
		bot.api.Send(tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				InlineMessageID: inlineMessageID,
				ReplyMarkup:     replyMarkup,
			},
			Text: msgText,
		})
		return
	}

	bot.userIDToMessageIDMutex.Lock()
	messageID1 := bot.userIDToMessageID[user1.ID]
	messageID2 := bot.userIDToMessageID[user2.ID]
	bot.userIDToMessageIDMutex.Unlock()

	msg1 := tgbotapi.NewEditMessageTextAndMarkup(user1.ID, messageID1, msgText, *replyMarkup)
	msg2 := tgbotapi.NewEditMessageTextAndMarkup(user2.ID, messageID2, msgText, *replyMarkup)

	bot.api.Send(msg1)
	bot.api.Send(msg2)
}

func (bot *Bot) opponentOf(user *tgbotapi.User) (*tgbotapi.User, error) {
	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		return nil, fmt.Errorf("Game is too old!")
	}
	return game.OpponentOf(user), nil
}
