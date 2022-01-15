package othellogame

import (
	"fmt"

	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/color"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/direction"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/turn"
	"github.com/ArminGh02/othello-bot/pkg/util"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var offset = [direction.COUNT]util.Coord{
	{X: -1, Y: -1},
	{X: 0, Y: -1},
	{X: 1, Y: -1},
	{X: -1, Y: 0},
	{X: 1, Y: 0},
	{X: -1, Y: 1},
	{X: 0, Y: 1},
	{X: 1, Y: 1},
}

type Game struct {
	users           [2]tgbotapi.User
	disksCount      [2]int
	board           [BOARD_SIZE][BOARD_SIZE]cell.Cell
	turn            turn.Turn
	placeableCoords util.CoordSet
	ended           bool
}

func New(user1, user2 *tgbotapi.User) *Game {
	g := &Game{
		users:           [2]tgbotapi.User{*user1, *user2},
		disksCount:      [2]int{2, 2},
		turn:            turn.Random(),
		placeableCoords: util.NewCoordSet(),
	}

	mid := len(g.board)/2 - 1
	g.board[mid][mid] = cell.WHITE
	g.board[mid][mid+1] = cell.BLACK
	g.board[mid+1][mid] = cell.BLACK
	g.board[mid+1][mid+1] = cell.WHITE

	g.updatePlaceableCoords()

	return g
}

func (g *Game) String() string {
	return fmt.Sprintf("Game between %s and %s", g.users[0].UserName, g.users[1].UserName)
}

func (g *Game) ActiveColor() string {
	return g.turn.Cell().Emoji()
}

func (g *Game) ActiveUser() *tgbotapi.User {
	return &g.users[g.turn.Int()]
}

func (g *Game) WhiteUser() *tgbotapi.User {
	return &g.users[color.WHITE]
}

func (g *Game) BlackUser() *tgbotapi.User {
	return &g.users[color.BLACK]
}

func (g *Game) WhiteDisks() int {
	return g.disksCount[color.WHITE]
}

func (g *Game) BlackDisks() int {
	return g.disksCount[color.BLACK]
}

func (g *Game) IsEnded() bool {
	return g.ended
}

func (g *Game) InlineKeyboard() [][]tgbotapi.InlineKeyboardButton {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, len(g.board))
	for i := range keyboard {
		keyboard[i] = make([]tgbotapi.InlineKeyboardButton, len(g.board[i]))
		for j, cell := range g.board[i] {
			keyboard[i][j] = tgbotapi.NewInlineKeyboardButtonData(
				cell.Emoji(),
				fmt.Sprintf("%d_%d", j, i),
			)
		}
	}
	return keyboard
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
	return *g.ActiveUser() == *user
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
	opponent := g.turn.Cell().Reversed()
	directionsToFlip := g.findDirectionsToFlip(where)
	for _, dir := range directionsToFlip {
		for x, y := where.X+offset[dir].X, where.Y+offset[dir].Y; g.board[y][x] == opponent; {
			g.board[y][x] = g.turn.Cell()
			x += offset[dir].X
			y += offset[dir].Y
		}
	}
}

func (g *Game) findDirectionsToFlip(where util.Coord) []direction.Direction {
	opponent := g.turn.Cell().Reversed()
	res := make([]direction.Direction, 0, direction.COUNT)
	for i := direction.NORTH_WEST; i < direction.COUNT; i++ {
		x, y := where.X+offset[i].X, where.Y+offset[i].Y
		if isValidCoord(x, y, len(g.board)) && g.board[y][x] == opponent {
			for {
				x += offset[i].X
				y += offset[i].Y

				if !isValidCoord(x, y, len(g.board)) {
					break
				}

				switch g.board[y][x] {
				case g.turn.Cell():
					res = append(res, i)
					break
				case cell.EMPTY:
					break
				}
			}
		}
	}
	return res
}

func isValidCoord(x, y, length int) bool {
	return x >= 0 && y >= 0 && x < length && y < length
}

func (g *Game) passTurn() {
	g.turn = !g.turn
}

func (g *Game) updatePlaceableCoords() {
	g.placeableCoords.Clear()
	for y := range g.board {
		for x := range g.board[y] {
			if coord := util.NewCoord(x, y); g.isPlaceableCoord(coord) {
				g.placeableCoords.Insert(coord)
			}
		}
	}
}

func (g *Game) isPlaceableCoord(where util.Coord) bool {
	return len(g.findDirectionsToFlip(where)) > 0
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
