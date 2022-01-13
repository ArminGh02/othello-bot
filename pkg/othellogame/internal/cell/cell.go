package cell

import (
	"fmt"

	"github.com/ArminGh02/othello-bot/pkg/consts"
)

type Cell rune

const (
	EMPTY = Cell(0)
	BLACK = Cell('b')
	WHITE = Cell('w')
)

func (c Cell) Emoji() string {
	switch c {
	case EMPTY:
		return " "
	case BLACK:
		return consts.BLACK_DISK_EMOJI
	case WHITE:
		return consts.WHITE_DISK_EMOJI
	default:
		panic(fmt.Sprintf("Invalid receiver for Cell.Emoji: %v", c))
	}
}

func (c Cell) Reversed() Cell {
	switch c {
	case BLACK:
		return WHITE
	case WHITE:
		return BLACK
	default:
		panic(fmt.Sprintf("Invalid receiver for Cell.Emoji: %v", c))
	}
}
