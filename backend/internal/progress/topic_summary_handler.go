package progress

import (
	"net/http"
	"time"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

type summaryLessonJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type skillJSON struct {
	Skill   string `json:"skill"`
	Percent int    `json:"percent"`
	Correct int    `json:"correct"`
	Total   int    `json:"total"`
}

type mistakeCountJSON struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type topicSummaryJSON struct {
	TopicID        string              `json:"topicId"`
	TopicName      string              `json:"topicName"`
	Level          string              `json:"level"`
	CanDoVi        string              `json:"canDoVi"`
	GrammarVi      []string            `json:"grammarVi"`
	Lessons        []summaryLessonJSON `json:"lessons"`
	Completed      int                 `json:"completed"`
	StartedAt      *time.Time          `json:"startedAt"`
	FinishedAt     *time.Time          `json:"finishedAt"`
	Days           int                 `json:"days"`
	Words          int                 `json:"words"`
	Remembered     int                 `json:"remembered"`
	Skills         []skillJSON         `json:"skills"`
	WritingAverage float64             `json:"writingAverage"`
	Mistakes       []mistakeCountJSON  `json:"mistakes"`
	Overall        int                 `json:"overall"`
	Rating         string              `json:"rating"`
}

func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// topicSummary serves GET /api/topics/{id}/summary?level=: the learner's summary of a topic.
func (h *StudyHandler) topicSummary(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.TopicSummary(r.Context(), p.UserID, r.PathValue("id"), r.URL.Query().Get("level"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	out := topicSummaryJSON{
		TopicID: v.TopicID, TopicName: v.TopicName, Level: v.Level, CanDoVi: v.CanDoVi, GrammarVi: v.GrammarVi,
		Lessons: make([]summaryLessonJSON, len(v.Lessons)), Completed: v.Completed,
		StartedAt: timeOrNil(v.StartedAt), FinishedAt: timeOrNil(v.FinishedAt), Days: v.Days,
		Words: v.Words, Remembered: v.Remembered, Skills: make([]skillJSON, len(v.Skills)),
		WritingAverage: v.WritingAverage, Mistakes: make([]mistakeCountJSON, len(v.Mistakes)),
		Overall: v.Overall, Rating: string(v.Rating),
	}
	if out.GrammarVi == nil {
		out.GrammarVi = []string{}
	}
	for i, l := range v.Lessons {
		out.Lessons[i] = summaryLessonJSON(l)
	}
	for i, s := range v.Skills {
		out.Skills[i] = skillJSON(s)
	}
	for i, m := range v.Mistakes {
		out.Mistakes[i] = mistakeCountJSON(m)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
