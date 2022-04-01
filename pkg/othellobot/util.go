package othellobot

import (
	"fmt"
	"math"
	"strconv"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
		return nil, errTooOldGame
	}
	return game.OpponentOf(user), nil
}

func getRunningGameMsgAndReplyMarkup(
	game *othellogame.Game,
	showLegalMoves, inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	msg = fmt.Sprintf(
		"Turn of: %s%s\n%s%s: %d\n%s%s: %d\nDon't count your chickens before they hatch!",
		game.ActiveColor(),
		util.FirstNameElseLastName(game.ActiveUser()),
		consts.WhiteDiskEmoji,
		util.FirstNameElseLastName(game.WhiteUser()),
		game.WhiteDisks(),
		consts.BlackDiskEmoji,
		util.FirstNameElseLastName(game.BlackUser()),
		game.BlackDisks(),
	)
	return msg, buildGameKeyboard(game, showLegalMoves, inline)
}

func getGameOverMsgAndReplyMarkup(
	game *othellogame.Game,
	botUsername string,
	inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	if winner := game.Winner(); winner == nil {
		msg = "Draw"
	} else {
		msg = fmt.Sprintf(
			"%s%s WON! %d to %d! 🔥",
			game.WinnerColor(),
			util.FirstNameElseLastName(winner),
			int(math.Max(float64(game.WhiteDisks()), float64(game.BlackDisks()))),
			int(math.Min(float64(game.WhiteDisks()), float64(game.BlackDisks()))),
		)
	}
	return msg, buildGameOverKeyboard(game, botUsername, inline)
}

func getSurrenderMsgAndReplyMarkup(
	game *othellogame.Game,
	winner, loser *tgbotapi.User,
	botUsername string,
	inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	msg = fmt.Sprintf(
		"%s surrendered to %s!",
		util.FirstNameElseLastName(loser),
		util.FirstNameElseLastName(winner),
	)
	return msg, buildGameOverKeyboard(game, botUsername, inline)
}

func getEarlyEndMsgAndReplyMarkup(
	game *othellogame.Game,
	loser *tgbotapi.User,
	botUsername string,
	inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	msg = fmt.Sprintf(
		"Game ended due to inactivity of %s.",
		util.FirstNameElseLastName(loser),
	)
	return msg, buildGameOverKeyboard(game, botUsername, inline)
}

func buildGameKeyboard(
	game *othellogame.Game,
	showLegalMoves, inline bool,
) *tgbotapi.InlineKeyboardMarkup {
	var button1 tgbotapi.InlineKeyboardButton
	if inline {
		button1 = tgbotapi.InlineKeyboardButton{
			Text:                         "🔽 Send down",
			SwitchInlineQueryCurrentChat: &resendQuery,
		}
	} else {
		button1 = tgbotapi.NewInlineKeyboardButtonData("💬 Chat", "chat")
	}

	var button2text string
	if showLegalMoves {
		button2text = "Hide legal moves"
	} else {
		button2text = "Show legal moves"
	}

	row2 := tgbotapi.NewInlineKeyboardRow(
		button1,
		tgbotapi.NewInlineKeyboardButtonData(button2text, "toggleShowingLegalMoves"),
	)

	row3 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔚 End", "end"),
		tgbotapi.NewInlineKeyboardButtonData("🏳️ Surrender", "surrender"),
	)

	keyboard := append(game.InlineKeyboard(showLegalMoves), buildProfilesRow(game), row2, row3)
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func buildProfilesRow(game *othellogame.Game) []tgbotapi.InlineKeyboardButton {
	whiteProfile := fmt.Sprintf(
		"%s%s: %d",
		consts.WhiteDiskEmoji,
		util.FirstNameElseLastName(game.WhiteUser()),
		game.WhiteDisks(),
	)
	blackProfile := fmt.Sprintf(
		"%s%s: %d",
		consts.BlackDiskEmoji,
		util.FirstNameElseLastName(game.BlackUser()),
		game.BlackDisks(),
	)
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			whiteProfile, "profile"+strconv.FormatInt(game.WhiteUser().ID, 10)),
		tgbotapi.NewInlineKeyboardButtonData(
			blackProfile, "profile"+strconv.FormatInt(game.BlackUser().ID, 10)),
	)
}

func buildGameOverKeyboard(
	game *othellogame.Game,
	botUsername string,
	inline bool,
) *tgbotapi.InlineKeyboardMarkup {
	button2data := "replay"
	if game.WhiteStarted() {
		button2data += "w"
	} else {
		button2data += "b"
	}
	button2data += game.ID()

	var button1, button2 tgbotapi.InlineKeyboardButton
	if inline {
		inlineQuery := ""
		button1 = tgbotapi.InlineKeyboardButton{
			Text:                         "🔄 Play again",
			SwitchInlineQueryCurrentChat: &inlineQuery,
		}

		url := fmt.Sprintf("https://telegram.me/%s?start=%s", botUsername, button2data)
		button2 = tgbotapi.NewInlineKeyboardButtonURL("🎞 Game replay", url)
	} else {
		rematchData := fmt.Sprint(
			"rematch", game.WhiteUser().ID, "&", game.BlackUser().ID, ":", game.ID())
		button1 = tgbotapi.NewInlineKeyboardButtonData("🔄 Rematch", rematchData)
		button2 = tgbotapi.NewInlineKeyboardButtonData("🎞 Game replay", button2data)
	}
	row := tgbotapi.NewInlineKeyboardRow(button1, button2)
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(game.EndInlineKeyboard(), buildProfilesRow(game), row),
	}
}

func buildMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(newGameButtonText),
			tgbotapi.NewKeyboardButton(scoreboardButtonText),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(profileButtonText),
			tgbotapi.NewKeyboardButton(helpButtonText),
		),
	)
}

func buildGameModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("Play with friends!", ""),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Play with random opponents!",
				"playWithRandomOpponent",
			),
		),
	)
}

func buildJoinToGameKeyboard() *tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Join", "join"),
		),
	)
	return &keyboard
}
