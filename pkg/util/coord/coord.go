package coord

type Coord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func New(x, y int) Coord {
	return Coord{
		X: x,
		Y: y,
	}
}

func Plus(a, b Coord) Coord {
	return Coord{
		X: a.X + b.X,
		Y: a.Y + b.Y,
	}
}

func (this *Coord) Plus(other Coord) {
	*this = Plus(*this, other)
}
