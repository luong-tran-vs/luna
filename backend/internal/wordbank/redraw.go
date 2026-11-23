package wordbank

import (
	"context"
	"fmt"
	"slices"
	"time"
)

// Redrawing the pictures of whole topics (added 2026-11-23): the admin picks topics and draws all
// their words again, ("Vẽ lại hết"), which starts a new round of each topic; a round cut short (the
// admin stopped it, or the topics had more than MaxBatch words) goes on with "Vẽ tiếp", which draws
// only the words whose picture is older than the round. A word is in a round once its picture was
// saved after the round started, whichever batch drew it.

// BatchMode is what a batch over topics draws.
type BatchMode string

const (
	// BatchMissing draws the words lacking a picture.
	BatchMissing BatchMode = ""
	// BatchRedraw draws every word of the topics again and starts a new round of each.
	BatchRedraw BatchMode = "redraw"
	// BatchContinue draws the words of the topics the latest round has not reached.
	BatchContinue BatchMode = "continue"
)

// redrawWords lists the words a redraw batch draws, by lemma, at most MaxBatch.
func (s *Service) redrawWords(ctx context.Context, req BatchRequest) ([]string, error) {
	if req.Mode != BatchRedraw && req.Mode != BatchContinue {
		return nil, &ValidationError{Fields: map[string]string{"mode": "Chế độ vẽ không hợp lệ"}}
	}
	ids := req.topicIDs()
	if len(ids) == 0 || len(req.Lemmas) > 0 {
		return nil, &ValidationError{Fields: map[string]string{"topicIds": "Hãy chọn ít nhất một chủ đề"}}
	}
	topics, err := s.topics.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("wordbank: topic words: %w", err)
	}
	rounds, err := s.redrawsByTopic(ctx)
	if err != nil {
		return nil, err
	}
	// since is, for each word, the start of the latest round of the chosen topics holding it.
	since := map[string]time.Time{}
	var lemmas []string
	for _, id := range ids {
		t, ok := findTopic(topics, id)
		if !ok {
			return nil, &ValidationError{Fields: map[string]string{"topicId": msgNoTopic}}
		}
		at := rounds[id].LastAt
		for _, l := range lemmasOf(t) {
			prev, seen := since[l]
			if !seen {
				lemmas = append(lemmas, l)
			}
			if !seen || at.After(prev) {
				since[l] = at
			}
		}
	}
	words, err := s.repo.Find(ctx, lemmas)
	if err != nil {
		return nil, fmt.Errorf("wordbank: find topic words: %w", err)
	}
	var out []string
	for _, w := range words {
		if req.Mode == BatchRedraw || !w.HasImage() || w.ImageAt.Before(since[w.Lemma]) {
			out = append(out, w.Lemma)
		}
	}
	slices.Sort(out)
	if len(out) > MaxBatch {
		out = out[:MaxBatch]
	}
	return out, nil
}

func (s *Service) redrawsByTopic(ctx context.Context) (map[string]TopicRedraw, error) {
	list, err := s.repo.TopicRedraws(ctx)
	if err != nil {
		return nil, fmt.Errorf("wordbank: topic redraws: %w", err)
	}
	out := make(map[string]TopicRedraw, len(list))
	for _, r := range list {
		out[r.TopicID] = r
	}
	return out, nil
}

// TopicPictures is how the pictures of one topic's bank words stand: how many words, how many
// lack a picture, and its redraw rounds with how many words the latest one has reached (Fresh).
type TopicPictures struct {
	TopicID string
	Words   int
	Missing int
	Rounds  int
	LastAt  time.Time
	Fresh   int
}

// TopicsPictures returns the pictures of every topic with bank words, in the topics' order.
func (s *Service) TopicsPictures(ctx context.Context) ([]TopicPictures, error) {
	topics, err := s.topics.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("wordbank: topic words: %w", err)
	}
	rounds, err := s.redrawsByTopic(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var lemmas []string
	for _, t := range topics {
		for _, l := range lemmasOf(t) {
			if !seen[l] {
				seen[l] = true
				lemmas = append(lemmas, l)
			}
		}
	}
	words, err := s.repo.Find(ctx, lemmas)
	if err != nil {
		return nil, fmt.Errorf("wordbank: find topic words: %w", err)
	}
	byLemma := make(map[string]Word, len(words))
	for _, w := range words {
		byLemma[w.Lemma] = w
	}
	out := []TopicPictures{}
	for _, t := range topics {
		r := rounds[t.ID]
		p := TopicPictures{TopicID: t.ID, Rounds: r.Rounds, LastAt: r.LastAt}
		counted := map[string]bool{}
		for _, l := range lemmasOf(t) {
			w, ok := byLemma[l]
			if !ok || counted[l] {
				continue
			}
			counted[l] = true
			p.Words++
			switch {
			case !w.HasImage():
				p.Missing++
			case r.Rounds > 0 && !w.ImageAt.Before(r.LastAt):
				p.Fresh++
			}
		}
		if p.Words > 0 {
			out = append(out, p)
		}
	}
	return out, nil
}
