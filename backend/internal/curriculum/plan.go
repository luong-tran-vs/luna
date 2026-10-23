// Package curriculum holds the course of each level (added 2026-10-22, A1 first,
// design/de-xuat-giao-trinh-a1.png; in the database for every level since 2026-10-23): its topics
// in order, each with its grammar points and lesson slots, 3 main lessons then the extra ones. A
// slot is filled by the lesson of the topic's roadmap with the slot's title. The AI drafts the
// course of a level, the admin edits and approves it, applies the order to the topics, generates
// the empty slots and sees checks on drafts; learners see the topic's number, grammar and which
// lessons are extra.
package curriculum

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luongtran/luna/backend/internal/grammar"
)

// Kind tells a main lesson, needed to move on to the next topic, from an extra one.
type Kind string

const (
	KindMain  Kind = "main"
	KindExtra Kind = "extra"
)

// What a course should hold (2026-10-23): at least MinTopics topics, each with MainPerTopic main
// lessons and MinExtra–MaxExtra extra ones to practise. A1 was approved with 2–3 extra lessons, so
// these are warnings; MinExtraHard is what approving needs. MaxTopics is 12 since A1 was approved
// with 12 topics.
const (
	MainPerTopic = 3
	MinTopics    = 7
	MaxTopics    = 12
	MinExtra     = 4
	MaxExtra     = 5
	MinExtraHard = 2
)

// Slot is one lesson of the course.
type Slot struct {
	Key   string `json:"key"`
	Kind  Kind   `json:"kind"`
	Title string `json:"title"`
	// Grammar lists the slot's syllabus points, the first being the one the lesson is built on.
	Grammar []string `json:"grammar"`
	// Idea is the story given to the AI (in English, it goes into the prompt).
	Idea string `json:"idea"`
}

// Topic is one topic of the course, named as in the topic catalogue.
type Topic struct {
	// Number is the topic's place in the course, from 1.
	Number int `json:"-"`
	// TopicID is the catalogue's topic; "" until the topic is created (it is then found by name).
	TopicID string `json:"topicId,omitempty"`
	Name    string `json:"name"`
	// CanDoVi ends "Sau chủ đề này bạn có thể …".
	CanDoVi string   `json:"canDoVi"`
	Grammar []string `json:"grammar"`
	Slots   []Slot   `json:"slots"`
}

// Plan is the course of one level.
type Plan struct {
	Level  string  `json:"level"`
	Topics []Topic `json:"topics"`
}

//go:embed a1.json
var a1JSON []byte

// seeds are the courses written before they were kept in the database: the first read of a level
// without a course saves its seed as approved.
var seeds = map[string][]byte{"A1": a1JSON}

// seed returns the seed of level.
func seed(level string) (Plan, bool) {
	data, ok := seeds[level]
	if !ok {
		return Plan{}, false
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		panic(fmt.Sprintf("curriculum: seed %s: %v", level, err))
	}
	p.Normalize()
	if err := p.Validate(grammar.Default()); err != nil {
		panic(err)
	}
	return p, true
}

// Normalize numbers the topics, makes the first MainPerTopic slots of each topic main and the
// others extra, and gives every slot the key "<level>-<topic>-<slot>" (a1-03-2). Lessons fill
// slots by title, so keys may change when the admin moves slots.
func (p *Plan) Normalize() {
	for i := range p.Topics {
		t := &p.Topics[i]
		t.Number = i + 1
		t.Name = strings.TrimSpace(t.Name)
		t.CanDoVi = strings.TrimSpace(t.CanDoVi)
		for j := range t.Slots {
			s := &t.Slots[j]
			s.Kind = KindMain
			if j >= MainPerTopic {
				s.Kind = KindExtra
			}
			s.Key = fmt.Sprintf("%s-%02d-%d", strings.ToLower(p.Level), i+1, j+1)
			s.Title = strings.TrimSpace(s.Title)
			s.Idea = strings.TrimSpace(s.Idea)
		}
	}
}

// Validate checks what a saved course must hold: named topics, each once, with points of the
// plan's level, and slots with a title (once per topic), an idea and points. A topic may have no
// slot yet while the AI has not written them.
func (p Plan) Validate(syl *grammar.Syllabus) error {
	keys := map[string]bool{}
	names := map[string]bool{}
	checkPoints := func(where string, ids []string) error {
		if len(ids) == 0 {
			return fmt.Errorf("%s: chưa có điểm ngữ pháp", where)
		}
		for _, id := range ids {
			pt, ok := syl.Get(id)
			if !ok || pt.Level != p.Level {
				return fmt.Errorf("%s: %q không phải điểm ngữ pháp %s", where, id, p.Level)
			}
		}
		return nil
	}
	for _, t := range p.Topics {
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("chủ đề %d chưa có tên", t.Number)
		}
		if names[NameKey(t.Name)] {
			return fmt.Errorf("chủ đề “%s” có hai lần", t.Name)
		}
		names[NameKey(t.Name)] = true
		if err := checkPoints("Chủ đề “"+t.Name+"”", t.Grammar); err != nil {
			return err
		}
		titles := map[string]bool{}
		for i, s := range t.Slots {
			if keys[s.Key] || s.Key == "" {
				return fmt.Errorf("chủ đề “%s”: khoá bài %q trống hoặc trùng", t.Name, s.Key)
			}
			keys[s.Key] = true
			if strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Idea) == "" {
				return fmt.Errorf("chủ đề “%s”, bài %d: chưa có tên hoặc ý tưởng", t.Name, i+1)
			}
			if titles[TitleKey(s.Title)] {
				return fmt.Errorf("chủ đề “%s”: bài “%s” có hai lần", t.Name, s.Title)
			}
			titles[TitleKey(s.Title)] = true
			want := KindMain
			if i >= MainPerTopic {
				want = KindExtra
			}
			if s.Kind != want {
				return fmt.Errorf("chủ đề “%s”: bài %s phải là bài %s", t.Name, s.Key, want)
			}
			if err := checkPoints(fmt.Sprintf("Chủ đề “%s”, bài “%s”", t.Name, s.Title), s.Grammar); err != nil {
				return err
			}
		}
	}
	return nil
}

// Ready checks what approving needs besides Validate: some topics, each with its main lessons and
// MinExtraHard extra ones.
func (p Plan) Ready() error {
	if len(p.Topics) == 0 {
		return fmt.Errorf("giáo trình %s chưa có chủ đề", p.Level)
	}
	for _, t := range p.Topics {
		if len(t.Slots) < MainPerTopic+MinExtraHard {
			return fmt.Errorf("chủ đề “%s” cần ít nhất %d bài chính và %d bài thêm", t.Name, MainPerTopic, MinExtraHard)
		}
	}
	return nil
}

// Warnings lists what the course lacks without blocking it: too few topics, extra lessons outside
// MinExtra–MaxExtra, and grammar points of the level no topic teaches.
func (p Plan) Warnings(syl *grammar.Syllabus) []string {
	var out []string
	if len(p.Topics) < MinTopics {
		out = append(out, fmt.Sprintf("Mới có %d chủ đề, nên có ít nhất %d.", len(p.Topics), MinTopics))
	}
	taught := map[string]bool{}
	for _, t := range p.Topics {
		for _, id := range t.Grammar {
			taught[id] = true
		}
		extra := 0
		for _, s := range t.Slots {
			for _, id := range s.Grammar {
				taught[id] = true
			}
			if s.Kind == KindExtra {
				extra++
			}
		}
		switch {
		case len(t.Slots) == 0:
			out = append(out, fmt.Sprintf("Chủ đề “%s” chưa có bài.", t.Name))
		case len(t.Slots) < MainPerTopic:
			out = append(out, fmt.Sprintf("Chủ đề “%s” mới có %d bài chính, cần %d.", t.Name, len(t.Slots), MainPerTopic))
		case extra < MinExtra || extra > MaxExtra:
			out = append(out, fmt.Sprintf("Chủ đề “%s” có %d bài thêm, nên %d–%d.", t.Name, extra, MinExtra, MaxExtra))
		}
	}
	var missing []string
	for _, pt := range syl.ByLevel(p.Level) {
		if !taught[pt.ID] {
			missing = append(missing, pt.TitleVi)
		}
	}
	if len(missing) > 0 {
		out = append(out, fmt.Sprintf("Chưa chủ đề nào dạy %d điểm ngữ pháp: %s.", len(missing), strings.Join(missing, ", ")))
	}
	return out
}

// FindSlot returns the topic and slot of key.
func (p Plan) FindSlot(key string) (Topic, Slot, bool) {
	for _, t := range p.Topics {
		for _, s := range t.Slots {
			if s.Key == key {
				return t, s, true
			}
		}
	}
	return Topic{}, Slot{}, false
}

// LevelOfKey reads the level from a slot key ("a2-03-1" → "A2").
func LevelOfKey(key string) string {
	level, _, _ := strings.Cut(key, "-")
	return strings.ToUpper(level)
}

// NameKey compares topic names ignoring case and spaces.
func NameKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// TitleKey compares lesson titles ignoring case, spaces and the kind of apostrophe.
func TitleKey(s string) string {
	return NameKey(strings.ReplaceAll(s, "’", "'"))
}

// PointTopic is the first topic of a level's course that teaches a grammar point.
type PointTopic struct {
	Number int
	Name   string
}

// PointTopics maps each grammar point the course teaches to its first topic, and lists the points
// in course order.
func (p Plan) PointTopics() (map[string]PointTopic, []string) {
	byPoint := map[string]PointTopic{}
	var order []string
	for _, t := range p.Topics {
		for _, id := range t.Grammar {
			if _, seen := byPoint[id]; !seen {
				byPoint[id] = PointTopic{Number: t.Number, Name: t.Name}
				order = append(order, id)
			}
		}
	}
	return byPoint, order
}
