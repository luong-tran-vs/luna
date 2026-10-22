package curriculum

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luongtran/luna/backend/internal/cefr"
	"github.com/luongtran/luna/backend/internal/grammar"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/wordmatch"
)

// The checks of a lesson before it is published (design section 3): the generation prompt asks for
// all of them, these count them on what came back.

// MinGrammarUses is how often a lesson should use its grammar point (the prompt asks for 3).
const MinGrammarUses = 3

// lengthRange is the word count expected of a lesson of a level: the admin's default ±25 %.
var lengthRange = map[string][2]int{"A1": {90, 150}, "A2": {120, 200}}

// usage finds the uses of an A1 grammar point; a point without one is not counted.
var usage = map[string]*regexp.Regexp{
	"a1-to-be":        regexp.MustCompile(`(?i)\b(am|is|are|isn't|aren't|i'm|you're|we're|they're)\b|\b(he|she|it|that|there)'s\b`),
	"a1-pronouns":     regexp.MustCompile(`\b(I|[Yy]ou|[Hh]e|[Ss]he|[Ii]t|[Ww]e|[Tt]hey|me|him|her|us|them)\b`),
	"a1-articles":     regexp.MustCompile(`(?i)\b(a|an|the)\b`),
	"a1-this-that":    regexp.MustCompile(`(?i)\b(this|that|these|those)\b`),
	"a1-plural-nouns": regexp.MustCompile(`(?i)\b(two|three|four|five|six|seven|eight|nine|ten|many|some|these|those|lots of|a lot of)\s+([a-z]+\s+)?[a-z]+(s|es)\b`),
	"a1-possessives":  regexp.MustCompile(`(?i)\b(my|your|his|her|its|our|their)\b|\b[a-z]+'s\s+[a-z]+`),
	"a1-have-has":     regexp.MustCompile(`(?i)\b(have|has|haven't|hasn't)\b`),
	"a1-there-is-are": regexp.MustCompile(`(?i)\bthere\s+(is|are|isn't|aren't)\b|\bthere's\b|\b(is|are)\s+there\b`),
	"a1-present-simple": regexp.MustCompile(
		`(?i)\b(he|she|it)\s+(always\s+|usually\s+|often\s+|sometimes\s+|never\s+)?[a-z]+(s|es)\b|\b(I|you|we|they)\s+(always\s+|usually\s+|often\s+|sometimes\s+|never\s+)?(like|love|go|get|have|eat|drink|live|work|play|walk|read|watch|listen|study|help|cook|clean|take|make|come|want|need|know|see|visit|wake|sleep|start|finish)\b`),
	"a1-present-simple-neg-q": regexp.MustCompile(`(?i)\b(don't|doesn't|do not|does not)\b|\b(do|does)\s+(I|you|he|she|it|we|they|[a-z]+)\s+[a-z]+`),
	"a1-imperatives": regexp.MustCompile(
		`(?m)(^|[.!?]\s+|"\s*)((?:Then|And|Now),?\s+)?(?i:go|turn|take|walk|cross|stop|look|come|open|close|sit|stand|listen|wait|follow|get|don't|please)\b`),
	"a1-can":                regexp.MustCompile(`(?i)\b(can|can't|cannot)\b`),
	"a1-question-words":     regexp.MustCompile(`(?i)\b(what|where|when|who|how|why|which)\b`),
	"a1-prepositions-place": regexp.MustCompile(`(?i)\b(in front of|next to|behind|between|under|near|opposite|on the|in the)\b`),
	"a1-prepositions-time": regexp.MustCompile(
		`(?i)\b(at|on|in)\s+(the\s+)?(\d+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|noon|night|midnight|morning|afternoon|evening|weekend|weekends|monday|tuesday|wednesday|thursday|friday|saturday|sunday|january|february|march|april|may|june|july|august|september|october|november|december|spring|summer|autumn|fall|winter)\b`),
	"a1-adverbs-frequency": regexp.MustCompile(`(?i)\b(always|usually|often|sometimes|rarely|seldom|never)\b`),
	"a1-like-love-ing":     regexp.MustCompile(`(?i)\b(like|likes|love|loves|enjoy|enjoys|hate|hates)\s+[a-z]+ing\b`),
	"a1-conjunctions":      regexp.MustCompile(`(?i)\b(and|but|or|because)\b`),
}

var (
	otherPersonRe = regexp.MustCompile(`(?i)\b(he|she|his|her|him)\b`)
	negativeRe    = regexp.MustCompile(`(?i)\b(not|never)\b|n't\b`)
)

// Check is one line of the checks: OK passed, Warn is only a warning (the admin may publish).
type Check struct {
	Key    string
	OK     bool
	Warn   bool
	TextVi string
}

// Checks are the checks of one lesson; Above lists the words above its level.
type Checks struct {
	Items []Check
	Above []cefr.Word
}

// GrammarUses counts the uses of a grammar point in text; ok is false when the point has no
// pattern.
func GrammarUses(pointID, text string) (int, bool) {
	re, ok := usage[pointID]
	if !ok {
		return 0, false
	}
	return len(re.FindAllStringIndex(text, -1)), true
}

// Inspect runs the checks on a lesson.
func Inspect(l lesson.Lesson) Checks {
	text := l.Content
	var out Checks
	add := func(key string, ok, warn bool, text string) {
		out.Items = append(out.Items, Check{Key: key, OK: ok, Warn: !ok && warn, TextVi: text})
	}
	if l.GrammarPointID == "" {
		add("grammar", false, false, "Chưa gắn điểm ngữ pháp")
	} else if p, found := grammar.Default().Get(l.GrammarPointID); found {
		if n, counted := GrammarUses(l.GrammarPointID, text); counted {
			add("grammar", n >= MinGrammarUses, false, fmt.Sprintf("Dùng \"%s\" %d lần (cần ít nhất %d)", p.TitleVi, n, MinGrammarUses))
		} else {
			add("grammar", true, false, fmt.Sprintf("Điểm ngữ pháp: %s (không tự đếm được)", p.TitleVi))
		}
	}
	add("otherPerson", otherPersonRe.MatchString(text), false, "Có đoạn về người khác (he / she)")
	add("negative", negativeRe.MatchString(text), false, "Có câu phủ định")
	add("question", strings.Contains(text, "?"), true, "Có câu hỏi")
	if len(l.TargetWords) > 0 {
		m := wordmatch.New(text, nil)
		var missing []string
		for _, w := range l.TargetWords {
			if !m.Contains(w) {
				missing = append(missing, w)
			}
		}
		msg := fmt.Sprintf("Đủ %d/%d từ mục tiêu", len(l.TargetWords)-len(missing), len(l.TargetWords))
		if len(missing) > 0 {
			msg = fmt.Sprintf("Thiếu %d/%d từ mục tiêu: %s", len(missing), len(l.TargetWords), strings.Join(missing, ", "))
		}
		add("targetWords", len(missing) == 0, true, msg)
	}
	if r, ok := lengthRange[string(l.Level)]; ok {
		n := len(strings.Fields(text))
		add("length", n >= r[0] && n <= r[1], true, fmt.Sprintf("%d từ (nên %d–%d)", n, r[0], r[1]))
	}
	if l.Practice != nil && l.Practice.Dialogue != nil {
		questions, copied := 0, 0
		sentences := map[string]bool{}
		for _, s := range l.Sentences {
			sentences[TitleKey(s.Text)] = true
		}
		for _, t := range l.Practice.Dialogue.Turns {
			if strings.Contains(t.Text, "?") {
				questions++
			}
			if sentences[TitleKey(t.Text)] {
				copied++
			}
		}
		msg := fmt.Sprintf("Hội thoại: %d lượt hỏi", questions)
		if copied > 0 {
			msg += fmt.Sprintf(", %d câu chép nguyên từ bài đọc", copied)
		}
		add("dialogue", questions >= 3 && copied == 0, true, msg)
	}
	out.Above = cefr.Above(text, string(l.Level))
	if len(out.Above) > 0 {
		words := make([]string, len(out.Above))
		for i, w := range out.Above {
			words[i] = fmt.Sprintf("%s (%s)", w.Text, w.Level)
		}
		add("level", false, true, fmt.Sprintf("%d từ trên %s: %s", len(out.Above), l.Level, strings.Join(words, ", ")))
	} else {
		add("level", true, false, fmt.Sprintf("Không có từ trên %s", l.Level))
	}
	return out
}
