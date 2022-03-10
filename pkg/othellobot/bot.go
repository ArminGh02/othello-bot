package othellobot

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	cron "github.com/robfig/cron/v3"
)

type Bot struct {
	token                        string
	api                          *tgbotapi.BotAPI
	db                           *database.Handler
	scoreboard                   util.Scoreboard
	waitingPlayer                chan *tgbotapi.User
	inlineMessageIDToUser        map[string]*tgbotapi.User
	gameIDToMovesSequence        map[string][]coord.Coord
	gameIDToInlineMessageID      map[string]string
	userIDToCurrentGame          map[int64]*othellogame.Game
	userIDToLastTimeActive       map[int64]time.Time
	userIDToMessageID            map[int64]int
	userIDToChatBuddy            map[int64]*tgbotapi.User
	userIDToUser                 map[int64]*tgbotapi.User
	userIDToRematchGameID        map[int64]string
	inlineMessageIDToUserMutex   sync.Mutex
	gameIDToMovesSequenceMutex   sync.Mutex
	gameIDToInlineMessageIDMutex sync.Mutex
	userIDToCurrentGameMutex     sync.Mutex
	userIDToLastTimeActiveMutex  sync.Mutex
	userIDToMessageIDMutex       sync.Mutex
	userIDToChatBuddyMutex       sync.Mutex
	userIDToUserMutex            sync.Mutex
	userIDToRematchGameIDMutex   sync.Mutex

	gamesPlayedToday uint64
	usersJoinedToday uint64
}

func New(token, mongodbURI string) *Bot {
	db := database.New(mongodbURI)
	return &Bot{
		token:                   token,
		db:                      db,
		scoreboard:              util.NewScoreboard(db.GetAllPlayers()),
		waitingPlayer:           make(chan *tgbotapi.User, 1),
		inlineMessageIDToUser:   make(map[string]*tgbotapi.User),
		gameIDToMovesSequence:   make(map[string][]coord.Coord),
		gameIDToInlineMessageID: make(map[string]string),
		userIDToCurrentGame:     make(map[int64]*othellogame.Game),
		userIDToLastTimeActive:  make(map[int64]time.Time),
		userIDToMessageID:       make(map[int64]int),
		userIDToChatBuddy:       make(map[int64]*tgbotapi.User),
		userIDToUser:            make(map[int64]*tgbotapi.User),
		userIDToRematchGameID:   make(map[int64]string),
	}
}

func (bot *Bot) Run() {
	var err error
	bot.api, err = tgbotapi.NewBotAPI(bot.token)
	if err != nil {
		log.Panicln(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.api.GetUpdatesChan(updateConfig)

	defer bot.db.Disconnect()

	log.Println("Bot started.")

	c := cron.New()
	c.AddFunc("@daily", func() {
		atomic.SwapUint64(&bot.gamesPlayedToday, 0)
		atomic.SwapUint64(&bot.usersJoinedToday, 0)
	})
	c.Start()

	for update := range updates {
		go bot.handleUpdate(update)
	}
}

func (bot *Bot) handleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		bot.handleMessage(update.Message)
	case update.CallbackQuery != nil:
		bot.handleCallbackQuery(update.CallbackQuery)
	case update.InlineQuery != nil:
		bot.handleInlineQuery(update.InlineQuery)
	case update.ChosenInlineResult != nil:
		bot.handleChosenInlineResult(update.ChosenInlineResult)
	}
}
