package lesson

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func isErr(err, target error) bool { return errors.Is(err, target) }

func asValidation(err error, out **ValidationError) bool { return errors.As(err, out) }

func TestRemapFlags(t *testing.T) {
	t.Parallel()
	q := func(i int, confirmed bool) Flag {
		return Flag{Area: AreaQuestion, Index: i, Kind: FlagMismatch, NoteVi: "n", Confirmed: confirmed}
	}
	other := Flag{Area: AreaSentence, Index: 4, Kind: FlagWrong}
	cases := []struct {
		name     string
		old, new []string
		flags    []Flag
		want     []Flag
	}{
		{"empty", nil, nil, nil, []Flag{}},
		{"no flags", []string{"a", "b"}, []string{"b"}, nil, []Flag{}},
		{"delete in the middle shifts later flags", []string{"a", "b", "c"}, []string{"a", "c"},
			[]Flag{q(0, false), q(1, false), q(2, true)}, []Flag{q(0, false), q(1, true)}},
		{"edit one item drops its flag", []string{"a", "b", "c"}, []string{"a", "B", "c"},
			[]Flag{q(0, false), q(1, false), q(2, false)}, []Flag{q(0, false), q(2, false)}},
		{"add an item at the front", []string{"a", "b"}, []string{"x", "a", "b"},
			[]Flag{q(1, true)}, []Flag{q(2, true)}},
		{"duplicates pair in order", []string{"a", "a"}, []string{"a"},
			[]Flag{q(0, false), q(1, false)}, []Flag{q(0, false)}},
		{"duplicates move together", []string{"a", "a"}, []string{"x", "a", "a"},
			[]Flag{q(1, true)}, []Flag{q(2, true)}},
		{"other areas are untouched", []string{"a"}, nil,
			[]Flag{other, q(0, false), {Area: AreaTranslation, Index: 0}}, []Flag{other, {Area: AreaTranslation, Index: 0}}},
		{"out of range index is dropped", []string{"a"}, []string{"a"}, []Flag{q(3, false), q(-1, false)}, []Flag{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := remapFlags(AreaQuestion, tc.old, tc.new, tc.flags)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// reviewed returns a lesson with three questions and a verified review whose flags sit on a sentence,
// two annotations, the three questions and the translation.
func (e *env) reviewed(t *testing.T) Lesson {
	t.Helper()
	l := e.checkable(t)
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if _, err := e.lessons.update(l.ID, func(l *Lesson) bool {
		l.Extras.Questions = append(l.Extras.Questions,
			Question{Prompt: "Who gave up?", Options: []string{"He", "She", "We", "They"}, AnswerIndex: 0, ExplanationVi: "Câu 2"})
		l.Review = &Review{CheckedAt: checked, VerifiedAt: checked.Add(time.Hour), Flags: []Flag{
			{Area: AreaSentence, Index: 1, Kind: FlagWrong, NoteVi: "s"},
			{Area: AreaAnnotation, Index: 0, Kind: FlagWrong, NoteVi: "a0"},
			{Area: AreaAnnotation, Index: 2, Kind: FlagWrong, NoteVi: "a2", Confirmed: true},
			{Area: AreaQuestion, Index: 0, Kind: FlagMismatch, NoteVi: "q0"},
			{Area: AreaQuestion, Index: 1, Kind: FlagAmbiguous, NoteVi: "q1"},
			{Area: AreaQuestion, Index: 2, Kind: FlagUnchecked, NoteVi: "q2", Confirmed: true},
			{Area: AreaTranslation, Index: 0, Kind: FlagWrong, NoteVi: "t0"},
		}}
		return true
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := e.lessons.Get(t.Context(), l.ID)
	return got
}

func extrasInputOf(l Lesson, keep ...int) ExtrasInput {
	in := ExtrasInput{WritingPrompt: l.Extras.WritingPrompt}
	for _, i := range keep {
		in.Questions = append(in.Questions, l.Extras.Questions[i])
	}
	return in
}

func flagAt(r *Review, area FlagArea, index int) *Flag {
	if r == nil {
		return nil
	}
	for i := range r.Flags {
		if r.Flags[i].Area == area && r.Flags[i].Index == index {
			return &r.Flags[i]
		}
	}
	return nil
}

func TestUpdateExtrasKeepsFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)

	// Delete question 0, edit question 1: question 2 moves to index 1 and keeps its confirmation.
	in := extrasInputOf(l, 1, 2)
	in.Questions[0].Prompt = "What was the weather like?"
	got, err := e.svc.UpdateExtras(t.Context(), l.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	r := got.Review
	if r == nil || !r.CheckedAt.Equal(l.Review.CheckedAt) || !r.VerifiedAt.IsZero() {
		t.Fatalf("review = %+v", r)
	}
	if len(r.Flags) != 5 {
		t.Fatalf("flags = %+v", r.Flags)
	}
	for _, a := range []struct {
		area  FlagArea
		index int
	}{{AreaSentence, 1}, {AreaAnnotation, 0}, {AreaAnnotation, 2}, {AreaTranslation, 0}} {
		if flagAt(r, a.area, a.index) == nil {
			t.Errorf("flag %v %d lost", a.area, a.index)
		}
	}
	if f := flagAt(r, AreaQuestion, 1); f == nil || f.NoteVi != "q2" || !f.Confirmed {
		t.Errorf("moved question flag = %+v", f)
	}
	if flagAt(r, AreaQuestion, 0) != nil || flagAt(r, AreaQuestion, 2) != nil {
		t.Errorf("flags of edited or removed questions kept: %+v", r.Flags)
	}
	if f := flagAt(r, AreaAnnotation, 2); f == nil || !f.Confirmed {
		t.Errorf("annotation flag = %+v", f)
	}
}

func TestUpdateExtrasWithoutQuestionChangeKeepsQuestionFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)
	in := extrasInputOf(l, 0, 1, 2)
	in.WritingPrompt = "Write about your day."
	got, err := e.svc.UpdateExtras(t.Context(), l.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Review.Flags) != len(l.Review.Flags) || !got.Review.VerifiedAt.IsZero() {
		t.Fatalf("review = %+v", got.Review)
	}
}

func TestUpdateAnnotationsKeepsFlags(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)
	var items []AnnotationInput
	for i, a := range l.Annotations {
		if i == 0 { // removed
			continue
		}
		items = append(items, AnnotationInput{Text: a.Text, Lemma: a.Lemma, MeaningVi: a.MeaningVi})
	}
	got, err := e.svc.UpdateAnnotations(t.Context(), l.ID, items)
	if err != nil {
		t.Fatal(err)
	}
	r := got.Review
	if r == nil || !r.VerifiedAt.IsZero() || !r.CheckedAt.Equal(l.Review.CheckedAt) {
		t.Fatalf("review = %+v", r)
	}
	if flagAt(r, AreaAnnotation, 0) != nil && flagAt(r, AreaAnnotation, 0).NoteVi == "a0" {
		t.Errorf("flag of the removed annotation kept")
	}
	if f := flagAt(r, AreaAnnotation, 1); f == nil || f.NoteVi != "a2" || !f.Confirmed {
		t.Errorf("moved annotation flag = %+v", f)
	}
	if len(r.Flags) != 6 {
		t.Errorf("flags = %+v", r.Flags)
	}

	// Editing the meaning of the flagged annotation drops its flag.
	items[1].MeaningVi = "nắng đẹp"
	got, err = e.svc.UpdateAnnotations(t.Context(), l.ID, items)
	if err != nil {
		t.Fatal(err)
	}
	if flagAt(got.Review, AreaAnnotation, 1) != nil || len(got.Review.Flags) != 5 {
		t.Errorf("flags = %+v", got.Review.Flags)
	}
}

func TestUpdateWithoutReviewStaysWithout(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.checkable(t)
	got, err := e.svc.UpdateExtras(t.Context(), l.ID, extrasInputOf(l, 0))
	if err != nil || got.Review != nil {
		t.Fatalf("review = %+v, err %v", got.Review, err)
	}
}

func TestKeepReviewFailureDoesNotFailTheEdit(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)
	e.lessons.failSaveReview = true
	got, err := e.svc.UpdateExtras(t.Context(), l.ID, extrasInputOf(l, 0, 1))
	if err != nil || len(got.Extras.Questions) != 2 || got.Review != nil {
		t.Fatalf("got %+v, err %v", got.Review, err)
	}
}

func TestContentUpdateStillDropsReview(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)
	if _, err := e.svc.Update(t.Context(), l.ID, Input{Title: "Park", Content: "A new text.", TopicID: "topic-b1", Source: "s", License: "l"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.lessons.Get(t.Context(), l.ID); got.Review != nil {
		t.Fatalf("review kept: %+v", got.Review)
	}
}

// --- translations ---

func tr(vi, en string, d ...string) TranslationInput {
	return TranslationInput{Vi: vi, En: en, Distractors: d}
}

func TestUpdateTranslations(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.reviewed(t)
	before := l.PracticeVersion

	got, err := e.svc.UpdateTranslations(t.Context(), l.ID, []TranslationInput{
		tr("Tôi đi công viên.", "I go to the park.", "run", "walk"), // unchanged: keeps its flag
		tr("Tôi bỏ cuộc.", "I never give up.", "stop"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Practice.Translations) != 2 || got.Practice.Translations[1].En != "I never give up." ||
		got.PracticeVersion != before+1 || got.PracticeStatus != StatusDone || got.Practice.Dialogue == nil {
		t.Fatalf("practice = %+v", got.Practice)
	}
	if f := flagAt(got.Review, AreaTranslation, 0); f == nil || f.NoteVi != "t0" {
		t.Errorf("translation flag = %+v", f)
	}
	if got.Review == nil || !got.Review.VerifiedAt.IsZero() || len(got.Review.Flags) != len(l.Review.Flags) {
		t.Errorf("review = %+v", got.Review)
	}

	// Editing the first sentence drops its flag.
	got, err = e.svc.UpdateTranslations(t.Context(), l.ID, []TranslationInput{tr("Tôi đi tới công viên.", "I go to the park.")})
	if err != nil {
		t.Fatal(err)
	}
	if flagAt(got.Review, AreaTranslation, 0) != nil {
		t.Errorf("flag of the edited translation kept")
	}

	// An empty list removes them all and keeps the rest of the practice.
	got, err = e.svc.UpdateTranslations(t.Context(), l.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Practice.Translations) != 0 || len(got.Practice.Examples) == 0 || got.PracticeStatus != StatusDone {
		t.Fatalf("practice = %+v", got.Practice)
	}
}

func TestUpdateTranslationsEmptyKeepsPracticeWithoutOtherParts(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	if _, err := e.lessons.update(l.ID, func(l *Lesson) bool {
		p := *l.Practice
		p.Examples, p.Dialogue = nil, nil
		l.Practice = &p
		return true
	}); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.UpdateTranslations(t.Context(), l.ID, []TranslationInput{})
	if err != nil || got.Practice == nil || len(got.Practice.Translations) != 0 {
		t.Fatalf("got %+v, err %v", got.Practice, err)
	}
}

func TestUpdateTranslationsErrors(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	long := strings.Repeat("a", 201)
	cases := map[string]struct {
		in   []TranslationInput
		keys []string
	}{
		"empty":                  {[]TranslationInput{tr("", "")}, []string{"translations.0.vi", "translations.0.en"}},
		"one tile":               {[]TranslationInput{tr("Đi", "Go")}, []string{"translations.0.en"}},
		"sixteen tiles":          {[]TranslationInput{tr("Đi", strings.TrimSpace(strings.Repeat("go ", 16)))}, []string{"translations.0.en"}},
		"too long":               {[]TranslationInput{tr(long, "I go home")}, []string{"translations.0.vi"}},
		"duplicate":              {[]TranslationInput{tr("A", "I go home"), tr("B", "i  go home")}, []string{"translations.1.en"}},
		"distractor equals tile": {[]TranslationInput{tr("A", "I go home", "Home")}, []string{"translations.0.distractors"}},
		"duplicate distractor":   {[]TranslationInput{tr("A", "I go home", "run", "RUN")}, []string{"translations.0.distractors"}},
		"empty distractor":       {[]TranslationInput{tr("A", "I go home", " ")}, []string{"translations.0.distractors"}},
		"five distractors":       {[]TranslationInput{tr("A", "I go home", "a", "b", "c", "d", "e")}, []string{"translations.0.distractors"}},
		"six sentences": {[]TranslationInput{
			tr("1", "I go 1"), tr("2", "I go 2"), tr("3", "I go 3"), tr("4", "I go 4"), tr("5", "I go 5"), tr("6", "I go 6"),
		}, []string{"translations"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := e.svc.UpdateTranslations(t.Context(), l.ID, tc.in)
			var verr *ValidationError
			if !asValidation(err, &verr) {
				t.Fatalf("err = %v", err)
			}
			for _, k := range tc.keys {
				if verr.Fields[k] == "" {
					t.Errorf("fields = %v, want key %s", verr.Fields, k)
				}
			}
		})
	}
	if got, _ := e.lessons.Get(t.Context(), l.ID); len(got.Practice.Translations) != 1 {
		t.Errorf("a failed edit changed the practice")
	}
}

func TestUpdateTranslationsStates(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	if _, err := e.svc.UpdateTranslations(t.Context(), "missing", nil); !isErr(err, ErrNotFound) {
		t.Errorf("missing lesson: %v", err)
	}
	none := e.annotated(t)
	if _, err := e.lessons.update(none.ID, func(l *Lesson) bool { l.PracticeStatus, l.Practice = StatusNone, nil; return true }); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.UpdateTranslations(t.Context(), none.ID, nil); !isErr(err, ErrNoPractice) {
		t.Errorf("no practice: %v", err)
	}
	l := e.withPractice(t)
	if _, err := e.lessons.update(l.ID, func(l *Lesson) bool { l.PracticeStatus = StatusRunning; return true }); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.UpdateTranslations(t.Context(), l.ID, nil); !isErr(err, ErrPracticeRunning) {
		t.Errorf("running: %v", err)
	}
}

func TestUpdateTranslationsConflict(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	l := e.withPractice(t)
	e.lessons.beforeSavePractice = func() {
		_, _ = e.lessons.update(l.ID, func(l *Lesson) bool { l.PracticeVersion++; return true })
	}
	if _, err := e.svc.UpdateTranslations(t.Context(), l.ID, nil); !isErr(err, ErrPracticeChanged) {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateTranslationsHandler(t *testing.T) {
	t.Parallel()
	a := newAPI(t)
	l := a.env.withPractice(t)
	path := "/api/admin/lessons/" + l.ID + "/practice/translations"

	rec := a.do(t, http.MethodPut, path, "admin", `{"translations":[{"vi":"Tôi đi.","en":"I go home","distractors":["run"]}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ok: %d %s", rec.Code, rec.Body)
	}
	p := decodeBody(t, rec)["lesson"].(map[string]any)["practice"].(map[string]any)
	ts := p["translations"].([]any)
	if len(ts) != 1 || ts[0].(map[string]any)["en"] != "I go home" {
		t.Fatalf("translations = %v", ts)
	}

	rec = a.do(t, http.MethodPut, path, "admin", `{"translations":[{"vi":"","en":"Go","distractors":[]}]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid: %d %s", rec.Code, rec.Body)
	}
	fields, _ := decodeBody(t, rec)["fields"].(map[string]any)
	if fields["translations.0.vi"] == nil || fields["translations.0.en"] == nil {
		t.Fatalf("fields = %v", fields)
	}

	if rec := a.do(t, http.MethodPut, "/api/admin/lessons/missing/practice/translations", "admin", `{"translations":[]}`); rec.Code != http.StatusNotFound {
		t.Errorf("missing: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, path, "learner", `{"translations":[]}`); rec.Code != http.StatusForbidden {
		t.Errorf("learner: %d", rec.Code)
	}
	if rec := a.do(t, http.MethodPut, path, "", `{"translations":[]}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}

	// The practice changes between the read and the write.
	a.env.lessons.beforeSavePractice = func() {
		_, _ = a.env.lessons.update(l.ID, func(l *Lesson) bool { l.PracticeVersion++; return true })
	}
	rec = a.do(t, http.MethodPut, path, "admin", `{"translations":[]}`)
	if rec.Code != http.StatusConflict || decodeBody(t, rec)["error"] != "practice_changed" {
		t.Errorf("changed: %d %s", rec.Code, rec.Body)
	}
	a.env.lessons.beforeSavePractice = nil

	_, _ = a.env.lessons.update(l.ID, func(l *Lesson) bool { l.PracticeStatus = StatusRunning; return true })
	rec = a.do(t, http.MethodPut, path, "admin", `{"translations":[]}`)
	if rec.Code != http.StatusConflict || decodeBody(t, rec)["error"] != "practice_running" {
		t.Errorf("running: %d %s", rec.Code, rec.Body)
	}

	_, _ = a.env.lessons.update(l.ID, func(l *Lesson) bool { l.PracticeStatus, l.Practice = StatusNone, nil; return true })
	rec = a.do(t, http.MethodPut, path, "admin", `{"translations":[]}`)
	if rec.Code != http.StatusConflict || decodeBody(t, rec)["error"] != "no_practice" {
		t.Errorf("no practice: %d %s", rec.Code, rec.Body)
	}
}
