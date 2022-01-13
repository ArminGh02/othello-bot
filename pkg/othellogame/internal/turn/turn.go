package turn

import (
	"math/rand"
)

type Turn bool

const (
	PLAYER1 = Turn(false)
	PLAYER2 = Turn(true)
)

func Random() Turn {
	return rand.Int31n(2) == 0
}

func (t Turn) Int() int {
	if t {
		return 0
	}
	return 1
}
