package othellobot

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
