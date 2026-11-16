package games

import (
	"context"
	"math/rand/v2"
)

// Round sizes.
const (
	pictureQuestions = 40 // enough for 60 seconds
	letterWords      = 25
	memoryPairs      = 6
	minPictureWords  = 4
	minLetterWords   = 5
)

func hasPicture(w PoolWord) bool { return w.ImageURL != "" }

// PictureRound builds an "Ảnh nhanh tay" round (FR-010, FR-012): 40 pictures of the learner's
// words, each with four different words of which one is right.
func (s *Service) PictureRound(ctx context.Context, userID string) ([]PictureQuestion, error) {
	return s.pictureQuestions(ctx, userID, pictureQuestions, s.newRand())
}

// pictureQuestions builds n picture questions; a word is asked again only when there are fewer
// than n words with a picture.
func (s *Service) pictureQuestions(ctx context.Context, userID string, n int, rnd *rand.Rand) ([]PictureQuestion, error) {
	pool, err := s.wordPool(ctx, userID, n, hasPicture, rnd)
	if err != nil {
		return nil, err
	}
	var pics []PoolWord
	for _, w := range pool {
		if hasPicture(w) {
			pics = append(pics, w)
		}
	}
	if len(pics) < minPictureWords {
		return nil, notEnoughWords()
	}
	out := make([]PictureQuestion, 0, n)
	for i := 0; len(out) < n && i < 2*n; i++ {
		answer := pics[i%len(pics)]
		q, ok := pictureQuestion(answer, pool, rnd)
		if ok {
			out = append(out, q)
		}
	}
	if len(out) == 0 {
		return nil, notEnoughWords()
	}
	return out, nil
}

// pictureQuestion asks answer with three distractors from pool: words of the same lesson first,
// never another form of a word shown, a word with the same meaning or the same picture.
func pictureQuestion(answer PoolWord, pool []PoolWord, rnd *rand.Rand) (PictureQuestion, bool) {
	var sameLesson, others []PoolWord
	for _, w := range pool {
		if answer.LessonID != "" && w.LessonID == answer.LessonID {
			sameLesson = append(sameLesson, w)
		} else {
			others = append(others, w)
		}
	}
	shuffle(rnd, sameLesson)
	shuffle(rnd, others)
	chosen := []PoolWord{answer}
	for _, w := range append(sameLesson, others...) {
		if len(chosen) == 4 {
			break
		}
		if clashes(w, chosen) {
			continue
		}
		chosen = append(chosen, w)
	}
	if len(chosen) < 4 {
		return PictureQuestion{}, false
	}
	shuffle(rnd, chosen)
	q := PictureQuestion{Lemma: answer.Lemma, ImageURL: answer.ImageURL, MeaningVi: answer.MeaningVi}
	for i, w := range chosen {
		q.Options = append(q.Options, w.Lemma)
		if w.Lemma == answer.Lemma {
			q.Answer = i
		}
	}
	return q, true
}

// clashes reports whether w could be taken for one of the chosen words.
func clashes(w PoolWord, chosen []PoolWord) bool {
	for _, c := range chosen {
		if sameForm(w.Lemma, c.Lemma) || sameMeaning(w.MeaningVi, c.MeaningVi) ||
			(w.ImageURL != "" && w.ImageURL == c.ImageURL) {
			return true
		}
	}
	return false
}

// LettersRound builds a "Chữ rơi" round (FR-016): up to 25 one-word lemmas of 3 to 8 letters with
// their meaning, none twice.
func (s *Service) LettersRound(ctx context.Context, userID string) ([]LetterWord, error) {
	return s.letterWords(ctx, userID, letterWords, minLetterWords, s.newRand())
}

func letterReady(w PoolWord) bool { return spellable(w.Lemma) && w.MeaningVi != "" }

func (s *Service) letterWords(ctx context.Context, userID string, n, least int, rnd *rand.Rand) ([]LetterWord, error) {
	pool, err := s.wordPool(ctx, userID, n, letterReady, rnd)
	if err != nil {
		return nil, err
	}
	var out []LetterWord
	for _, w := range pool {
		if len(out) == n {
			break
		}
		if letterReady(w) {
			out = append(out, LetterWord{Lemma: w.Lemma, MeaningVi: w.MeaningVi, ImageURL: w.ImageURL})
		}
	}
	if len(out) < least {
		return nil, notEnoughWords()
	}
	return out, nil
}

// MemoryRound builds a "Lật thẻ" round (FR-019, FR-020): 6 words with their picture, or meaning
// when they have none. With a lesson, the words are that lesson's (pictures first), and the learner
// must be able to open it; without, the learner's words, due ones first.
func (s *Service) MemoryRound(ctx context.Context, userID string, isAdmin bool, lessonID string) ([]MemoryPair, error) {
	rnd := s.newRand()
	var pool []PoolWord
	if lessonID != "" {
		ok, err := s.d.Study.CanOpen(ctx, userID, isAdmin, lessonID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrLessonLocked
		}
		words, err := s.lessonPool(ctx, lessonID, 1)
		if err != nil {
			return nil, err
		}
		if err := s.decorate(ctx, words); err != nil {
			return nil, err
		}
		var withPic, without []PoolWord
		for _, w := range words {
			w.Lemma = normalize(w.Lemma)
			if hasPicture(w) {
				withPic = append(withPic, w)
			} else {
				without = append(without, w)
			}
		}
		shuffle(rnd, withPic)
		shuffle(rnd, without)
		pool = append(withPic, without...)
	} else {
		var err error
		pool, err = s.wordPool(ctx, userID, memoryPairs, memoryReady, rnd)
		if err != nil {
			return nil, err
		}
	}
	var chosen []PoolWord
	for _, w := range pool {
		if len(chosen) == memoryPairs {
			break
		}
		if !memoryReady(w) || clashes(w, chosen) {
			continue
		}
		chosen = append(chosen, w)
	}
	if len(chosen) < memoryPairs {
		return nil, notEnoughWords()
	}
	out := make([]MemoryPair, len(chosen))
	for i, w := range chosen {
		out[i] = MemoryPair{Lemma: w.Lemma, ImageURL: w.ImageURL, MeaningVi: w.MeaningVi}
	}
	return out, nil
}

func memoryReady(w PoolWord) bool { return w.Lemma != "" && (w.ImageURL != "" || w.MeaningVi != "") }
