package othellobot

import (
	"fmt"
	"math"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	return msg, buildGameOverKeyboard(game, inline)
}

func getSurrenderMsgAndReplyMarkup(
	game *othellogame.Game,
	winner, loser *tgbotapi.User,
	inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	msg = fmt.Sprintf(
		"%s surrendered to %s!",
		util.FirstNameElseLastName(loser),
		util.FirstNameElseLastName(winner),
	)
	return msg, buildGameOverKeyboard(game, inline)
}

func getEarlyEndMsgAndReplyMarkup(
	game *othellogame.Game,
	loser *tgbotapi.User,
	inline bool,
) (msg string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	msg = fmt.Sprintf(
		"Game ended due to inactivity of %s.",
		util.FirstNameElseLastName(loser),
	)
	return msg, buildGameOverKeyboard(game, inline)
}

func buildGameKeyboard(
	game *othellogame.Game,
	showLegalMoves, inline bool,
) *tgbotapi.InlineKeyboardMarkup {
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

	row1 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(whiteProfile, "whiteProfile"),
		tgbotapi.NewInlineKeyboardButtonData(blackProfile, "blackProfile"),
	)

	var button tgbotapi.InlineKeyboardButton
	if inline {
		button = tgbotapi.InlineKeyboardButton{
			Text:                         "🔽 Send down",
			SwitchInlineQueryCurrentChat: &resendQuery,
		}
	} else {
		button = tgbotapi.NewInlineKeyboardButtonData("💬 Chat", "chat")
	}

	var buttonText string
	if showLegalMoves {
		buttonText = "Hide legal moves"
	} else {
		buttonText = "Show legal moves"
	}

	row2 := tgbotapi.NewInlineKeyboardRow(
		button,
		tgbotapi.NewInlineKeyboardButtonData(buttonText, "toggleShowingLegalMoves"),
	)

	row3 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔚 End", "end"),
		tgbotapi.NewInlineKeyboardButtonData("🏳️ Surrender", "surrender"),
	)

	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(game.InlineKeyboard(showLegalMoves), row1, row2, row3),
	}
}

func buildGameOverKeyboard(game *othellogame.Game, inline bool) *tgbotapi.InlineKeyboardMarkup {
	var button tgbotapi.InlineKeyboardButton
	if inline {
		inlineQuery := ""
		button = tgbotapi.InlineKeyboardButton{
			Text:                         "🔄 Play again",
			SwitchInlineQueryCurrentChat: &inlineQuery,
		}
	} else {
		button = tgbotapi.NewInlineKeyboardButtonData("🔄 Rematch", "rematch")
	}
	row := tgbotapi.NewInlineKeyboardRow(button)
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(game.EndInlineKeyboard(), row),
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
