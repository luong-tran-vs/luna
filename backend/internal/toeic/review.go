package toeic

import (
	"context"
	"math"
	"slices"
	"time"
)

// ResultView is the result page of a submitted attempt.
type ResultView struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Mode        Mode          `json:"mode"`
	Source      Source        `json:"source"`
	Parts       []Part        `json:"parts"`
	Retake      bool          `json:"retake"`
	Counted     bool          `json:"counted"`
	Correct     int           `json:"correct"`
	Wrong       int           `json:"wrong"`
	Skipped     int           `json:"skipped"`
	Questions   int           `json:"questions"`
	Accuracy    float64       `json:"accuracy"`
	DurationMs  int64         `json:"durationMs"`
	Listening   *SectionScore `json:"listening"`
	Reading     *SectionScore `json:"reading"`
	Total       *int          `json:"total"`
	Goal        *int          `json:"goal"`
	ByPart      []PartTags    `json:"byPart"`
	Answers     []AnswerRow   `json:"answers"`
	SubmittedAt time.Time     `json:"submittedAt"`
}

// PartTags is the "question types" table of one part.
type PartTags struct {
	Part Part     `json:"part"`
	Tags []TagRow `json:"tags"`
}

// TagRow is one question type of a part in the result.
type TagRow struct {
	Tag       string      `json:"tag"`
	LabelVi   string      `json:"labelVi"`
	Right     int         `json:"right"`
	Wrong     int         `json:"wrong"`
	Skipped   int         `json:"skipped"`
	Questions []NumberRef `json:"questions"`
}

// NumberRef is a question number with its outcome.
type NumberRef struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
	State  string `json:"state"`
}

// AnswerRow is a line of "right answer : your answer".
type AnswerRow struct {
	Number  int    `json:"number"`
	ID      string `json:"id"`
	Part    Part   `json:"part"`
	Correct string `json:"correct"`
	Chosen  string `json:"chosen"`
	State   string `json:"state"`
}

// letter is the option letter of an index, "" for no choice.
func letter(i int) string {
	if i < 0 || i > 25 {
		return ""
	}
	return string(rune('A' + i))
}

// accuracy is right of total as a percentage with one decimal.
func accuracy(right, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(right)*1000/float64(total)) / 10
}

// Result returns the result page of a submitted attempt; ErrAttemptInProgress before.
func (s *Service) Result(ctx context.Context, userID, id string) (ResultView, error) {
	a, err := s.d.Attempts.Get(ctx, userID, id)
	if err != nil {
		return ResultView{}, err
	}
	if a.Status != AttemptSubmitted || a.Result == nil {
		return ResultView{}, &ErrAttemptInProgress{ID: a.ID}
	}
	_, byKey, err := s.attemptGroups(ctx, a)
	if err != nil {
		return ResultView{}, err
	}
	r := a.Result
	v := ResultView{
		ID: a.ID, Title: a.Title, Mode: a.Mode, Source: a.Source, Parts: a.Parts, Counted: a.Counted,
		Correct: r.Correct, Wrong: r.Wrong, Skipped: r.Skipped, Questions: len(a.QuestionIDs),
		Accuracy: accuracy(r.Correct, len(a.QuestionIDs)), DurationMs: r.DurationMs, Listening: r.Listening, Reading: r.Reading,
		ByPart: []PartTags{}, Answers: []AnswerRow{}, SubmittedAt: a.SubmittedAt,
	}
	if r.Total > 0 {
		t := r.Total
		v.Total = &t
	}
	if v.Retake, err = s.retake(ctx, a); err != nil {
		return ResultView{}, err
	}
	if g, err := s.d.Goals.Get(ctx, userID); err == nil {
		t := g.Target
		v.Goal = &t
	} else if !errIsNotFound(err) {
		return ResultView{}, err
	}

	parts := map[Part]*PartTags{}
	for _, id := range a.QuestionIDs {
		key, no, ok := SplitQuestionID(id)
		g, found := byKey[key]
		if !ok || !found {
			continue
		}
		q, found := g.Question(no)
		if !found {
			continue
		}
		state := r.ByQuestion[id]
		if state == "" {
			state = OutcomeSkipped
		}
		num := a.Numbers[id]
		chosen := -1
		if c, ok := a.Answers[id]; ok {
			chosen = c
		}
		v.Answers = append(v.Answers, AnswerRow{Number: num, ID: id, Part: g.Part, Correct: letter(q.Answer), Chosen: letter(chosen), State: state})
		pt := parts[g.Part]
		if pt == nil {
			pt = &PartTags{Part: g.Part}
			parts[g.Part] = pt
		}
		for _, tag := range q.Tags {
			i := slices.IndexFunc(pt.Tags, func(t TagRow) bool { return t.Tag == tag })
			if i < 0 {
				pt.Tags = append(pt.Tags, TagRow{Tag: tag, LabelVi: labelOf(tag), Questions: []NumberRef{}})
				i = len(pt.Tags) - 1
			}
			row := &pt.Tags[i]
			switch state {
			case OutcomeRight:
				row.Right++
			case OutcomeWrong:
				row.Wrong++
			default:
				row.Skipped++
			}
			row.Questions = append(row.Questions, NumberRef{Number: num, ID: id, State: state})
		}
	}
	for p := Part(1); p <= 7; p++ {
		if pt := parts[p]; pt != nil {
			v.ByPart = append(v.ByPart, *pt)
		}
	}
	return v, nil
}

// retake reports whether the learner submitted the same test before this attempt.
func (s *Service) retake(ctx context.Context, a Attempt) (bool, error) {
	if a.Source.Kind != SourceTest {
		return false, nil
	}
	list, err := s.d.Attempts.List(ctx, a.UserID, AttemptFilter{TestKey: a.Source.TestKey, Status: AttemptSubmitted})
	if err != nil {
		return false, err
	}
	// Newest first: an older sitting comes after this one.
	i := slices.IndexFunc(list, func(o Attempt) bool { return o.ID == a.ID })
	return i >= 0 && i < len(list)-1, nil
}

// --- answer detail ---

// GroupDetail is a whole group with its translations, for the answer detail.
type GroupDetail struct {
	Key      string      `json:"key"`
	Part     Part        `json:"part"`
	ImageURL *string     `json:"imageUrl"`
	Docs     []DocDetail `json:"docs,omitempty"`
	Script   []Turn      `json:"script,omitempty"`
	Graphic  *Graphic    `json:"graphic"`
}

// DocDetail is a text with its translation.
type DocDetail struct {
	PaperDoc
	TranslateVi string `json:"translateVi,omitempty"`
}

// TagRef is a tag of a question with its label and grammar point.
type TagRef struct {
	Tag            string `json:"tag"`
	LabelVi        string `json:"labelVi"`
	GrammarPointID string `json:"grammarPointId,omitempty"`
}

// QuestionAnswer is a question with its answer and explanation.
type QuestionAnswer struct {
	ID          string   `json:"id"`
	No          int      `json:"no"`
	Number      int      `json:"number"`
	Prompt      string   `json:"prompt,omitempty"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Chosen      int      `json:"chosen"`
	State       string   `json:"state"`
	Evidence    []Ref    `json:"evidence"`
	ExplainVi   string   `json:"explainVi"`
	TranslateVi string   `json:"translateVi,omitempty"`
	Tags        []TagRef `json:"tags"`
	Vocab       []Vocab  `json:"vocab"`
}

// QuestionDetail is the answer detail page of one question.
type QuestionDetail struct {
	AttemptID string         `json:"attemptId"`
	Title     string         `json:"title"`
	Group     GroupDetail    `json:"group"`
	Question  QuestionAnswer `json:"question"`
	Prev      string         `json:"prev,omitempty"`
	Next      string         `json:"next,omitempty"`
}

// QuestionDetail returns a question of a submitted attempt with its answer.
func (s *Service) QuestionDetail(ctx context.Context, userID, id, qid string) (QuestionDetail, error) {
	a, err := s.d.Attempts.Get(ctx, userID, id)
	if err != nil {
		return QuestionDetail{}, err
	}
	if a.Status != AttemptSubmitted || a.Result == nil {
		return QuestionDetail{}, &ErrAttemptInProgress{ID: a.ID}
	}
	i := slices.Index(a.QuestionIDs, qid)
	if i < 0 {
		return QuestionDetail{}, ErrNotFound
	}
	key, no, _ := SplitQuestionID(qid)
	g, err := s.GetGroup(ctx, key)
	if err != nil {
		return QuestionDetail{}, err
	}
	q, ok := g.Question(no)
	if !ok {
		return QuestionDetail{}, ErrNotFound
	}
	slots, err := s.d.Images.Has(ctx, []string{key})
	if err != nil {
		return QuestionDetail{}, err
	}
	chosen := -1
	if c, ok := a.Answers[qid]; ok {
		chosen = c
	}
	out := QuestionDetail{
		AttemptID: a.ID, Title: a.Title, Group: groupDetailOf(g, slots[key]),
		Question: answerOf(q, qid, a.Numbers[qid], chosen),
	}
	out.Question.State = a.Result.ByQuestion[qid]
	if i > 0 {
		out.Prev = a.QuestionIDs[i-1]
	}
	if i+1 < len(a.QuestionIDs) {
		out.Next = a.QuestionIDs[i+1]
	}
	return out, nil
}

// answerOf is a question with everything a learner may see after answering.
func answerOf(q Question, id string, number, chosen int) QuestionAnswer {
	qa := QuestionAnswer{
		ID: id, No: q.No, Number: number, Prompt: q.Prompt, Options: q.Options, Answer: q.Answer, Chosen: chosen,
		Evidence: q.Evidence, ExplainVi: q.ExplainVi, TranslateVi: q.TranslateVi, Tags: []TagRef{}, Vocab: q.Vocab,
	}
	if qa.Evidence == nil {
		qa.Evidence = []Ref{}
	}
	if qa.Vocab == nil {
		qa.Vocab = []Vocab{}
	}
	for _, t := range q.Tags {
		qa.Tags = append(qa.Tags, TagRef{Tag: t, LabelVi: labelOf(t), GrammarPointID: grammarPointOf(t)})
	}
	return qa
}

// groupDetailOf is the whole group with its translations; slots are its pictures.
func groupDetailOf(g Group, slots []string) GroupDetail {
	pg := paperGroupOf(g, slots)
	d := GroupDetail{Key: g.Key, Part: g.Part, ImageURL: pg.ImageURL, Graphic: g.Graphic, Script: g.Script}
	for i, pd := range pg.Docs {
		d.Docs = append(d.Docs, DocDetail{PaperDoc: pd, TranslateVi: g.Docs[i].TranslateVi})
	}
	return d
}
