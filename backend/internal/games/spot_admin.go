package games

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// The admin side of "Bắn lỗi" (FR-026a, FR-026b): how many sentences each point has, and extra
// sentences the AI writes on request, kept as drafts until an admin approves them.

// spotGenerateTimeout bounds the AI request; it is a short answer.
const spotGenerateTimeout = 2 * time.Minute

// maxSpotRule bounds the rule of an extra sentence.
const maxSpotRule = 300

// errSpotAI wraps a failed AI request so the handler can tell it from a storage error.
var errSpotAI = errors.New("games: spot AI")

// IsAIError reports whether err is a failure of the AI request.
func IsAIError(err error) bool { return errors.Is(err, errSpotAI) }

// PointSpotCount is how many "Bắn lỗi" sentences a point with a published lesson gives, by source.
type PointSpotCount struct {
	PointID string
	Level   string
	TitleVi string
	Choice  int
	Fix     int
	Spot    int
	Draft   int
	Total   int
}

// SpotCounts counts the sentences with a mistake each published point gives (FR-026a); Total
// leaves the drafts out, since they are not played.
func (s *Service) SpotCounts(ctx context.Context) ([]PointSpotCount, error) {
	sources, err := s.d.Grammar.PublishedSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("games: grammar sources: %w", err)
	}
	items, err := s.d.Spots.Approved(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("games: spot items: %w", err)
	}
	counts, err := s.d.Spots.Counts(ctx)
	if err != nil {
		return nil, fmt.Errorf("games: spot counts: %w", err)
	}
	byPoint := map[string][]SpotItem{}
	for _, it := range items {
		byPoint[it.PointID] = append(byPoint[it.PointID], it)
	}
	// A fixed seed: a choice sentence shows a wrong option at random, and that may change which
	// fix sentences find a third option; the counts must not change between two page loads.
	rnd := rand.New(rand.NewPCG(1, 2))
	out := make([]PointSpotCount, 0, len(sources))
	for _, p := range sources {
		c := PointSpotCount{PointID: p.PointID, Level: p.Level, TitleVi: p.TitleVi, Draft: counts[p.PointID].Draft}
		for _, w := range candidatesOf(p, byPoint[p.PointID], rnd).wrong {
			switch w.q.Source {
			case SourceChoice:
				c.Choice++
			case SourceSpot:
				c.Spot++
			default:
				c.Fix++
			}
		}
		c.Total = c.Choice + c.Fix + c.Spot
		out = append(out, c)
	}
	return out, nil
}

// SpotItems lists the extra sentences of a point, drafts and approved.
func (s *Service) SpotItems(ctx context.Context, pointID string) ([]SpotItem, error) {
	list, err := s.d.Spots.ListByPoint(ctx, pointID)
	if err != nil {
		return nil, fmt.Errorf("games: spot items: %w", err)
	}
	return list, nil
}

// ValidateSpot checks an extra sentence (FR-026b): 3–20 words, the mistake 1–3 words found exactly
// once, a fix different from it, two different distractors different from the fix, a rule.
func ValidateSpot(it SpotItem) map[string]string {
	fields := map[string]string{}
	words := len(strings.Fields(it.Sentence))
	if words < 3 || words > 20 {
		fields["sentence"] = "Câu cần từ 3 đến 20 từ"
	}
	wrong := tokenize(it.Wrong)
	switch {
	case len(wrong) == 0 || len(wrong) > maxSpotSpan:
		fields["wrong"] = "Chỗ sai cần 1 đến 3 từ"
	case len(findTokens(tokenize(it.Sentence), wrong)) != 1:
		fields["wrong"] = "Chỗ sai phải nằm trong câu đúng một lần"
	}
	right := normalize(it.Right)
	switch {
	case right == "":
		fields["right"] = "Cần chữ đúng"
	case right == normalize(it.Wrong):
		fields["right"] = "Chữ đúng phải khác chỗ sai"
	}
	if len(it.Distractors) != 2 {
		fields["distractors"] = "Cần đúng 2 chữ nhiễu"
	} else {
		a, b := normalize(it.Distractors[0]), normalize(it.Distractors[1])
		if a == "" || b == "" || a == b || a == right || b == right {
			fields["distractors"] = "2 chữ nhiễu phải khác nhau và khác chữ đúng"
		}
	}
	if r := strings.TrimSpace(it.RuleVi); r == "" || utf8.RuneCountInString(r) > maxSpotRule {
		fields["ruleVi"] = fmt.Sprintf("Cần quy tắc ngắn (tối đa %d ký tự)", maxSpotRule)
	}
	return fields
}

// fixedSentence is the sentence with its mistake replaced by the fix, for spotting duplicates.
func fixedSentence(sentence, wrong, right string) string {
	tokens := tokenize(sentence)
	w := tokenize(wrong)
	at := findTokens(tokens, w)
	if len(at) != 1 {
		return normalize(sentence)
	}
	return normalize(joinTokens(append(append(append([]string{}, tokens[:at[0]]...), tokenize(right)...), tokens[at[0]+len(w):]...)))
}

// GenerateResult is what one AI request gave: the drafts kept and how many sentences were dropped.
type GenerateResult struct {
	Created []SpotItem
	Dropped int
}

// GenerateSpots asks the AI for up to 10 extra sentences of a published point in one request and
// keeps those that pass ValidateSpot and are not already there, as drafts (FR-026b).
func (s *Service) GenerateSpots(ctx context.Context, pointID string) (GenerateResult, error) {
	if s.d.AI == nil {
		return GenerateResult{}, fmt.Errorf("%w: %w", errSpotAI, ai.ErrNotConfigured)
	}
	sources, err := s.d.Grammar.PublishedSources(ctx)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("games: grammar sources: %w", err)
	}
	p := sourceByID(sources, pointID)
	if p == nil {
		return GenerateResult{}, ErrNotFound
	}
	existing, err := s.d.Spots.ListByPoint(ctx, pointID)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("games: spot items: %w", err)
	}
	seen := map[string]bool{}
	var known []string
	addKnown := func(sentence string) {
		k := normalize(sentence)
		if k != "" && !seen[k] {
			seen[k] = true
			known = append(known, sentence)
		}
	}
	for _, it := range existing {
		addKnown(fixedSentence(it.Sentence, it.Wrong, it.Right))
	}
	for _, r := range candidatesOf(*p, nil, s.newRand()).right {
		addKnown(joinTokens(r.q.Tokens))
	}

	actx, cancel := context.WithTimeout(ctx, spotGenerateTimeout)
	defer cancel()
	got, err := s.d.AI.GrammarSpotItems(actx, ai.GrammarSpotRequest{
		Level: p.Level, TitleEn: p.TitleEn, Pattern: p.Pattern, Examples: p.Examples, Existing: known, Count: maxSpotItems,
	})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("%w: %w", errSpotAI, err)
	}
	now := s.d.Now().UTC()
	var keep []SpotItem
	res := GenerateResult{}
	for i, g := range got {
		if i >= maxSpotItems {
			res.Dropped++
			continue
		}
		it := SpotItem{
			PointID: pointID, Level: p.Level, Sentence: strings.TrimSpace(g.Sentence), Wrong: strings.TrimSpace(g.Wrong),
			Right: strings.TrimSpace(g.Right), RuleVi: strings.TrimSpace(g.RuleVi), Status: SpotDraft, CreatedAt: now, UpdatedAt: now,
		}
		for _, d := range g.Distractors {
			it.Distractors = append(it.Distractors, strings.TrimSpace(d))
		}
		fixed := fixedSentence(it.Sentence, it.Wrong, it.Right)
		if len(ValidateSpot(it)) > 0 || seen[fixed] {
			res.Dropped++
			continue
		}
		seen[fixed] = true
		keep = append(keep, it)
	}
	if len(keep) > 0 {
		if res.Created, err = s.d.Spots.Create(ctx, keep); err != nil {
			return GenerateResult{}, fmt.Errorf("games: save spot items: %w", err)
		}
	}
	if res.Created == nil {
		res.Created = []SpotItem{}
	}
	return res, nil
}

// maxSpotItems bounds the sentences of one AI request.
const maxSpotItems = 10

// SpotEdit is what an admin may change of an extra sentence.
type SpotEdit struct {
	Sentence    string
	Wrong       string
	Right       string
	Distractors []string
	RuleVi      string
}

// UpdateSpot saves an admin's edit after checking it again; the status stays.
func (s *Service) UpdateSpot(ctx context.Context, id string, e SpotEdit) (SpotItem, error) {
	it, err := s.d.Spots.Get(ctx, id)
	if err != nil {
		return SpotItem{}, err
	}
	it.Sentence, it.Wrong, it.Right, it.RuleVi = strings.TrimSpace(e.Sentence), strings.TrimSpace(e.Wrong), strings.TrimSpace(e.Right), strings.TrimSpace(e.RuleVi)
	it.Distractors = nil
	for _, d := range e.Distractors {
		it.Distractors = append(it.Distractors, strings.TrimSpace(d))
	}
	if fields := ValidateSpot(it); len(fields) > 0 {
		return SpotItem{}, &ValidationError{Fields: fields}
	}
	it.UpdatedAt = s.d.Now().UTC()
	if err := s.d.Spots.Update(ctx, it); err != nil {
		return SpotItem{}, err
	}
	return it, nil
}

// ApproveSpot makes a draft playable.
func (s *Service) ApproveSpot(ctx context.Context, id string) (SpotItem, error) {
	it, err := s.d.Spots.Get(ctx, id)
	if err != nil {
		return SpotItem{}, err
	}
	it.Status, it.UpdatedAt = SpotApproved, s.d.Now().UTC()
	if err := s.d.Spots.Update(ctx, it); err != nil {
		return SpotItem{}, err
	}
	return it, nil
}

// DeleteSpot removes an extra sentence.
func (s *Service) DeleteSpot(ctx context.Context, id string) error {
	return s.d.Spots.Delete(ctx, id)
}
