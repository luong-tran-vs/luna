package progress

import (
	"net/http"

	"github.com/luongtran/luna/backend/internal/platform/httpx"
)

// --- JSON shapes (contracts/dashboard-api.md) ---

type skillsJSON struct {
	Read   int `json:"read"`
	Listen int `json:"listen"`
	Write  int `json:"write"`
	Total  int `json:"total"`
}

type dashboardLessonJSON struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	TopicName string `json:"topicName"`
	Level     string `json:"level"`
}

type actionJSON struct {
	Kind string `json:"kind"`
	Step string `json:"step"`
}

type dashboardJSON struct {
	Kind          string               `json:"kind"`
	Goal          *goalJSON            `json:"goal"`
	GoalCompleted bool                 `json:"goalCompleted"`
	Skills        *skillsJSON          `json:"skills"`
	Lesson        *dashboardLessonJSON `json:"lesson"`
	Steps         map[string]string    `json:"steps"`
	CurrentStep   string               `json:"currentStep"`
	Action        *actionJSON          `json:"action"`
	Streak        int                  `json:"streak"`
	TomorrowCards int                  `json:"tomorrowCards"`
}

func toDashboardJSON(v DashboardView) dashboardJSON {
	out := dashboardJSON{
		Kind: string(v.Kind), GoalCompleted: v.GoalCompleted, Steps: map[string]string{}, CurrentStep: string(v.CurrentStep),
		Streak: v.Streak, TomorrowCards: v.TomorrowCards,
	}
	if v.Goal != nil {
		g := toGoalJSON(*v.Goal)
		out.Goal = &g
	}
	if v.Skills != nil {
		s := skillsJSON(*v.Skills)
		out.Skills = &s
	}
	if v.Lesson != nil {
		l := dashboardLessonJSON(*v.Lesson)
		out.Lesson = &l
	}
	if v.Action != nil {
		out.Action = &actionJSON{Kind: string(v.Action.Kind), Step: string(v.Action.Step)}
	}
	for s, st := range v.Steps {
		out.Steps[string(s)] = string(st)
	}
	return out
}

type statsJSON struct {
	Cards     int `json:"cards"`
	Dictation struct {
		Sentences    int      `json:"sentences"`
		CorrectWords int      `json:"correctWords"`
		TotalWords   int      `json:"totalWords"`
		Rate         *float64 `json:"rate"`
	} `json:"dictation"`
	Lessons struct {
		Read      int `json:"read"`
		Listen    int `json:"listen"`
		Write     int `json:"write"`
		Completed int `json:"completed"`
	} `json:"lessons"`
	Reading struct {
		Answered int      `json:"answered"`
		Correct  int      `json:"correct"`
		Rate     *float64 `json:"rate"`
	} `json:"reading"`
	Writing struct {
		Submitted    int      `json:"submitted"`
		AverageScore *float64 `json:"averageScore"`
	} `json:"writing"`
}

// --- handlers ---

func (h *StudyHandler) dashboard(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.Dashboard(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toDashboardJSON(v))
}

func (h *StudyHandler) stats(w http.ResponseWriter, r *http.Request) {
	p, _ := httpx.PrincipalFrom(r.Context())
	v, err := h.svc.Stats(r.Context(), p.UserID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var out statsJSON
	out.Cards = v.Cards
	out.Dictation.Sentences, out.Dictation.CorrectWords, out.Dictation.TotalWords = v.Dictation.Sentences, v.Dictation.CorrectWords, v.Dictation.TotalWords
	out.Dictation.Rate = v.Rate
	out.Lessons.Read, out.Lessons.Listen, out.Lessons.Completed = v.Lessons.Read, v.Lessons.Listen, v.Lessons.Completed
	out.Reading.Answered, out.Reading.Correct, out.Reading.Rate = v.Reading.Answered, v.Reading.Correct, v.ReadingRate
	out.Lessons.Write = v.Lessons.Write
	out.Writing.Submitted, out.Writing.AverageScore = v.Writing.Submitted, v.Writing.AverageScore
	httpx.WriteJSON(w, http.StatusOK, out)
}
