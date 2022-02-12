package gifmaker

import (
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"

	"github.com/ArminGh02/othello-bot/pkg/othellogame"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/cell"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
)

const (
	x0 = 77
	y0 = 121
)

const (
	diskLength = 39
	cellLength = 44
)

var (
	whiteDisk  = readPngImage("resources/white-disk.png")
	blackDisk  = readPngImage("resources/black-disk.png")
	boardImage = readPngImage("resources/board.png")
)

func Make(movesSequence []coord.Coord, whiteStarts bool) {
	frames := getGameFrames(movesSequence, whiteStarts)
}

func getGameFrames(movesSequence []coord.Coord, whiteStarts bool) []image.Image {
	game := othellogame.New(nil, nil)
	game.SetTurn(whiteStarts)

	res := make([]image.Image, len(movesSequence))
	for _, move := range movesSequence {
		game.PlaceDiskUnchecked(move)
		res = append(res, getGameFrame(game))
	}
	return res
}

func getGameFrame(game *othellogame.Game) image.Image {
	getDiskImage := func(white bool) image.Image {
		if white {
			return whiteDisk
		}
		return blackDisk
	}

	res := cloneImage(boardImage)
	board := game.Board()
	for i := range board {
		for j := range board[i] {
			if board[i][j] == cell.Empty {
				break
			}

			x := x0 + j*cellLength
			y := y0 + i*cellLength
			draw.Draw(
				res,
				image.Rect(x, y, x+diskLength, y+diskLength),
				getDiskImage(board[i][j] == cell.White),
				image.Point{},
				draw.Over,
			)
		}
	}
	return res
}

func readPngImage(filename string) image.Image {
	f, err := os.Open(filename)
	if err != nil {
		log.Panicln(err)
	}

	img, err := png.Decode(f)
	if err != nil {
		log.Panicln(err)
	}
	return img
}
