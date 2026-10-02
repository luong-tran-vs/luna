package lesson

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type practiceJSON struct {
	Status       string            `json:"status"`
	LessonNumber int               `json:"lessonNumber"`
	ObjectiveVi  string            `json:"objectiveVi"`
	Examples     []exampleJSON     `json:"examples"`
	Dialogue     *dialogueJSON     `json:"dialogue"`
	Fill         *fillJSON         `json:"fill"`
	GrammarTipVi string            `json:"grammarTipVi"`
	Translations []translationJSON `json:"translations"`
}

type exampleJSON struct {
	Lemma    string  `json:"lemma"`
	Sentence string  `json:"sentence"`
	AudioURL *string `json:"audioUrl"`
}

type dialogueJSON struct {
	Speakers []string   `json:"speakers"`
	Turns    []turnJSON `json:"turns"`
}

type turnJSON struct {
	Speaker   int     `json:"speaker"`
	Text      string  `json:"text"`
	MeaningVi string  `json:"meaningVi"`
	AudioURL  *string `json:"audioUrl"`
}

type fillJSON struct {
	Turns    []fillTurnJSON `json:"turns"`
	Blanks   []blankJSON    `json:"blanks"`
	WordBank []string       `json:"wordBank"`
}

type fillTurnJSON struct {
	Speaker   int        `json:"speaker"`
	TurnIndex int        `json:"turnIndex"`
	MeaningVi string     `json:"meaningVi"`
	Parts     []partJSON `json:"parts"`
}

// partJSON is {"text": "..."} or {"blank": i}.
type partJSON struct {
	Text  *string `json:"text,omitempty"`
	Blank *int    `json:"blank,omitempty"`
}

type blankJSON struct {
	Answer string `json:"answer"`
}

type translationJSON struct {
	Vi       string   `json:"vi"`
	Answer   []string `json:"answer"`
	Tiles    []string `json:"tiles"`
	AudioURL *string  `json:"audioUrl"`
}

// practiceStatusJSON names the "not generated yet" status for clients.
func practiceStatusJSON(s Status) string {
	if s == StatusNone {
		return "none"
	}
	return string(s)
}

func optionalURL(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toPracticeJSON(v PracticeView) practiceJSON {
	out := practiceJSON{
		Status: practiceStatusJSON(v.Status), LessonNumber: v.LessonNumber,
		ObjectiveVi: v.ObjectiveVi, GrammarTipVi: v.GrammarTipVi,
		Examples:     make([]exampleJSON, len(v.Examples)),
		Translations: make([]translationJSON, len(v.Translations)),
	}
	for i, e := range v.Examples {
		out.Examples[i] = exampleJSON{Lemma: e.Lemma, Sentence: e.Sentence, AudioURL: optionalURL(e.AudioURL)}
	}
	if v.Dialogue != nil {
		out.Dialogue = &dialogueJSON{Speakers: v.Dialogue.Speakers, Turns: make([]turnJSON, len(v.Dialogue.Turns))}
		for i, t := range v.Dialogue.Turns {
			out.Dialogue.Turns[i] = turnJSON{Speaker: t.Speaker, Text: t.Text, MeaningVi: t.MeaningVi, AudioURL: optionalURL(t.AudioURL)}
		}
	}
	if f := v.Fill; f != nil {
		out.Fill = &fillJSON{Turns: make([]fillTurnJSON, len(f.Turns)), Blanks: make([]blankJSON, len(f.Blanks)), WordBank: f.WordBank}
		for i, t := range f.Turns {
			ft := fillTurnJSON{Speaker: t.Speaker, TurnIndex: t.TurnIndex, MeaningVi: t.MeaningVi, Parts: make([]partJSON, len(t.Parts))}
			for k, p := range t.Parts {
				if p.Blank != nil {
					ft.Parts[k] = partJSON{Blank: p.Blank}
				} else {
					ft.Parts[k] = partJSON{Text: &p.Text}
				}
			}
			out.Fill.Turns[i] = ft
		}
		for i, b := range f.Blanks {
			out.Fill.Blanks[i] = blankJSON(b)
		}
	}
	for i, t := range v.Translations {
		out.Translations[i] = translationJSON{Vi: t.Vi, Answer: t.Answer, Tiles: t.Tiles, AudioURL: optionalURL(t.AudioURL)}
	}
	return out
}

func (h *ReadingHandler) practice(w http.ResponseWriter, r *http.Request) {
	v, err := h.reader.Practice(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err, "not_found")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toPracticeJSON(v))
}

// practiceAudioKinds are the valid {kind} values of a practice audio URL.
var practiceAudioKinds = map[string]bool{audioExample: true, audioTurn: true, audioAnswer: true}

// PracticeAudioHandler serves GET /api/audio/{lessonId}/{revision}/practice/{version}/{kind}/{index}
// from dir. Like AudioHandler, path values are parsed strictly so no user text reaches the
// file path.
func PracticeAudioHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, kind := r.PathValue("lessonId"), r.PathValue("kind")
		revision, revErr := strconv.Atoi(r.PathValue("revision"))
		version, verErr := strconv.Atoi(r.PathValue("version"))
		index, idxErr := strconv.Atoi(r.PathValue("index"))
		if !objectIDPattern.MatchString(id) || !practiceAudioKinds[kind] || revErr != nil || verErr != nil || idxErr != nil ||
			revision < 0 || version < 0 || index < 0 {
			http.NotFound(w, r)
			return
		}

		file := filepath.Join(dir, id, strconv.Itoa(revision), "practice", strconv.Itoa(version), audioName(kind, index)+".mp3")
		w.Header().Set("Content-Type", "audio/mpeg")
		// The URL contains the content revision and practice version, so a given URL never changes.
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		http.ServeFile(w, r, file) //nolint:gosec // file is built only from a validated hex id, a known kind and integers
	})
}

// adminPracticeJSON is the stored practice as the admin lesson page shows it.
type adminPracticeJSON struct {
	ObjectiveVi  string                 `json:"objectiveVi"`
	Examples     []adminExampleJSON     `json:"examples"`
	Dialogue     *adminDialogueJSON     `json:"dialogue"`
	GrammarTipVi string                 `json:"grammarTipVi"`
	Translations []adminTranslationJSON `json:"translations"`
}

type adminExampleJSON struct {
	Lemma    string `json:"lemma"`
	Sentence string `json:"sentence"`
}

type adminDialogueJSON struct {
	Speakers []string        `json:"speakers"`
	Turns    []adminTurnJSON `json:"turns"`
}

type adminTurnJSON struct {
	Speaker   int    `json:"speaker"`
	Text      string `json:"text"`
	MeaningVi string `json:"meaningVi"`
}

type adminTranslationJSON struct {
	Vi          string   `json:"vi"`
	En          string   `json:"en"`
	Distractors []string `json:"distractors"`
}

func toAdminPracticeJSON(p *Practice) *adminPracticeJSON {
	if p == nil {
		return nil
	}
	out := &adminPracticeJSON{
		ObjectiveVi: p.ObjectiveVi, GrammarTipVi: p.GrammarTipVi,
		Examples:     make([]adminExampleJSON, len(p.Examples)),
		Translations: make([]adminTranslationJSON, len(p.Translations)),
	}
	for i, e := range p.Examples {
		out.Examples[i] = adminExampleJSON(e)
	}
	if p.Dialogue != nil {
		out.Dialogue = &adminDialogueJSON{Speakers: p.Dialogue.Speakers, Turns: make([]adminTurnJSON, len(p.Dialogue.Turns))}
		for i, t := range p.Dialogue.Turns {
			out.Dialogue.Turns[i] = adminTurnJSON(t)
		}
	}
	for i, t := range p.Translations {
		out.Translations[i] = adminTranslationJSON{Vi: t.Vi, En: t.En, Distractors: nonNil(t.Distractors)}
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (h *Handler) regeneratePractice(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.RegeneratePractice(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeLesson(w, r, http.StatusAccepted, l)
}
