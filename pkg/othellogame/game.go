package othellogame

import (
	"errors"
	"fmt"
	"log"

	"github.com/ArminGh02/othello-bot/pkg/consts"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/color"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/direction"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/turn"
	"github.com/ArminGh02/othello-bot/pkg/util"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
	"github.com/ArminGh02/othello-bot/pkg/util/sets"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/xid"
)

var offset = [direction.Count]coord.Coord{
	{-1, -1},
	{0, -1},
	{1, -1},
	{-1, 0},
	{1, 0},
	{-1, 1},
	{0, 1},
	{1, 1},
}

type Game struct {
	id              string
	users           [2]*tgbotapi.User
	disksCount      [2]int
	board           [boardSize][boardSize]cell.Cell
	turn            turn.Turn
	placeableCoords sets.Set[coord.Coord]
	ended           bool
	whiteStarted    bool
	movesSequence   []coord.Coord
}

func New(user1, user2 *tgbotapi.User) *Game {
	game := &Game{
		id:              xid.New().String(),
		users:           [2]*tgbotapi.User{user1, user2},
		disksCount:      [2]int{2, 2},
		turn:            turn.Random(),
		placeableCoords: sets.New[coord.Coord](),
		movesSequence:   make([]coord.Coord, 0, boardSize*boardSize-4),
	}

	game.whiteStarted = game.turn == turn.White

	mid := len(game.board)/2 - 1
	game.board[mid][mid] = cell.White
	game.board[mid][mid+1] = cell.Black
	game.board[mid+1][mid] = cell.Black
	game.board[mid+1][mid+1] = cell.White

	game.updatePlaceableCoords()

	return game
}

func (game *Game) String() string {
	return fmt.Sprintf(
		"Game between %s and %s",
		util.UsernameElseName(game.users[0]),
		util.UsernameElseName(game.users[1]),
	)
}

func (game *Game) ID() string {
	return game.id
}

func (game *Game) Board() [][]cell.Cell {
	res := make([][]cell.Cell, len(game.board))
	for i := range game.board {
		res[i] = game.board[i][:]
	}
	return res
}

func (game *Game) ActiveColor() string {
	return game.turn.Cell().Emoji()
}

func (game *Game) ActiveUser() *tgbotapi.User {
	return game.users[game.turn.Int()]
}

func (game *Game) WhiteUser() *tgbotapi.User {
	return game.users[color.White]
}

func (game *Game) BlackUser() *tgbotapi.User {
	return game.users[color.Black]
}

func (game *Game) WhiteDisks() int {
	return game.disksCount[color.White]
}

func (game *Game) BlackDisks() int {
	return game.disksCount[color.Black]
}

func (game *Game) IsEnded() bool {
	return game.ended
}

func (game *Game) Winner() *tgbotapi.User {
	if game.disksCount[color.White] == game.disksCount[color.Black] {
		return nil
	}
	if game.disksCount[color.White] > game.disksCount[color.Black] {
		return game.users[color.White]
	}
	return game.users[color.Black]
}

func (game *Game) Loser() *tgbotapi.User {
	winner := game.Winner()
	if winner == nil {
		return nil
	}
	return game.OpponentOf(winner)
}

func (game *Game) OpponentOf(user *tgbotapi.User) *tgbotapi.User {
	if *user == *game.WhiteUser() {
		return game.BlackUser()
	}
	if *user == *game.BlackUser() {
		return game.WhiteUser()
	}
	log.Panicln("Invalid state: OpponentOf called with an argument unequal to both game users.")
	panic("")
}

func (game *Game) WinnerColor() string {
	winner := game.Winner()
	if winner == nil {
		log.Panicln("Invalid state: WinnerColor called when the game is a draw.")
	}
	if *winner == *game.users[color.White] {
		return cell.White.Emoji()
	}
	return cell.Black.Emoji()
}

func (game *Game) InlineKeyboard(showLegalMoves bool) [][]tgbotapi.InlineKeyboardButton {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, len(game.board))
	for y := range game.board {
		keyboard[y] = make([]tgbotapi.InlineKeyboardButton, len(game.board[y]))
		for x, cell := range game.board[y] {
			buttonText := cell.Emoji()
			if showLegalMoves && game.placeableCoords.Contains(coord.New(x, y)) {
				buttonText = consts.LegalMoveEmoji
			}

			keyboard[y][x] = tgbotapi.NewInlineKeyboardButtonData(
				buttonText,
				fmt.Sprintf("%d_%d", x, y),
			)
		}
	}
	return keyboard
}

func (game *Game) EndInlineKeyboard() [][]tgbotapi.InlineKeyboardButton {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, len(game.board))
	for y := range game.board {
		keyboard[y] = make([]tgbotapi.InlineKeyboardButton, len(game.board[y]))
		for x, cell := range game.board[y] {
			keyboard[y][x] = tgbotapi.NewInlineKeyboardButtonData(
				cell.Emoji(),
				"gameOver",
			)
		}
	}
	return keyboard
}

func (game *Game) WhiteStarted() bool {
	return game.whiteStarted
}

func (game *Game) MovesSequence() []coord.Coord {
	return game.movesSequence
}

func (game *Game) SetTurn(white bool) {
	game.turn = turn.Turn(!white)
	if len(game.movesSequence) == 0 {
		game.whiteStarted = white
	}
}

func (game *Game) PlaceDisk(where coord.Coord, user *tgbotapi.User) error {
	if err := game.checkPlacingDisk(where, user); err != nil {
		return err
	}
	game.PlaceDiskUnchecked(where)
	return nil
}

func (game *Game) PlaceDiskUnchecked(where coord.Coord) {
	game.board[where.Y][where.X] = game.turn.Cell()
	game.flipDisks(where)

	for i := 0; i < 2; i++ {
		game.passTurn()
		game.updatePlaceableCoords()
		if !game.placeableCoords.IsEmpty() {
			break
		}
	}

	if game.placeableCoords.IsEmpty() {
		game.ended = true
	}

	game.movesSequence = append(game.movesSequence, where)
}

func (game *Game) IsTurnOf(user *tgbotapi.User) bool {
	return *game.ActiveUser() == *user
}

func (game *Game) checkPlacingDisk(where coord.Coord, user *tgbotapi.User) error {
	if !game.IsTurnOf(user) {
		return errors.New("It's not your turn!")
	}
	if game.board[where.Y][where.X] != cell.Empty {
		return errors.New("That cell is not empty!")
	}
	if !game.placeableCoords.Contains(where) {
		return errors.New("You can't place a disk there!")
	}
	return nil
}

func (game *Game) flipDisks(where coord.Coord) {
	opponent := game.turn.Cell().Reversed()
	directionsToFlip := game.findDirectionsToFlip(where, false)
	for _, dir := range directionsToFlip {
		c := coord.Plus(where, offset[dir])
		for game.board[c.Y][c.X] == opponent {
			game.board[c.Y][c.X] = game.turn.Cell()
			c.Plus(offset[dir])
		}
	}
	game.updateDisksCount()
}

func (game *Game) findDirectionsToFlip(
	where coord.Coord,
	mustBeEmptyCell bool,
) []direction.Direction {
	opponent := game.turn.Cell().Reversed()
	res := make([]direction.Direction, 0, direction.Count)

	if mustBeEmptyCell && game.board[where.Y][where.X] != cell.Empty {
		return res
	}

	for dir := direction.NorthWest; dir < direction.Count; dir++ {
		c := coord.Plus(where, offset[dir])
		if isValidCoord(c, len(game.board)) && game.board[c.Y][c.X] == opponent {
		loop:
			for {
				c.Plus(offset[dir])

				if !isValidCoord(c, len(game.board)) {
					break
				}

				switch game.board[c.Y][c.X] {
				case game.turn.Cell():
					res = append(res, dir)
					break loop
				case cell.Empty:
					break loop
				}
			}
		}
	}
	return res
}

func isValidCoord(c coord.Coord, length int) bool {
	return c.X >= 0 && c.Y >= 0 && c.X < length && c.Y < length
}

func (game *Game) passTurn() {
	game.turn = !game.turn
}

func (game *Game) updatePlaceableCoords() {
	game.placeableCoords.Clear()
	for y := range game.board {
		for x := range game.board[y] {
			if c := coord.New(x, y); game.isPlaceableCoord(c) {
				game.placeableCoords.Insert(c)
			}
		}
	}
}

func (game *Game) isPlaceableCoord(where coord.Coord) bool {
	return len(game.findDirectionsToFlip(where, true)) > 0
}

func (game *Game) updateDisksCount() {
	white, black := 0, 0
	for _, row := range game.board {
		for _, c := range row {
			switch c {
			case cell.White:
				white++
			case cell.Black:
				black++
			}
		}
	}
	game.disksCount[color.White] = white
	game.disksCount[color.Black] = black
}
