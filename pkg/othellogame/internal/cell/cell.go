package cell

import "fmt"

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
		return "⚫️"
	case WHITE:
		return "⚪️"
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
