package othellobot

import (
	"fmt"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getGameMsg(whiteUser, blackUser *tgbotapi.User, whiteDisks, blackDisks int) string {
	return fmt.Sprintf("%s%s: %d\n%s%s: %d\nDon't count your chickens before they hatch!",
		consts.WHITE_DISK_EMOJI,
		whiteUser.FirstName,
		whiteDisks,
		consts.BLACK_DISK_EMOJI,
		blackUser.FirstName,
		blackDisks,
	)
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
