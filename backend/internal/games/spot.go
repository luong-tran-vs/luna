package games

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"
)

// "Bắn lỗi ngữ pháp" (P5, research R4): sentences built from what admins approved — choice
// exercises with a wrong option in the blank, fix exercises and common mistakes that differ from
// their correction by a few words, and extra sentences approved by an admin.

// Round shape.
const (
	spotQuestions = 10
	spotRight     = 3 // sentences without a mistake in a round
	maxSpotSpan   = 3 // words of a mistake
	blank         = "___"
)

// Sources of a sentence.
const (
	SourceChoice  = "choice"
	SourceFix     = "fix"
	SourceMistake = "mistake"
	SourceSpot    = "spot"
)

// tokenize splits a sentence into words, each punctuation mark its own token; apostrophes and
// hyphens inside a word stay ("aren't", "well-known").
func tokenize(s string) []string {
	var out []string
	for _, field := range strings.Fields(s) {
		runes := []rune(field)
		start, end := 0, len(runes)
		var lead, trail []string
		for start < end && isMark(runes[start]) {
			lead = append(lead, string(runes[start]))
			start++
		}
		for end > start && isMark(runes[end-1]) {
			trail = append([]string{string(runes[end-1])}, trail...)
			end--
		}
		out = append(out, lead...)
		if start < end {
			out = append(out, string(runes[start:end]))
		}
		out = append(out, trail...)
	}
	return out
}

// isMark reports whether r is punctuation that stands apart from words.
func isMark(r rune) bool {
	if r == '\'' || r == '’' || r == '-' {
		return false
	}
	return unicode.IsPunct(r) || r == '–' || r == '—'
}

func isMarkToken(t string) bool {
	for _, r := range t {
		if !isMark(r) {
			return false
		}
	}
	return t != ""
}

// diffSpan finds the shortest middle part where two token lists differ: the common start and end
// are skipped. It returns the differing ranges of a and b.
func diffSpan(a, b []string) (Span, Span) {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	return Span{pre, len(a) - suf}, Span{pre, len(b) - suf}
}

// spotCandidate is a sentence a round may ask.
type spotCandidate struct {
	q   SpotQuestion
	key string // one round uses a source once
	// wrongText is the mistake, for the third option of another sentence of the point.
	wrongText string
}

func joinTokens(ts []string) string { return strings.Join(ts, " ") }

// choiceCandidates builds a sentence with a mistake (a wrong option in the blank) and the right
// sentence from a choice exercise; none when the exercise does not have exactly one blank inside
// a sentence or its options are unusable.
func choiceCandidates(p PointSource, e SourceExercise, rnd *rand.Rand) (wrong, right *spotCandidate) {
	if strings.Count(e.Text, blank) != 1 || len(e.Options) < 3 || len(e.Options) > 4 ||
		e.AnswerIndex < 0 || e.AnswerIndex >= len(e.Options) {
		return nil, nil
	}
	before, after, _ := strings.Cut(e.Text, blank)
	if strings.TrimSpace(before+after) == "" || endsInLetter(before) || startsWithLetter(after) {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, o := range e.Options {
		k := normalize(o)
		if k == "" || strings.Contains(o, blank) || seen[k] || len(tokenize(o)) > maxSpotSpan {
			return nil, nil
		}
		seen[k] = true
	}
	head, tail := tokenize(before), tokenize(after)
	build := func(option string) ([]string, Span) {
		mid := tokenize(option)
		tokens := slices.Concat(head, mid, tail)
		return tokens, Span{len(head), len(head) + len(mid)}
	}
	key := p.PointID + "/" + e.ID
	rightTokens, _ := build(e.Options[e.AnswerIndex])
	right = &spotCandidate{key: key, q: SpotQuestion{
		Tokens: rightTokens, RuleVi: e.ExplanationVi, TopicVi: p.TitleVi, PointID: p.PointID, Source: SourceChoice,
	}}

	var wrongIdx []int
	for i := range e.Options {
		if i != e.AnswerIndex {
			wrongIdx = append(wrongIdx, i)
		}
	}
	shuffle(rnd, wrongIdx)
	shown := wrongIdx[0]
	tokens, span := build(e.Options[shown])
	fixes := []string{e.Options[e.AnswerIndex], e.Options[shown]}
	if len(wrongIdx) > 1 {
		fixes = append(fixes, e.Options[wrongIdx[1]])
	}
	if len(fixes) < 3 {
		return nil, right
	}
	q := SpotQuestion{Tokens: tokens, Wrong: &span, RuleVi: e.ExplanationVi, TopicVi: p.TitleVi, PointID: p.PointID, Source: SourceChoice}
	setOptions(&q, fixes, rnd)
	return &spotCandidate{key: key, q: q, wrongText: e.Options[shown]}, right
}

// setOptions shuffles the fixes into q; fixes[0] is the right one.
func setOptions(q *SpotQuestion, fixes []string, rnd *rand.Rand) {
	right := fixes[0]
	opts := slices.Clone(fixes)
	shuffle(rnd, opts)
	q.Options = opts
	q.Answer = slices.Index(opts, right)
}

func endsInLetter(s string) bool {
	r := []rune(s)
	return len(r) > 0 && (unicode.IsLetter(r[len(r)-1]) || unicode.IsDigit(r[len(r)-1]))
}

func startsWithLetter(s string) bool {
	r := []rune(s)
	return len(r) > 0 && (unicode.IsLetter(r[0]) || unicode.IsDigit(r[0]))
}

// pairCandidate is a wrong sentence and its correction that differ in one place of 1–3 words on
// each side (a fix exercise or a common mistake). The third option is filled in later.
type pairCandidate struct {
	spotCandidate
	rightText string
	right     *spotCandidate
}

func pairOf(p PointSource, key, source, wrong, right, rule string) *pairCandidate {
	if strings.ContainsAny(wrong+right, "()/") {
		return nil
	}
	wt, rt := tokenize(wrong), tokenize(right)
	ws, rs := diffSpan(wt, rt)
	wn, rn := ws.To-ws.From, rs.To-rs.From
	if wn < 1 || wn > maxSpotSpan || rn < 1 || rn > maxSpotSpan {
		return nil
	}
	if allMarks(wt[ws.From:ws.To]) || allMarks(rt[rs.From:rs.To]) {
		return nil
	}
	span := ws
	c := &pairCandidate{
		spotCandidate: spotCandidate{key: key, wrongText: joinTokens(wt[ws.From:ws.To]), q: SpotQuestion{
			Tokens: wt, Wrong: &span, RuleVi: rule, TopicVi: p.TitleVi, PointID: p.PointID, Source: source,
		}},
		rightText: joinTokens(rt[rs.From:rs.To]),
		right: &spotCandidate{key: key, q: SpotQuestion{
			Tokens: rt, RuleVi: rule, TopicVi: p.TitleVi, PointID: p.PointID, Source: source,
		}},
	}
	return c
}

func allMarks(ts []string) bool {
	for _, t := range ts {
		if !isMarkToken(t) {
			return false
		}
	}
	return true
}

// spotItemCandidate builds the sentence of an approved extra item and its right sentence.
func spotItemCandidate(p PointSource, it SpotItem, rnd *rand.Rand) (wrong, right *spotCandidate) {
	tokens := tokenize(it.Sentence)
	at := findTokens(tokens, tokenize(it.Wrong))
	if len(at) != 1 || len(it.Distractors) < 2 {
		return nil, nil
	}
	n := len(tokenize(it.Wrong))
	span := Span{at[0], at[0] + n}
	key := "spot/" + it.ID
	q := SpotQuestion{Tokens: tokens, Wrong: &span, RuleVi: it.RuleVi, TopicVi: p.TitleVi, PointID: p.PointID, Source: SourceSpot}
	setOptions(&q, []string{it.Right, it.Distractors[0], it.Distractors[1]}, rnd)
	fixed := slices.Concat(tokens[:span.From], tokenize(it.Right), tokens[span.To:])
	right = &spotCandidate{key: key, q: SpotQuestion{Tokens: fixed, RuleVi: it.RuleVi, TopicVi: p.TitleVi, PointID: p.PointID, Source: SourceSpot}}
	return &spotCandidate{key: key, q: q, wrongText: it.Wrong}, right
}

// findTokens returns every index where part starts in tokens, comparing case-insensitively.
func findTokens(tokens, part []string) []int {
	var out []int
	if len(part) == 0 {
		return nil
	}
	for i := 0; i+len(part) <= len(tokens); i++ {
		match := true
		for j, t := range part {
			if !strings.EqualFold(tokens[i+j], t) {
				match = false
				break
			}
		}
		if match {
			out = append(out, i)
		}
	}
	return out
}

// pointCandidates are the sentences of one point: with a mistake, and right ones.
type pointCandidates struct {
	wrong []*spotCandidate
	right []*spotCandidate
}

// candidatesOf builds every sentence of a point from its sources and approved items.
func candidatesOf(p PointSource, items []SpotItem, rnd *rand.Rand) pointCandidates {
	var out pointCandidates
	var pairs []*pairCandidate
	for _, e := range p.Exercises {
		switch e.Kind {
		case SourceChoice:
			w, r := choiceCandidates(p, e, rnd)
			if w != nil {
				out.wrong = append(out.wrong, w)
			}
			if r != nil {
				out.right = append(out.right, r)
			}
		case SourceFix:
			if c := pairOf(p, p.PointID+"/"+e.ID, SourceFix, e.Text, e.Sentence, e.ExplanationVi); c != nil {
				pairs = append(pairs, c)
			}
		}
	}
	for i, m := range p.Mistakes {
		if c := pairOf(p, fmt.Sprintf("%s/mistake%d", p.PointID, i), SourceMistake, m.Wrong, m.Right, m.NoteVi); c != nil {
			pairs = append(pairs, c)
		}
	}
	for _, it := range items {
		w, r := spotItemCandidate(p, it, rnd)
		if w != nil {
			out.wrong = append(out.wrong, w)
			out.right = append(out.right, r)
		}
	}
	// The third option of a pair is the mistake of another sentence of the point, different from
	// both of its own words; a pair without one is left out (its right sentence is still used).
	for _, c := range pairs {
		out.right = append(out.right, c.right)
		third := ""
		for _, other := range pairs {
			if other != c && !strings.EqualFold(other.wrongText, c.wrongText) && !strings.EqualFold(other.wrongText, c.rightText) {
				third = other.wrongText
				break
			}
		}
		if third == "" {
			for _, w := range out.wrong {
				if w.wrongText != "" && !strings.EqualFold(w.wrongText, c.wrongText) && !strings.EqualFold(w.wrongText, c.rightText) {
					third = w.wrongText
					break
				}
			}
		}
		if third == "" {
			continue
		}
		setOptions(&c.q, []string{c.rightText, c.wrongText, third}, rnd)
		out.wrong = append(out.wrong, &c.spotCandidate)
	}
	for i, ex := range p.Examples {
		if strings.ContainsAny(ex, "()/") || strings.Contains(ex, blank) {
			continue
		}
		out.right = append(out.right, &spotCandidate{key: fmt.Sprintf("%s/example%d", p.PointID, i), q: SpotQuestion{
			Tokens: tokenize(ex), RuleVi: p.HintVi, TopicVi: p.TitleVi, PointID: p.PointID, Source: SourceChoice,
		}})
	}
	return out
}

// levelIndex orders CEFR levels; unknown ones go last.
func levelIndex(level string) int {
	for i, l := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		if l == level {
			return i
		}
	}
	return 99
}

// GrammarRound builds a "Bắn lỗi" round (FR-023–FR-026): 10 sentences, about 3 without a mistake,
// from the points the learner has started first, then other points of their level, then lower
// levels, then the rest.
func (s *Service) GrammarRound(ctx context.Context, userID string) ([]SpotQuestion, error) {
	return s.spotQuestions(ctx, userID, spotQuestions, spotRight, s.newRand())
}

func (s *Service) spotQuestions(ctx context.Context, userID string, n, rightCount int, rnd *rand.Rand) ([]SpotQuestion, error) {
	sources, err := s.d.Grammar.PublishedSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("games: grammar sources: %w", err)
	}
	items, err := s.d.Spots.Approved(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("games: spot items: %w", err)
	}
	started, err := s.d.Grammar.StartedPoints(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: grammar progress: %w", err)
	}
	level, err := s.d.Study.Level(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("games: level: %w", err)
	}
	byPoint := map[string][]SpotItem{}
	for _, it := range items {
		byPoint[it.PointID] = append(byPoint[it.PointID], it)
	}
	isStarted := map[string]bool{}
	for _, id := range started {
		isStarted[id] = true
		if p := sourceByID(sources, id); p != nil && levelIndex(p.Level) > levelIndex(level) {
			level = p.Level
		}
	}
	if level == "" {
		level = "A1"
	}
	tier := func(p PointSource) int {
		switch {
		case isStarted[p.PointID]:
			return 0
		case p.Level == level:
			return 1
		case levelIndex(p.Level) < levelIndex(level):
			return 2
		}
		return 3
	}
	var tiers [4]pointCandidates
	for _, p := range sources {
		c := candidatesOf(p, byPoint[p.PointID], rnd)
		t := tier(p)
		tiers[t].wrong = append(tiers[t].wrong, c.wrong...)
		tiers[t].right = append(tiers[t].right, c.right...)
	}
	used := map[string]bool{}
	var out []SpotQuestion
	take := func(list []*spotCandidate, max int) {
		for _, c := range list {
			if max <= 0 {
				return
			}
			if used[c.key] {
				continue
			}
			used[c.key] = true
			out = append(out, c.q)
			max--
		}
	}
	for i := range tiers {
		shuffle(rnd, tiers[i].wrong)
		shuffle(rnd, tiers[i].right)
	}
	pick := func(wrongs bool, want int) {
		for i := range tiers {
			have := len(out)
			if wrongs {
				take(tiers[i].wrong, want)
			} else {
				take(tiers[i].right, want)
			}
			want -= len(out) - have
		}
	}
	pick(true, n-rightCount)
	pick(false, n-len(out))
	if len(out) < n {
		pick(true, n-len(out))
	}
	if len(out) < n {
		return nil, notEnoughSentences()
	}
	shuffle(rnd, out)
	return out, nil
}

func sourceByID(sources []PointSource, id string) *PointSource {
	for i := range sources {
		if sources[i].PointID == id {
			return &sources[i]
		}
	}
	return nil
}
