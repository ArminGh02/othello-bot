package othellobot

import (
	"fmt"
	"sync/atomic"

	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

func (bot *Bot) handleInlineQuery(inlineQuery *tgbotapi.InlineQuery) {
	if inlineQuery.Query == resendQuery {
		bot.resendGame(inlineQuery)
		return
	}

	user := inlineQuery.From

	bot.userIDToCurrentGameMutex.Lock()
	_, ok := bot.userIDToCurrentGame[user.ID]
	bot.userIDToCurrentGameMutex.Unlock()
	if ok {
		bot.api.Request(tgbotapi.InlineConfig{
			InlineQueryID:     inlineQuery.ID,
			Results:           []interface{}{},
			CacheTime:         0,
			SwitchPMText:      "Can't play two games at the same time!",
			SwitchPMParameter: "playingSimultaneously",
		})
		return
	}

	if bot.db.AddPlayer(user.ID, util.FullNameOf(user)) {
		bot.scoreboard.Insert(bot.db.Find(user.ID))
		atomic.AddUint64(&bot.usersJoinedToday, 1)
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

	bot.userIDToCurrentGameMutex.Lock()
	defer bot.userIDToCurrentGameMutex.Unlock()

	game, ok := bot.userIDToCurrentGame[user.ID]
	if !ok {
		bot.api.Request(tgbotapi.InlineConfig{
			InlineQueryID:     inlineQuery.ID,
			Results:           []interface{}{},
			CacheTime:         0,
			SwitchPMText:      "Game is too old!",
			SwitchPMParameter: "oldGame",
		})
		return
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
