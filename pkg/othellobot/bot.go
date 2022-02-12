package othellobot

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/gifmaker"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type Bot struct {
	token                        string
	api                          *tgbotapi.BotAPI
	db                           *database.Handler
	scoreboard                   util.Scoreboard
	inlineMessageIDsToUsers      map[string]*tgbotapi.User
	gamesToInlineMessageIDs      map[*othellogame.Game]string
	usersToCurrentGames          map[tgbotapi.User]*othellogame.Game
	usersToLastTimeActive        map[tgbotapi.User]time.Time
	usersToMessageIDs            map[tgbotapi.User]int
	usersToChatBuddy             map[tgbotapi.User]*tgbotapi.User
	inlineMessageIDsToUsersMutex sync.Mutex
	gamesToInlineMessageIDsMutex sync.Mutex
	usersToCurrentGamesMutex     sync.Mutex
	usersToLastTimeActiveMutex   sync.Mutex
	usersToMessageIDsMutex       sync.Mutex
	usersToChatBuddyMutex        sync.Mutex
	waitingPlayer                chan *tgbotapi.User
}

func New(token, mongodbURI string) *Bot {
	db := database.New(mongodbURI)
	return &Bot{
		token:                   token,
		db:                      db,
		scoreboard:              util.NewScoreboard(db.GetAllPlayers()),
		inlineMessageIDsToUsers: make(map[string]*tgbotapi.User),
		gamesToInlineMessageIDs: make(map[*othellogame.Game]string),
		usersToCurrentGames:     make(map[tgbotapi.User]*othellogame.Game),
		usersToLastTimeActive:   make(map[tgbotapi.User]time.Time),
		usersToMessageIDs:       make(map[tgbotapi.User]int),
		usersToChatBuddy:        make(map[tgbotapi.User]*tgbotapi.User),
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

	if strings.HasPrefix(message.Text, "End chat with") {
		bot.usersToChatBuddyMutex.Lock()
		delete(bot.usersToChatBuddy, *message.From)
		bot.usersToChatBuddyMutex.Unlock()

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

		bot.usersToChatBuddyMutex.Lock()
		user2, ok := bot.usersToChatBuddy[*user1]
		if !ok {
			break
		}
		bot.usersToChatBuddyMutex.Unlock()

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
		user := message.From

		if arg := message.CommandArguments(); strings.HasPrefix(arg, "replay") {
			bot.sendGameReplay(user, arg)
			break
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
		}

		log.Printf("Bot started by %v.", user)
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
	bot.api.Send(tgbotapi.NewMessage(message.Chat.ID, helpMsg))
}

func (bot *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	data := query.Data

	if strings.HasPrefix(data, "replay") {
		bot.sendGameReplay(query.From, query.Data)
		return
	}

	if match, _ := regexp.MatchString(`^\d+_\d+$`, data); match {
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
	case "end":
		bot.handleEndEarly(query)
	case "gameOver":
		bot.api.Request(tgbotapi.NewCallback(query.ID, "Game is over!"))
	case "chat":
		bot.startChatBetweenOpponents(query)
	}
}

func (bot *Bot) sendGameReplay(user *tgbotapi.User, data string) {
	data = strings.TrimPrefix(data, "replay")

	var whiteStarted bool
	if data[0] == 'w' {
		whiteStarted = true
	} else {
		whiteStarted = false
	}

	var movesSequence []coord.Coord
	err := json.Unmarshal([]byte(data[1:]), &movesSequence)
	if err != nil {
		log.Panicln(err)
	}

	gifFilename := uuid.NewString() + ".gif"
	gifmaker.Make(gifFilename, movesSequence, whiteStarted)

	bot.api.Send(tgbotapi.NewAnimation(user.ID, tgbotapi.FilePath(gifFilename)))

	err = os.Remove(gifFilename)
	if err != nil {
		log.Panicln(err)
	}
}

func (bot *Bot) placeDisk(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
	}

	var where coord.Coord
	fmt.Sscanf(query.Data, "%d_%d", &where.X, &where.Y)

	err := game.PlaceDisk(where, user)
	if err != nil {
		bot.api.Request(tgbotapi.NewCallback(query.ID, err.Error()))
	} else if game.IsEnded() {
		bot.handleGameEnd(game, query)
	} else {
		bot.usersToLastTimeActiveMutex.Lock()
		bot.usersToLastTimeActive[*user] = time.Now()
		bot.usersToLastTimeActiveMutex.Unlock()

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
}

func (bot *Bot) cleanUp(game *othellogame.Game, query *tgbotapi.CallbackQuery) {
	user1 := *game.WhiteUser()
	user2 := *game.BlackUser()

	delete(bot.usersToCurrentGames, user1)
	delete(bot.usersToCurrentGames, user2)

	if query.InlineMessageID != "" {
		bot.inlineMessageIDsToUsersMutex.Lock()
		delete(bot.inlineMessageIDsToUsers, query.InlineMessageID)
		bot.inlineMessageIDsToUsersMutex.Unlock()

		bot.gamesToInlineMessageIDsMutex.Lock()
		delete(bot.gamesToInlineMessageIDs, game)
		bot.gamesToInlineMessageIDsMutex.Unlock()
	} else {
		bot.usersToMessageIDsMutex.Lock()
		delete(bot.usersToMessageIDs, user1)
		delete(bot.usersToMessageIDs, user2)
		bot.usersToMessageIDsMutex.Unlock()
	}
}

func (bot *Bot) startNewGameWithFriend(query *tgbotapi.CallbackQuery) {
	bot.inlineMessageIDsToUsersMutex.Lock()
	user1, ok := bot.inlineMessageIDsToUsers[query.InlineMessageID]
	if !ok {
		log.Panicf(
			"Invalid state: inlineMessageIDsToUsers does not contain %v.\n",
			query.InlineMessageID,
		)
	}
	bot.inlineMessageIDsToUsersMutex.Unlock()

	user2 := query.From

	if bot.db.AddPlayer(user2.ID, util.FullNameOf(user2)) {
		bot.scoreboard.Insert(bot.db.Find(user2.ID))
	}

	game := othellogame.New(user1, user2)

	log.Printf("Started %v.\n", game)

	bot.gamesToInlineMessageIDsMutex.Lock()
	bot.gamesToInlineMessageIDs[game] = query.InlineMessageID
	bot.gamesToInlineMessageIDsMutex.Unlock()

	now := time.Now()
	bot.usersToLastTimeActiveMutex.Lock()
	bot.usersToLastTimeActive[*user1] = now
	bot.usersToLastTimeActive[*user2] = now
	bot.usersToLastTimeActiveMutex.Unlock()

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game

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

	log.Printf("Started %s.\n", game)

	now := time.Now()
	bot.usersToLastTimeActiveMutex.Lock()
	bot.usersToLastTimeActive[*user1] = now
	bot.usersToLastTimeActive[*user2] = now
	bot.usersToLastTimeActiveMutex.Unlock()

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	bot.usersToCurrentGames[*user1] = game
	bot.usersToCurrentGames[*user2] = game

	msgText, replyMarkup := getRunningGameMsgAndReplyMarkup(
		game,
		bot.db.LegalMovesAreShown(game.ActiveUser().ID),
		false,
	)
	msg1 := tgbotapi.NewMessage(user1.ID, msgText)
	msg2 := tgbotapi.NewMessage(user2.ID, msgText)
	msg1.ReplyMarkup = replyMarkup
	msg2.ReplyMarkup = replyMarkup

	msg, _ := bot.api.Send(msg1)
	bot.usersToMessageIDsMutex.Lock()
	bot.usersToMessageIDs[*user1] = msg.MessageID
	bot.usersToMessageIDsMutex.Unlock()

	msg, _ = bot.api.Send(msg2)
	bot.usersToMessageIDsMutex.Lock()
	bot.usersToMessageIDs[*user2] = msg.MessageID
	bot.usersToMessageIDsMutex.Unlock()
}

func (bot *Bot) toggleShowingLegalMoves(query *tgbotapi.CallbackQuery) {
	user := query.From

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
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
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", loser)
	}

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

	bot.usersToCurrentGamesMutex.Unlock()

	bot.db.IncrementWins(winner.ID)
	bot.db.IncrementLosses(loser.ID)
	bot.scoreboard.UpdateRankOf(winner.ID, 1, 0)
	bot.scoreboard.UpdateRankOf(loser.ID, 0, 1)

	log.Printf("%s surrendered in %v.\n", loser, game)
}

func (bot *Bot) handleEndEarly(query *tgbotapi.CallbackQuery) {
	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	user1 := query.From

	game := bot.usersToCurrentGames[*user1]

	if game.IsTurnOf(user1) {
		bot.api.Request(
			tgbotapi.NewCallback(query.ID, "You can't end the game in your turn."),
		)
		return
	}

	user2 := game.OpponentOf(user1)

	bot.usersToLastTimeActiveMutex.Lock()
	lastActiveTime := bot.usersToLastTimeActive[*user2]
	bot.usersToLastTimeActiveMutex.Unlock()

	if secondsSinceLastActive := time.Since(lastActiveTime).Seconds(); secondsSinceLastActive > 90 {
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

		bot.api.Request(tgbotapi.CallbackConfig{
			CallbackQueryID: query.ID,
		})
	} else {
		msg := fmt.Sprintf(
			"You can end the game if your opponent doesn't place a disk for %d seconds.",
			90-int(secondsSinceLastActive),
		)
		bot.api.Request(tgbotapi.NewCallback(query.ID, msg))
	}
}

func (bot *Bot) startChatBetweenOpponents(query *tgbotapi.CallbackQuery) {
	user1 := query.From
	user2 := bot.opponentOf(user1)

	msg := tgbotapi.NewMessage(user1.ID, "Chat with your opponent:")
	buttonText := fmt.Sprint("End chat with ", util.FirstNameElseLastName(user2))
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(buttonText),
		),
	)
	bot.api.Send(msg)

	bot.usersToChatBuddyMutex.Lock()
	bot.usersToChatBuddy[*user1] = user2
	bot.usersToChatBuddyMutex.Unlock()
}

func (bot *Bot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	if inlineQuery.Query == resendQuery {
		bot.resendGame(inlineQuery)
	}

	user := inlineQuery.From

	if bot.db.AddPlayer(user.ID, util.FullNameOf(user)) {
		bot.scoreboard.Insert(bot.db.Find(user.ID))
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

	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
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
		bot.inlineMessageIDsToUsersMutex.Lock()
		bot.inlineMessageIDsToUsers[newID] = user
		bot.inlineMessageIDsToUsersMutex.Unlock()
		return
	}

	bot.usersToCurrentGamesMutex.Lock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
	}

	bot.gamesToInlineMessageIDsMutex.Lock()
	oldID, ok := bot.gamesToInlineMessageIDs[game]
	if !ok {
		log.Panicf("Invalid state: gamesToInlineMessageIDs does not contain %v.\n", game)
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

func (bot *Bot) sendEditMessageTextForGame(
	msgText string,
	replyMarkup *tgbotapi.InlineKeyboardMarkup,
	user1, user2 *tgbotapi.User,
	inlineMessageID string,
) {
	if inlineMessageID != "" {
		msg := tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				InlineMessageID: inlineMessageID,
				ReplyMarkup:     replyMarkup,
			},
			Text: msgText,
		}
		bot.api.Send(msg)
		return
	}

	bot.usersToMessageIDsMutex.Lock()
	messageID1 := bot.usersToMessageIDs[*user1]
	messageID2 := bot.usersToMessageIDs[*user2]
	bot.usersToMessageIDsMutex.Unlock()

	msg1 := tgbotapi.NewEditMessageTextAndMarkup(user1.ID, messageID1, msgText, *replyMarkup)
	msg2 := tgbotapi.NewEditMessageTextAndMarkup(user2.ID, messageID2, msgText, *replyMarkup)

	bot.api.Send(msg1)
	bot.api.Send(msg2)
}

func (bot *Bot) opponentOf(user *tgbotapi.User) *tgbotapi.User {
	bot.usersToCurrentGamesMutex.Lock()
	defer bot.usersToCurrentGamesMutex.Unlock()

	game, ok := bot.usersToCurrentGames[*user]
	if !ok {
		log.Panicf("Invalid state: usersToCurrentGames does not contain %v.\n", user)
	}
	return game.OpponentOf(user)
}
