package turn

import (
	"math/rand"

	"github.com/ArminGh02/othello-bot/pkg/othellogame/internal/cell"
)

type Turn bool

const (
	WHITE = Turn(false)
	BLACK = Turn(true)
)

func Random() Turn {
	return rand.Int31n(2) == 0
}

func (t Turn) Int() int {
	if t {
		return 1
	}
	return 0
}

func (t Turn) Cell() cell.Cell {
	if t == BLACK {
		return cell.BLACK
	}
	return cell.WHITE
}
