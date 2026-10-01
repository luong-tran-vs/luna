package lesson

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
)

// Limits of a generation request (F7).
const (
	minGenerateCount = 1
	maxGenerateCount = 5
	minGenerateWords = 50
	maxGenerateWords = 800
	maxGenerateIdea  = 500
	// generateTimeout bounds the AI call of one batch.
	generateTimeout = 60 * time.Second
	// wordTolerance is how far a draft may be from the requested length.
	wordTolerance = 0.2
)

// errGenerateAI wraps every error of the AI call so handlers can tell it from storage errors.
var errGenerateAI = errors.New("lesson: generate")

// GenerateInput is what an admin asks for when generating lessons for a topic.
type GenerateInput struct {
	Count int
	Words int
	Kind  string
	Idea  string
}

// Draft is a generated lesson that passed the checks; it is never stored.
type Draft struct {
	Title   string
	Content string
	Words   int
}

// GenerateResult is one batch of drafts. Dropped counts drafts the AI returned that were
// rejected; Requested - len(Drafts) can be larger when the AI returned fewer.
type GenerateResult struct {
	Drafts    []Draft
	Requested int
	Dropped   int
}

// ValidateGenerate trims the idea and checks the limits.
func ValidateGenerate(in GenerateInput) (GenerateInput, error) {
	in.Kind = strings.TrimSpace(in.Kind)
	in.Idea = strings.TrimSpace(in.Idea)
	fields := map[string]string{}
	if in.Count < minGenerateCount || in.Count > maxGenerateCount {
		fields["count"] = fmt.Sprintf("Số bài từ %d đến %d", minGenerateCount, maxGenerateCount)
	}
	if in.Words < minGenerateWords || in.Words > maxGenerateWords {
		fields["words"] = fmt.Sprintf("Độ dài từ %d đến %d từ", minGenerateWords, maxGenerateWords)
	}
	if !ai.ValidKind(in.Kind) {
		fields["kind"] = "Dạng bài không hợp lệ"
	}
	if utf8.RuneCountInString(in.Idea) > maxGenerateIdea {
		fields["idea"] = fmt.Sprintf("Ý chính tối đa %d ký tự", maxGenerateIdea)
	}
	if len(fields) > 0 {
		return in, &ValidationError{Fields: fields}
	}
	return in, nil
}

// Generate asks the AI for lesson drafts for a topic in a single request and returns those
// that pass the checks. Nothing is stored.
func (s *Service) Generate(ctx context.Context, topicID string, in GenerateInput) (GenerateResult, error) {
	in, err := ValidateGenerate(in)
	if err != nil {
		return GenerateResult{}, err
	}
	topic, err := s.Topics.Get(ctx, topicID)
	if err != nil {
		return GenerateResult{}, err
	}
	existing, err := s.Lessons.List(ctx, Filter{TopicID: topic.ID})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("lesson: topic lessons: %w", err)
	}
	titles := make([]string, len(existing))
	for i, l := range existing {
		titles[i] = l.Title
	}

	timeout := s.GenerateTimeout
	if timeout == 0 {
		timeout = generateTimeout
	}
	actx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	drafts, err := s.AI.GenerateLessons(actx, ai.GenerateRequest{
		Level: string(topic.Level), TopicName: topic.Name, Count: in.Count, Words: in.Words,
		Kind: ai.LessonKind(in.Kind), Idea: in.Idea, ExistingTitles: titles,
	})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("%w: %w", errGenerateAI, err)
	}

	out := filterDrafts(drafts, titles, in.Count, in.Words)
	if len(out.Drafts) == 0 {
		return GenerateResult{}, ErrUnusableDraft
	}
	return out, nil
}

// filterDrafts drops empty drafts, repeated titles (within the batch or of existing lessons),
// drafts outside ±20% of words and drafts over the lesson limits, then keeps at most count.
func filterDrafts(drafts []ai.LessonDraft, existing []string, count, words int) GenerateResult {
	seen := map[string]bool{}
	for _, t := range existing {
		seen[titleKey(t)] = true
	}
	low := int(math.Ceil(float64(words) * (1 - wordTolerance)))
	high := int(math.Floor(float64(words) * (1 + wordTolerance)))

	out := GenerateResult{Drafts: []Draft{}, Requested: count}
	for _, d := range drafts {
		title := strings.TrimSpace(d.Title)
		content := strings.TrimSpace(strings.ReplaceAll(d.Content, "\r\n", "\n"))
		n := len(strings.Fields(content))
		key := titleKey(title)
		switch {
		case title == "" || content == "",
			utf8.RuneCountInString(title) > maxTitle || utf8.RuneCountInString(content) > maxContent,
			seen[key],
			n < low || n > high:
			out.Dropped++
			continue
		}
		seen[key] = true
		if len(out.Drafts) < count {
			out.Drafts = append(out.Drafts, Draft{Title: title, Content: content, Words: n})
		}
	}
	return out
}

// titleKey compares titles ignoring case, extra spaces and trailing punctuation.
func titleKey(title string) string {
	key := strings.ToLower(strings.Join(strings.Fields(title), " "))
	return strings.TrimRightFunc(key, unicode.IsPunct)
}
