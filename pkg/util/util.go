package util

import "github.com/ArminGh02/othello-bot/pkg/util/coord"

type CoordSet struct {
	m map[coord.Coord]struct{}
}

func NewCoordSet() CoordSet {
	return CoordSet{
		m: make(map[coord.Coord]struct{}),
	}
}

func (set *CoordSet) Clear() {
	for key := range set.m {
		delete(set.m, key)
	}
}

func (set *CoordSet) Insert(c coord.Coord) {
	set.m[c] = struct{}{}
}

func (set *CoordSet) Contains(c coord.Coord) bool {
	_, present := set.m[c]
	return present
}

func (set *CoordSet) IsEmpty() bool {
	return len(set.m) == 0
}
