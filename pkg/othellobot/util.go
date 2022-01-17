package othellobot

import (
	"fmt"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getFullNameOf(user *tgbotapi.User) string {
	return user.FirstName + " " + user.LastName
}

func getGameMsg(game *othellogame.Game) string {
	return fmt.Sprintf("Turn of: %s%s\n%s%s: %d\n%s%s: %d\nDon't count your chickens before they hatch!",
		game.ActiveColor(),
		game.ActiveUser().FirstName,
		consts.WHITE_DISK_EMOJI,
		game.WhiteUser().FirstName,
		game.WhiteDisks(),
		consts.BLACK_DISK_EMOJI,
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
		tgbotapi.InlineKeyboardMarkup{InlineKeyboard: game.InlineKeyboard(showLegalMoves)},
	)
}

func getEditedMsgOfGameInline(
	game *othellogame.Game,
	inlineMessageID string,
	showLegalMoves bool,
) tgbotapi.Chattable {
	replyMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: game.InlineKeyboard(showLegalMoves)}
	return tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{
			InlineMessageID: inlineMessageID,
			ReplyMarkup:     &replyMarkup,
		},
		Text: getGameMsg(game),
	}
}

func buildMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(NEW_GAME_BUTTON_TEXT),
			tgbotapi.NewKeyboardButton(SCOREBOARD_BUTTON_TEXT),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(PROFILE_BUTTON_TEXT),
			tgbotapi.NewKeyboardButton(HELP_BUTTON_TEXT),
		),
	)
}

func buildGameModeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonSwitch("Play with friends!", ""),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Play with random opponents!", "playWithRandomOpponent"),
		),
	)
}

func buildJoinToGameKeyboard() *tgbotapi.InlineKeyboardMarkup {
	keyboard := [][]tgbotapi.InlineKeyboardButton{{tgbotapi.NewInlineKeyboardButtonData("Join", "join")}}
	return &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}
