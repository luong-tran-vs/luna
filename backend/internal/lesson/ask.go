package lesson

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// askTimeout bounds one AI explanation so the request stays under the server write timeout.
const askTimeout = 12 * time.Second

// Limits of an explanation stored for everyone.
const (
	maxAskMeaning = 200
	maxAskNote    = 300
)

// ErrUnusableExplanation means the AI answer had no meaning.
var ErrUnusableExplanation = errors.New("lesson: AI returned no meaning")

// errAskAI wraps a failed AI call of Ask, so the handler can tell it from a storage failure.
var errAskAI = errors.New("lesson: explain")

// AskKey names one asked word or phrase: a lesson revision, a sentence and the normalized text.
type AskKey struct {
	LessonID      string
	Revision      int
	SentenceIndex int
	Text          string
}

// AskResult is an AI explanation stored for every learner (F9).
type AskResult struct {
	AskKey
	Lemma     string
	MeaningVi string
	NoteVi    string
	CreatedAt time.Time
}

// AskRepository stores the explanations; implementations live in internal/storage.
type AskRepository interface {
	// Get returns the stored result for key; ok is false when there is none.
	Get(ctx context.Context, key AskKey) (r AskResult, ok bool, err error)
	// Put stores r; when the key is already stored it returns the stored result instead.
	Put(ctx context.Context, r AskResult) (AskResult, error)
}

// Ask explains a word or phrase of a lesson sentence with the AI (F9). A stored explanation for
// the same lesson revision, sentence and text is returned without calling the AI (cached);
// concurrent asks for one key share a single AI call. Failed answers are never stored.
func (r *Reader) Ask(ctx context.Context, lessonID, q string, sentence int) (LookupResult, bool, error) {
	l, err := r.lessons.Get(ctx, lessonID)
	if err != nil {
		return LookupResult{}, false, err
	}
	text := normalize(q)
	if err := checkAsk(l, text, sentence); err != nil {
		return LookupResult{}, false, err
	}
	key := AskKey{LessonID: l.ID, Revision: l.Revision, SentenceIndex: sentence, Text: text}
	if stored, ok, err := r.asks.Get(ctx, key); err != nil {
		return LookupResult{}, false, fmt.Errorf("lesson: stored explanation: %w", err)
	} else if ok {
		return r.askResult(ctx, stored), true, nil
	}

	flightKey := fmt.Sprintf("%s|%d|%d|%s", key.LessonID, key.Revision, key.SentenceIndex, key.Text)
	v, err, _ := r.group.Do(flightKey, func() (any, error) {
		// The learner may close the popup: finish and store the answer for next time anyway.
		actx, cancel := context.WithTimeout(context.WithoutCancel(ctx), askTimeout)
		defer cancel()
		if stored, ok, err := r.asks.Get(actx, key); err != nil {
			return nil, fmt.Errorf("lesson: stored explanation: %w", err)
		} else if ok {
			return stored, nil
		}
		ex, err := r.ai.Explain(actx, ai.ExplainRequest{Text: text, Sentence: l.Sentences[sentence].Text, Level: string(l.Level)})
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errAskAI, err)
		}
		res := AskResult{
			AskKey: key, Lemma: normalize(ex.Lemma), MeaningVi: cutText(ex.MeaningVi, maxAskMeaning),
			NoteVi: cutText(ex.NoteVi, maxAskNote), CreatedAt: time.Now().UTC(),
		}
		if res.MeaningVi == "" {
			return nil, ErrUnusableExplanation
		}
		if res.Lemma == "" {
			res.Lemma = text
		}
		stored, err := r.asks.Put(actx, res)
		if err != nil {
			return nil, fmt.Errorf("lesson: store explanation: %w", err)
		}
		return stored, nil
	})
	if err != nil {
		return LookupResult{}, false, err
	}
	return r.askResult(ctx, v.(AskResult)), false, nil
}

// checkAsk applies the lookup limits (F3) and requires the text to be in the sentence.
func checkAsk(l Lesson, text string, sentence int) error {
	fields := map[string]string{}
	switch {
	case text == "":
		fields["text"] = "Chưa chọn từ để hỏi"
	case utf8.RuneCountInString(text) > maxLookupChars:
		fields["text"] = "Tối đa 100 ký tự"
	case len(strings.Fields(text)) > maxLookupWords:
		fields["text"] = "Chọn tối đa 6 từ"
	}
	if sentence < 0 || sentence >= len(l.Sentences) {
		fields["sentenceIndex"] = "Câu không hợp lệ"
	} else if _, bad := fields["text"]; !bad && !strings.Contains(normalize(l.Sentences[sentence].Text), text) {
		fields["text"] = "Từ không có trong câu này"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// askResult shows a stored explanation like a lookup from the AI, with the IPA of its lemma.
func (r *Reader) askResult(ctx context.Context, a AskResult) LookupResult {
	res := LookupResult{
		Source: SourceAI, Text: a.Text, Lemma: a.Lemma,
		Meanings: []LookupMeaning{{Text: a.MeaningVi}}, Note: a.NoteVi,
	}
	if e, ok, err := r.dict.Resolve(ctx, a.Lemma); err == nil && ok {
		res.IPA = e.IPA
	}
	return res
}

func cutText(s string, limit int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > limit {
		s = strings.TrimSpace(string([]rune(s)[:limit]))
	}
	return s
}
