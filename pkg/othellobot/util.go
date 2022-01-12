package othellobot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
			tgbotapi.NewInlineKeyboardButtonData("Play with friends!", "friends"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Play with random opponents!", "randomOpponent"),
		),
	)
}
