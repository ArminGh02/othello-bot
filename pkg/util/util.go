package util

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/ArminGh02/othello-bot/pkg/database"
	"github.com/ArminGh02/othello-bot/pkg/util/coord"
)

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

type byScore []database.PlayerDoc

func (b byScore) Len() int {
	return len(b)
}

func (b byScore) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byScore) Less(i, j int) bool {
	return b[i].Score() > b[j].Score()
}

type Scoreboard struct {
	scoreboard []database.PlayerDoc
	mu         sync.Mutex
}

func NewScoreboard(players []database.PlayerDoc) Scoreboard {
	sort.Sort(byScore(players))
	return Scoreboard{
		scoreboard: players,
	}
}

func (s *Scoreboard) Insert(player *database.PlayerDoc) {
	s.mu.Lock()

	score := player.Score()
	i := len(s.scoreboard)
	for i-1 >= 0 && score > s.scoreboard[i-1].Score() {
		i--
	}
	s.scoreboard = append(s.scoreboard[:i+1], s.scoreboard[i:]...)
	s.scoreboard[i] = *player

	s.mu.Unlock()
}

func (s *Scoreboard) UpdateRankOf(userID int64, winsDelta, lossesDelta int) {
	s.mu.Lock()

	i := s.indexOf(userID)
	player := &s.scoreboard[i]

	player.Wins += winsDelta
	player.Losses += lossesDelta

	score := player.Score()

	for ; i-1 >= 0 && score > s.scoreboard[i-1].Score(); i-- {
		s.scoreboard[i], s.scoreboard[i-1] = s.scoreboard[i-1], s.scoreboard[i]
	}

	for ; i+1 < len(s.scoreboard) && score < s.scoreboard[i+1].Score(); i++ {
		s.scoreboard[i], s.scoreboard[i+1] = s.scoreboard[i+1], s.scoreboard[i]
	}

	s.mu.Unlock()
}

func (s *Scoreboard) indexOf(userID int64) int {
	for i := range s.scoreboard {
		if s.scoreboard[i].UserID == userID {
			return i
		}
	}
	panic("An attempt was made to retrieve the index of a user that was not inserted into scoreboard.")
}

func (s *Scoreboard) RankOf(userID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastScore := math.MinInt
	rank := 1
	for i := range s.scoreboard {
		if s.scoreboard[i].UserID == userID {
			return rank
		}
		if score := s.scoreboard[i].Score(); score != lastScore {
			rank++
			lastScore = score
		}
	}
	panic("An attempt was made to retrieve the rank of a user that was not inserted into scoreboard.")
}
