package othellobot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func buildMainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎮 New Game"),
			tgbotapi.NewKeyboardButton("🏆 Scoreboard"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👤 Profile"),
			tgbotapi.NewKeyboardButton("❓ Help"),
		),
	)
}