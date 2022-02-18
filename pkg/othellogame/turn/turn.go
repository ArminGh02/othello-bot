package turn

import (
	"math/rand"

	"github.com/ArminGh02/othello-bot/pkg/othellogame/cell"
	"github.com/ArminGh02/othello-bot/pkg/othellogame/color"
)

type Turn bool

const (
	White = Turn(false)
	Black = Turn(true)
)

func Random() Turn {
	return rand.Int31n(2) == 0
}

func (t Turn) Int() int {
	if t == Black {
		return color.Black
	}
	return color.White
}

func (t Turn) Cell() cell.Cell {
	if t == Black {
		return cell.Black
	}
	return cell.White
}
