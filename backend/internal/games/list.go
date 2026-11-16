package games

import (
	"context"
	"fmt"
	"time"
)

// GameView is one game of the list with the learner's best.
type GameView struct {
	ID     Game
	Best   int
	BestAt time.Time // zero when there is no best
	Locked bool
}

// DailyStatus is today's challenge as the list and the home page show it.
type DailyStatus struct {
	Day    string
	Done   bool
	Streak int
}

// ListView is the games page.
type ListView struct {
	Games []GameView
	Daily DailyStatus
}

// List returns the released games with the learner's bests (FR-001); "Bắn lỗi" is locked for
// guests (FR-033). It also says whether today's challenge is done and its streak.
func (s *Service) List(ctx context.Context, userID string, isGuest bool) (ListView, error) {
	bests, err := s.d.Bests.List(ctx, userID)
	if err != nil {
		return ListView{}, fmt.Errorf("games: bests: %w", err)
	}
	byGame := make(map[Game]Best, len(bests))
	for _, b := range bests {
		byGame[b.Game] = b
	}
	out := ListView{Games: make([]GameView, 0, len(released))}
	for _, g := range released {
		b := byGame[g]
		out.Games = append(out.Games, GameView{ID: g, Best: b.Score, BestAt: b.At, Locked: g == GameGrammar && isGuest})
	}
	today, results, err := s.dailyHistory(ctx, userID)
	if err != nil {
		return ListView{}, err
	}
	out.Daily = DailyStatus{Day: today, Streak: streakOf(daysOf(results), today).Current}
	out.Daily.Done = len(results) > 0 && results[0].Day == today
	return out, nil
}
