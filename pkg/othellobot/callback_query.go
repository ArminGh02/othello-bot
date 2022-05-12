package othellobot

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/gifmaker"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (bot *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	switch query.Data {
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
		match, _ := regexp.MatchString(`^\d+_\d+$`, query.Data)
		switch {
		case match:
			bot.placeDisk(query)
		case strings.HasPrefix(query.Data, "replay"):
			text := ""
			if err := bot.sendGameReplay(query.From, query.Data); err != nil {
				text = err.Error()
			}
			bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, text))
		case strings.HasPrefix(query.Data, "profile"):
			bot.alertProfile(query)
		case strings.HasPrefix(query.Data, "rematch"):
			bot.handleRematch(query)
		case strings.HasPrefix(query.Data, "accept"):
			bot.handleAcceptedRematch(query)
		case strings.HasPrefix(query.Data, "reject"):
			bot.handleRejectedRematch(query)
		}
	}
}

func (bot *Bot) sendGameReplay(user *tgbotapi.User, data string) error {
	gameID := strings.TrimPrefix(data, "replay")

	bot.gameIDToMovesSequenceMutex.Lock()
	gameData, ok := bot.gameIDToGameData[gameID]
	bot.gameIDToMovesSequenceMutex.Unlock()
	if !ok {
		return errTooOldGame
	}

	gifFilename := gameID + ".gif"
	gifmaker.Make(gifFilename, gameData.moveSequence, gameData.whiteStarts)

	gameGIF := tgbotapi.NewAnimation(user.ID, tgbotapi.FilePath(gifFilename))
	gameGIF.Caption = fmt.Sprintf(
		"%s White: %s | Score: %d\n%s Black: %s | Score: %d",
		consts.WhiteDiskEmoji,
		gameData.whitePlayerName,
		gameData.whiteScore,
		consts.BlackDiskEmoji,
		gameData.blackPlayerName,
		gameData.blackScore,
	)
	bot.api.Send(gameGIF)

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
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, errTooOldGame.Error()))
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
		bot.userIDToLastTimeActive[game.OpponentOf(user).ID] = time.Now()
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
	bot.gameIDToGameData[game.ID()] = newGameData(game)
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
	bot.inlineMessageIDToUserMutex.Unlock()

	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Invitation is too old!"))
		return
	}

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
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, errTooOldGame.Error()))
		return
	}

	bot.db.ToggleLegalMovesAreShown(user.ID)

	if game.IsTurnOf(user) {
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
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, errTooOldGame.Error()))
		return
	}

	bot.gameIDToMovesSequenceMutex.Lock()
	bot.gameIDToGameData[game.ID()] = newGameData(game)
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
		bot.api.Request(tgbotapi.NewCallback(query.ID, errTooOldGame.Error()))
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
		bot.gameIDToGameData[game.ID()] = newGameData(game)
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

	bot.api.Request(tgbotapi.CallbackConfig{CallbackQueryID: query.ID})

	bot.userIDToChatBuddyMutex.Lock()
	bot.userIDToChatBuddy[user1.ID] = user2
	bot.userIDToChatBuddyMutex.Unlock()
}

func (bot *Bot) handleRematch(query *tgbotapi.CallbackQuery) {
	var user1ID, user2ID int64
	var gameID string
	fmt.Sscanf(query.Data, "rematch%d&%d:%s", &user1ID, &user2ID, &gameID)

	otherUserID := user1ID
	if query.From.ID == user1ID {
		otherUserID = user2ID
	}

	bot.userIDToUserMutex.Lock()
	otherUser, ok := bot.userIDToUser[otherUserID]
	bot.userIDToUserMutex.Unlock()

	if !ok {
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, errTooOldGame.Error()))
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
		bot.api.Request(tgbotapi.NewCallbackWithAlert(query.ID, errTooOldGame.Error()))
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

	msg := "Rematch request was rejected."
	bot.api.Send(tgbotapi.NewEditMessageText(query.From.ID, query.Message.MessageID, msg))

	msg = util.FirstNameElseLastName(query.From) + " rejected the rematch request."
	bot.api.Send(tgbotapi.NewMessage(otherUserID, msg))
}
