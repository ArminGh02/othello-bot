package cell

type Cell rune

const (
	EMPTY = Cell('•')
	BLACK = Cell('b')
	WHITE = Cell('w')
)

func (c Cell) Emoji() string {
	switch c {
	case EMPTY:
		return "•"
	case BLACK:
		return "⚫️"
	case WHITE:
		return "⚪️"
	default:
		return ""
	}
}
