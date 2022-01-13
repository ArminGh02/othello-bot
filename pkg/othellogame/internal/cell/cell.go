package cell

import "fmt"

type Cell rune

const (
	EMPTY = Cell(' ')
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
