package othellogame

import (
	"fmt"

	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/color"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/turn"
	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Game struct {
	// first user's disks are white
	users           [2]*tgbotapi.User
	disksCount      [2]int
	board           [BOARD_SIZE][BOARD_SIZE]cell.Cell
	turn            turn.Turn
	placeableCoords util.CoordSet
	ended           bool
}

func New(user1, user2 *tgbotapi.User) *Game {
	g := &Game{
		users:           [2]*tgbotapi.User{user1, user2},
		disksCount:      [2]int{2, 2},
		turn:            turn.Random(),
		placeableCoords: util.NewCoordSet(),
	}

	mid := len(g.board)/2 - 1
	g.board[mid][mid] = cell.WHITE
	g.board[mid][mid+1] = cell.BLACK
	g.board[mid+1][mid] = cell.BLACK
	g.board[mid+1][mid+1] = cell.WHITE

	return g
}

func (g *Game) ActivePlayer() *tgbotapi.User {
	return g.users[g.turn.Int()]
}

func (g *Game) IsEnded() bool {
	return g.ended
}

func (g *Game) PlaceDisk(where util.Coord, user *tgbotapi.User) error {
	if err := g.checkPlacingDisk(where, user); err != nil {
		return err
	}

	g.board[where.Y][where.X] = g.turn.Cell()
	g.flipDisks(where)

	for i := 0; i < 2; i++ {
		g.passTurn()
		g.updatePlaceableCoords()
		if !g.placeableCoords.IsEmpty() {
			break
		}
	}

	if g.placeableCoords.IsEmpty() {
		g.ended = true
	} else {
		g.updateDisksCount()
	}

	return nil
}

func (g *Game) isTurnOf(user *tgbotapi.User) bool {
	return g.ActivePlayer() == user
}

func (g *Game) checkPlacingDisk(where util.Coord, user *tgbotapi.User) error {
	if !g.isTurnOf(user) {
		return fmt.Errorf("It's not your turn!")
	}
	if g.board[where.Y][where.X] != cell.EMPTY {
		return fmt.Errorf("That cell is not empty!")
	}
	if !g.placeableCoords.Contains(where) {
		return fmt.Errorf("You can't place a disk there!")
	}
	return nil
}

func (g *Game) flipDisks(where util.Coord) {

}

func (g *Game) passTurn() {
	g.turn = !g.turn
}

func (g *Game) updatePlaceableCoords() {

}

func (g *Game) updateDisksCount() {
	white, black := 0, 0
	for _, row := range g.board {
		for _, c := range row {
			if c == cell.WHITE {
				white++
			} else if c == cell.BLACK {
				black++
			}
		}
	}
	g.disksCount[color.WHITE] = white
	g.disksCount[color.BLACK] = black
}
