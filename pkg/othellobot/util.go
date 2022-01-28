package othellobot

import (
	"fmt"
	"math"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getFullNameOf(user *tgbotapi.User) string {
	return user.FirstName + " " + user.LastName
}

func getGameMsg(game *othellogame.Game) string {
	return fmt.Sprintf(
		"Turn of: %s%s\n%s%s: %d\n%s%s: %d\nDon't count your chickens before they hatch!",
		game.ActiveColor(),
		game.ActiveUser().FirstName,
		consts.WhiteDiskEmoji,
		game.WhiteUser().FirstName,
		game.WhiteDisks(),
		consts.BlackDiskEmoji,
		game.BlackUser().FirstName,
		game.BlackDisks(),
	)
}

func getEditedMsgOfGame(
	game *othellogame.Game,
	query *tgbotapi.CallbackQuery,
	userID int64,
	showLegalMoves bool,
) tgbotapi.Chattable {
	if query.InlineMessageID != "" {
		return getEditedMsgOfGameInline(
			game,
			query.InlineMessageID,
			showLegalMoves,
		)
	}
	return tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		getGameMsg(game),
		*buildGameKeyboard(game, showLegalMoves, false),
	)
}

func getEditedMsgOfGameInline(
	game *othellogame.Game,
	inlineMessageID string,
	showLegalMoves bool,
) tgbotapi.Chattable {
	return tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: inlineMessageID,
			ReplyMarkup:     buildGameKeyboard(game, showLegalMoves, true),
		},
		Text: getGameMsg(game),
	}
}

func getGameOverMsg(game *othellogame.Game, query *tgbotapi.CallbackQuery) tgbotapi.Chattable {
	var msgText string
	if winner := game.Winner(); winner == nil {
		msgText = "Draw"
	} else {
		msgText = fmt.Sprintf(
			"%s%s WON! %d to %d! 🔥",
			game.WinnerColor(),
			winner.FirstName,
			int(math.Max(float64(game.WhiteDisks()), float64(game.BlackDisks()))),
			int(math.Min(float64(game.WhiteDisks()), float64(game.BlackDisks()))),
		)
	}

	inline := query.InlineMessageID != ""

	replyMarkup := buildGameOverKeyboard(game, inline)

	if inline {
		return tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				InlineMessageID: query.InlineMessageID,
				ReplyMarkup:     replyMarkup,
			},
			Text: msgText,
		}
	}
	return tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		msgText,
		*replyMarkup,
	)
}

func getSurrenderMsg(
	game *othellogame.Game,
	query *tgbotapi.CallbackQuery,
	winner, loser *tgbotapi.User,
) tgbotapi.Chattable {
	msgText := fmt.Sprintf("%s surrendered to %s!", loser.FirstName, winner.FirstName)
	if query.InlineMessageID != "" {
		return tgbotapi.EditMessageTextConfig{
			BaseEdit: tgbotapi.BaseEdit{
				InlineMessageID: query.InlineMessageID,
				ReplyMarkup:     buildGameOverKeyboard(game, true),
			},
			Text: msgText,
		}
	}
	return tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		msgText,
		*buildGameOverKeyboard(game, false),
	)
}

func buildGameKeyboard(game *othellogame.Game, showLegalMoves, inline bool) *tgbotapi.InlineKeyboardMarkup {
	keyboard := game.InlineKeyboard(showLegalMoves)

	whiteProfile := fmt.Sprintf(
		"%s%s: %d",
		consts.WhiteDiskEmoji,
		game.WhiteUser().FirstName,
		game.WhiteDisks(),
	)
	blackProfile := fmt.Sprintf(
		"%s%s: %d",
		consts.BlackDiskEmoji,
		game.BlackUser().FirstName,
		game.BlackDisks(),
	)
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(whiteProfile, "whiteProfile"),
		tgbotapi.NewInlineKeyboardButtonData(blackProfile, "blackProfile"),
	)
	keyboard = append(keyboard, row)

	var buttonText string
	if showLegalMoves {
		buttonText = "Hide legal moves"
	} else {
		buttonText = "Show legal moves"
	}
	row = tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(buttonText, "toggleShowingLegalMoves"),
		tgbotapi.NewInlineKeyboardButtonData("🏳️ Surrender", "surrender"),
	)
	if inline {
		row = append(row, tgbotapi.InlineKeyboardButton{
			Text:                         "🔽 Send down",
			SwitchInlineQueryCurrentChat: &resendQuery,
		})
	}
	keyboard = append(keyboard, row)

	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

func buildGameOverKeyboard(game *othellogame.Game, inline bool) *tgbotapi.InlineKeyboardMarkup {
	var button tgbotapi.InlineKeyboardButton
	if inline {
		inlineQuery := ""
		button = tgbotapi.InlineKeyboardButton{
			Text:                         "Play again 🔄",
			SwitchInlineQueryCurrentChat: &inlineQuery,
		}
	} else {
		button = tgbotapi.NewInlineKeyboardButtonData("Rematch 🔄", "rematch")
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
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Join", "join")),
	)
	return &keyboard
}
