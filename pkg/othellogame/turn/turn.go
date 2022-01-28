package turn

import (
	"math/rand"

	"github.com/ArminGh02/othello-bot/pkg/othellogame/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/color"
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
	if t == BLACK {
		return color.Black
	}
	return color.White
}

func (t Turn) Cell() cell.Cell {
	if t == BLACK {
		return cell.Black
	}
	return cell.White
}
