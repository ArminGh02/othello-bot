package cell

import (
	"fmt"
	"log"

	"github.com/ArminGh02/othello-bot/pkg/consts"
)

type Cell rune

const (
	Empty = Cell(0)
	Black = Cell('b')
	White = Cell('w')
)

func (c Cell) Emoji() string {
	switch c {
	case Empty:
		return " "
	case Black:
		return consts.BlackDiskEmoji
	case White:
		return consts.WhiteDiskEmoji
	default:
		log.Panicln(fmt.Sprintf("Invalid receiver for Cell.Emoji: %v", c))
		panic("")
	}
}

func (c Cell) Reversed() Cell {
	switch c {
	case Black:
		return White
	case White:
		return Black
	default:
		log.Panicln(fmt.Sprintf("Invalid receiver for Cell.Emoji: %v", c))
		panic("")
	}
}
