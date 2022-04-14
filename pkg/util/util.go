package util

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/ArminGh02/othello-bot/pkg/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func RemoveInlineKeyboardMarkup() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	}
}

func UsernameElseName(user *tgbotapi.User) string {
	username := user.UserName
	if username != "" {
		return "@" + username
	}
	return FullNameOf(user)
}

func FullNameOf(user *tgbotapi.User) string {
	if user.FirstName == "" && user.LastName == "" {
		return user.UserName
	}
	if user.LastName == "" {
		return user.FirstName
	}
	if user.FirstName == "" {
		return user.LastName
	}
	return user.FirstName + " " + user.LastName
}

func FirstNameElseLastName(user *tgbotapi.User) string {
	if user.FirstName == "" {
		return user.LastName
	}
	return user.FirstName
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

	if i == len(s.scoreboard) {
		s.scoreboard = append(s.scoreboard, *player)
	} else {
		s.scoreboard = append(s.scoreboard[:i+1], s.scoreboard[i:]...)
		s.scoreboard[i] = *player
	}

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
	log.Panicln("An attempt was made to retrieve the index of a" +
		" user that was not inserted into scoreboard.")
	panic("")
}

func (s *Scoreboard) RankOf(userID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastScore := math.MinInt
	rank := 0
	for i := range s.scoreboard {
		if score := s.scoreboard[i].Score(); score != lastScore {
			rank++
			lastScore = score
		}
		if s.scoreboard[i].UserID == userID {
			return rank
		}
	}
	log.Panicln("An attempt was made to retrieve the rank of" +
		" a user that was not inserted into scoreboard.")
	panic("")
}

func (s *Scoreboard) String(userID int64) string {
	var sb strings.Builder
	rank := 0
	lastScore := math.MinInt
	emojis := map[int]string{1: "🥇", 2: "🥈", 3: "🥉"}
loop:
	for i := range s.scoreboard {
		if score := s.scoreboard[i].Score(); score != lastScore {
			rank++
			lastScore = score
		}
		switch rank {
		case 1, 2, 3:
			sb.WriteString(fmt.Sprintf(
				"%d. %s %s Score: %d\n",
				rank, s.scoreboard[i].Name, emojis[rank], s.scoreboard[i].Score()))
		default:
			break loop
		}
	}

	userRank := s.RankOf(userID)
	if 1 <= userRank && userRank <= 3 {
		return sb.String()
	}

	sb.WriteString("...\n")

	index := s.indexOf(userID)
	to := index + 1
	if index+1 >= len(s.scoreboard) {
		to--
	}
	if s.scoreboard[index-1].Score() == s.scoreboard[index].Score() {
		rank = userRank
	} else {
		rank = userRank - 1
	}
	lastScore = s.scoreboard[index-1].Score()
	for i := index - 1; ; i++ {
		sb.WriteString(fmt.Sprintf(
			"%d. %s Score: %d\n",
			rank, s.scoreboard[i].Name, s.scoreboard[i].Score()))
		if i >= to {
			break
		}
		if score := s.scoreboard[i+1].Score(); score != lastScore {
			rank++
			lastScore = score
		}
	}

	sb.WriteString("...\n")
	return sb.String()
}
