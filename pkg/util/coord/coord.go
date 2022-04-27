package coord

type Coord struct {
	X int
	Y int
}

func New(x, y int) Coord {
	return Coord{
		X: x,
		Y: y,
	}
}

func Plus(a, b Coord) Coord {
	a.Plus(b)
	return a
}

func (c *Coord) Plus(other Coord) {
	c.X += other.X
	c.Y += other.Y
}
