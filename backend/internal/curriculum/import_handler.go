package curriculum

import (
	"errors"
	"net/http"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// maxImportBytes bounds an imported lesson: its text, annotations, practice and glosses.
const maxImportBytes = 1 << 20

type briefGrammarJSON struct {
	ID       string   `json:"id"`
	TitleVi  string   `json:"titleVi"`
	TitleEn  string   `json:"titleEn"`
	Pattern  string   `json:"pattern"`
	HintVi   string   `json:"hintVi"`
	Examples []string `json:"examples"`
}

type briefJSON struct {
	Key            string             `json:"key"`
	Level          string             `json:"level"`
	TopicID        string             `json:"topicId"`
	TopicName      string             `json:"topicName"`
	TopicNumber    int                `json:"topicNumber"`
	CanDoVi        string             `json:"canDoVi"`
	Kind           Kind               `json:"kind"`
	Title          string             `json:"title"`
	Idea           string             `json:"idea"`
	Grammar        []briefGrammarJSON `json:"grammar"`
	TargetWords    []string           `json:"targetWords"`
	TopicWords     []string           `json:"topicWords"`
	Words          int                `json:"words"`
	MinGrammarUses int                `json:"minGrammarUses"`
	Filled         bool               `json:"filled"`
}

// importJSON is a lesson for a slot: its text and, in the shapes the AI returns them, its
// annotations and extras (annotate), practice and steps (practice) and glosses.
type importJSON struct {
	Content     string          `json:"content"`
	TargetWords []string        `json:"targetWords"`
	Annotate    ai.LessonExtras `json:"annotate"`
	Practice    ai.Practice     `json:"practice"`
	Glosses     []ai.Gloss      `json:"glosses"`
}

type importPartJSON struct {
	Key     string   `json:"key"`
	LabelVi string   `json:"labelVi"`
	Given   int      `json:"given"`
	Kept    int      `json:"kept"`
	RuleVi  string   `json:"ruleVi,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

type importReportJSON struct {
	Sentences      int              `json:"sentences"`
	FromDictionary int              `json:"fromDictionary"`
	Parts          []importPartJSON `json:"parts"`
}

type importResultJSON struct {
	DryRun bool             `json:"dryRun"`
	Slot   slotJSON         `json:"slot"`
	Checks checksJSON       `json:"checks"`
	Report importReportJSON `json:"report"`
}

func toImportReportJSON(r lesson.ImportReport) importReportJSON {
	out := importReportJSON{Sentences: r.Sentences, FromDictionary: r.FromDictionary, Parts: make([]importPartJSON, len(r.Parts))}
	for i, p := range r.Parts {
		out.Parts[i] = importPartJSON(p)
	}
	return out
}

func (h *Handler) brief(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.Brief(r.Context(), r.PathValue("key"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := briefJSON{
		Key: b.Key, Level: b.Level, TopicID: b.TopicID, TopicName: b.TopicName, TopicNumber: b.TopicNumber,
		CanDoVi: b.CanDoVi, Kind: b.Kind, Title: b.Title, Idea: b.Idea, Grammar: make([]briefGrammarJSON, len(b.Grammar)),
		TargetWords: b.TargetWords, TopicWords: b.TopicWords, Words: b.Words, MinGrammarUses: b.MinGrammarUses, Filled: b.Filled,
	}
	for i, g := range b.Grammar {
		out.Grammar[i] = briefGrammarJSON(g)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// importSlot saves the lesson of an empty slot from a lesson written outside the app; with
// ?dryRun=true it only checks it. An invalid lesson gets 400 with its fields and the report.
func (h *Handler) importSlot(w http.ResponseWriter, r *http.Request) {
	var in importJSON
	if httpx.DecodeJSONDetailed(w, r, &in, maxImportBytes) != nil {
		return
	}
	dryRun := r.URL.Query().Get("dryRun") == "true"
	res, err := h.svc.ImportSlot(r.Context(), r.PathValue("key"), SlotImport{
		Content: in.Content, TargetWords: in.TargetWords,
		ImportContent: lesson.ImportContent{Annotate: in.Annotate, Practice: in.Practice, Glosses: in.Glosses},
	}, dryRun)
	var verr *lesson.ValidationError
	if errors.As(err, &verr) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "validation_failed", "message": "Bài nhập chưa hợp lệ", "fields": verr.Fields,
			"report": toImportReportJSON(res.Report),
		})
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	status := http.StatusCreated
	if dryRun {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, importResultJSON{
		DryRun: dryRun, Slot: toSlotJSON(res.Slot), Checks: toChecksJSON(res.Checks), Report: toImportReportJSON(res.Report),
	})
}
