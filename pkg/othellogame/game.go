package othellogame

import (
	"github.com/ArminGh02/othello-bot/pkg/othellogame/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/turn"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Game struct {
	users [2]*tgbotapi.User
	board [BOARD_SIZE][BOARD_SIZE]cell.Cell
	turn  turn.Turn
}

func New(user1, user2 *tgbotapi.User) *Game {
	g := &Game{
		users: [2]*tgbotapi.User{user1, user2},
		turn:  turn.Random(),
	}

	for i := range g.board {
		for j := range g.board[i] {
			g.board[i][j] = cell.EMPTY
		}
	}

	mid := len(g.board)/2 - 1
	g.board[mid][mid] = cell.WHITE
	g.board[mid][mid+1] = cell.BLACK
	g.board[mid+1][mid] = cell.BLACK
	g.board[mid+1][mid+1] = cell.WHITE

	return g
}
