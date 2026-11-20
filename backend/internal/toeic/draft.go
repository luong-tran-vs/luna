package toeic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

// GroupWriter writes new groups with AI (ai.Provider in internal/service).
type GroupWriter interface {
	ToeicGroups(ctx context.Context, req ai.ToeicGroupsRequest) (string, error)
}

// SourceAI marks the groups the AI wrote.
const SourceAI = "ai"

// draftTimeout bounds one AI request for groups: long texts take a while.
const draftTimeout = 4 * time.Minute

// DraftQuestions is about how many questions one AI request writes: Part 1, 2, 5 one per group;
// Part 3/4 four groups of 3; Part 6 three texts of 4; Part 7 groups of 2-5 adding up to 10.
const DraftQuestions = 10

// draftGroups is how many groups one request asks for in each part (index = part); Part 7 asks for
// DraftQuestions questions instead and gets 2 to 5 groups.
var draftGroups = [8]int{0, 10, 10, 4, 4, 10, 3, 0}

// draftMaxGroups bounds the groups kept from one answer (Part 7: 10 questions in groups of 2 or more).
var draftMaxGroups = [8]int{0, 10, 10, 4, 4, 10, 3, 5}

// tagHints are the question types in English, for the AI prompt.
var tagHints = map[string]string{
	"photo-people": "photo with people", "photo-objects": "photo of objects", "photo-scene": "photo of a place",
	"q-wh": "WH question", "q-yesno": "yes/no question", "q-choice": "choice question (A or B)",
	"q-request": "request or offer", "q-statement": "statement", "indirect-answer": "indirect answer",
	"main-idea": "main idea or purpose", "detail": "detail", "inference": "inference",
	"speaker-intent": "implied meaning of a quoted sentence", "next-action": "what happens next",
	"graphic": "question about the graphic", "who-where": "who the speakers are or where they are",
	"word-form": "word form", "verb-tense": "verb tense", "verb-form": "verb form (gerund, infinitive, passive)",
	"preposition": "preposition", "conjunction": "conjunction", "pronoun": "pronoun", "relative": "relative clause",
	"comparison": "comparison", "vocabulary": "vocabulary", "collocation": "collocation",
	"sentence-insert": "sentence insertion", "not-true": "NOT true", "vocabulary-in-context": "word meaning in context",
	"cross-reference": "needs two texts",
}

// DraftInput asks the AI for new groups of one part.
type DraftInput struct {
	Part  Part   `json:"part"`
	Level int    `json:"level"`
	Topic string `json:"topic"`
}

// DraftReport says what one AI request gave: the drafts kept and the groups dropped with why.
type DraftReport struct {
	Created []string      `json:"created"`
	Dropped []GroupReport `json:"dropped"`
}

// errDraftAI wraps the failures of the AI itself, so the handler can say the AI is unavailable.
var errDraftAI = errors.New("toeic: AI draft failed")

// DraftGroups asks the AI for new groups of one part in one request, checks each one exactly like an
// import and keeps the valid ones as drafts (source "ai") for the admin to preview and publish.
func (s *Service) DraftGroups(ctx context.Context, in DraftInput) (DraftReport, error) {
	rep := DraftReport{Created: []string{}, Dropped: []GroupReport{}}
	f := map[string]string{}
	if !in.Part.Valid() {
		f["part"] = "Part phải từ 1 đến 7"
	}
	if !slices.Contains(Levels, in.Level) {
		f["level"] = "Mức phải là 450, 600, 750 hoặc 900"
	}
	if len([]rune(in.Topic)) > 200 {
		f["topic"] = "Gợi ý tối đa 200 ký tự"
	}
	if len(f) > 0 {
		return rep, &ValidationError{Fields: f}
	}
	if s.d.Writer == nil {
		return rep, fmt.Errorf("%w: %w", errDraftAI, ai.ErrNotConfigured)
	}
	existing, err := s.d.Groups.List(ctx, GroupFilter{Part: in.Part})
	if err != nil {
		return rep, err
	}
	req := ai.ToeicGroupsRequest{Part: int(in.Part), Count: draftGroups[in.Part], Level: in.Level, Topic: strings.TrimSpace(in.Topic)}
	if in.Part == 7 {
		req.Questions = DraftQuestions
	}
	for _, t := range Tags {
		if slices.Contains(t.Parts, in.Part) {
			req.Tags = append(req.Tags, ai.ToeicTag{Code: t.Code, Label: tagHints[t.Code]})
		}
	}
	for i := len(existing) - 1; i >= 0 && len(req.Existing) < 40; i-- {
		req.Existing = append(req.Existing, groupTitle(existing[i]))
	}

	actx, cancel := context.WithTimeout(ctx, draftTimeout)
	defer cancel()
	text, err := s.d.Writer.ToeicGroups(actx, req)
	if err != nil {
		return rep, fmt.Errorf("%w: %w", errDraftAI, err)
	}
	var out struct {
		Groups []Group `json:"groups"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return rep, fmt.Errorf("%w: decode: %w", errDraftAI, err)
	}

	stamp := s.now().Format("060102-1504")
	taken := map[string]bool{}
	for _, g := range existing {
		taken[g.Key] = true
	}
	var keep []Group
	dups := newDupFinder(existing)
	// The part is the one asked for; a Part 5 group written with several questions becomes one each.
	for i := range out.Groups {
		out.Groups[i].Part = in.Part
	}
	out.Groups = splitSingles(out.Groups)
	for i, g := range out.Groups {
		if i >= draftMaxGroups[in.Part] {
			break
		}
		g.Key = nextDraftKey(in.Part, stamp, i+1, taken)
		g.Part, g.Level, g.Source, g.Status, g.Version = in.Part, in.Level, SourceAI, StatusDraft, 1
		g.CreatedAt, g.UpdatedAt = s.now(), s.now()
		Normalize(&g)
		if g.Part != 1 {
			g.Scene = "" // only a Part 1 photo is drawn from a scene
		}
		repairKinds(&g)
		dropHeaderBlocks(&g)
		repairEvidence(&g)
		r := GroupReport{Key: g.Key, Part: g.Part, Questions: len(g.Questions), Status: ReportOK}
		errs := validateGroup(g)
		for k, v := range notEnglish(g) {
			errs[k] = v
		}
		if len(errs) == 0 {
			// A group the part already has, or one written twice in this answer, is not kept.
			if key, alike, dup := dups.find(g); dup {
				errs["duplicate"] = dupMessage(key, alike)
			}
		}
		if len(errs) > 0 {
			r.Status, r.Fields = ReportError, errs
			rep.Dropped = append(rep.Dropped, r)
			continue
		}
		taken[g.Key] = true
		dups.add(g)
		keep = append(keep, g)
	}
	if len(keep) > 0 {
		created, _, err := s.d.Groups.Upsert(ctx, keep)
		if err != nil {
			return rep, err
		}
		rep.Created = append(rep.Created, created...)
	}
	return rep, nil
}

// nextDraftKey is a free key for an AI draft: "p5-ai-261120-0930-1", with a suffix when taken.
func nextDraftKey(p Part, stamp string, n int, taken map[string]bool) string {
	key := fmt.Sprintf("p%d-ai-%s-%d", p, stamp, n)
	for i := 2; taken[key]; i++ {
		key = fmt.Sprintf("p%d-ai-%s-%d-%d", p, stamp, n, i)
	}
	return key
}
