package games

import (
	"context"
	"fmt"
	"strings"
)

// Result is what the page sends at the end of a round. It is not stored: it raises the best and
// brings the missed words and verbs back to the review queue.
type Result struct {
	Score      int
	Completed  bool
	WrongWords []string
	WrongVerbs []VerbMiss
}

// FinishReply is the best after a round and how many cards were brought forward.
type FinishReply struct {
	Best        int
	NewBest     bool
	Rescheduled int
}

// checkMisses validates the missed words and verbs of a result.
func checkMisses(words []string, verbs []VerbMiss) error {
	if len(words) > MaxWrongWords {
		return invalid("wrongWords", fmt.Sprintf("Tối đa %d từ mỗi lượt", MaxWrongWords))
	}
	if len(verbs) > MaxWrongVerbs {
		return invalid("wrongVerbs", fmt.Sprintf("Tối đa %d động từ mỗi lượt", MaxWrongVerbs))
	}
	for _, v := range verbs {
		if v.Form != "past" && v.Form != "participle" {
			return invalid("wrongVerbs", "Dạng động từ không hợp lệ")
		}
		if strings.TrimSpace(v.Base) == "" {
			return invalid("wrongVerbs", "Thiếu động từ")
		}
	}
	return nil
}

func checkScore(game Game, score int) error {
	if score < 0 || score > MaxScore[game] {
		return invalid("score", fmt.Sprintf("Điểm phải từ 0 đến %d", MaxScore[game]))
	}
	return nil
}

// Finish takes the result of a round of game (not the daily challenge): a completed round raises
// the best; missed words and verbs come back to the review queue either way (FR-003, FR-004).
func (s *Service) Finish(ctx context.Context, userID string, isGuest bool, game Game, r Result) (FinishReply, error) {
	if game == GameDaily || !Released(game) {
		return FinishReply{}, invalid("game", "Trò chơi không hợp lệ")
	}
	if game == GameGrammar && isGuest {
		return FinishReply{}, ErrMembersOnly
	}
	if err := checkScore(game, r.Score); err != nil {
		return FinishReply{}, err
	}
	if err := checkMisses(r.WrongWords, r.WrongVerbs); err != nil {
		return FinishReply{}, err
	}
	var reply FinishReply
	// A round with no point is no best, even the first one.
	if r.Completed && r.Score > 0 {
		improved, err := s.d.Bests.Raise(ctx, userID, game, r.Score, s.d.Now().UTC())
		if err != nil {
			return FinishReply{}, fmt.Errorf("games: raise best: %w", err)
		}
		reply.NewBest = improved
	}
	best, err := s.best(ctx, userID, game)
	if err != nil {
		return FinishReply{}, err
	}
	reply.Best = best
	if reply.Rescheduled, err = s.bringBack(ctx, userID, r.WrongWords, r.WrongVerbs); err != nil {
		return FinishReply{}, err
	}
	return reply, nil
}

// best is the user's best score of game, 0 when none.
func (s *Service) best(ctx context.Context, userID string, game Game) (int, error) {
	list, err := s.d.Bests.List(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("games: bests: %w", err)
	}
	for _, b := range list {
		if b.Game == game {
			return b.Score, nil
		}
	}
	return 0, nil
}

// bringBack brings missed words and verbs back to the review queue; it returns the cards moved.
func (s *Service) bringBack(ctx context.Context, userID string, words []string, verbs []VerbMiss) (int, error) {
	moved := 0
	if len(words) > 0 {
		n, err := s.d.Words.GameMisses(ctx, userID, words)
		if err != nil {
			return 0, fmt.Errorf("games: word misses: %w", err)
		}
		moved = n
	}
	if len(verbs) > 0 {
		clean := make([]VerbMiss, 0, len(verbs))
		for _, v := range verbs {
			v.Base = strings.ToLower(strings.TrimSpace(v.Base))
			clean = append(clean, v)
		}
		if err := s.d.Verbs.GameMisses(ctx, userID, clean); err != nil {
			return 0, fmt.Errorf("games: verb misses: %w", err)
		}
	}
	return moved, nil
}
