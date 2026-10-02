package lesson

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luongtran/luna/backend/internal/ai"
	"github.com/luongtran/luna/backend/internal/wordmatch"
)

// Limits of a generation request (F7).
const (
	minGenerateCount = 1
	maxGenerateCount = 10
	minGenerateWords = 50
	maxGenerateWords = 800
	maxGenerateIdea  = 500
	// generateTimeout bounds the AI call of one batch.
	generateTimeout = 120 * time.Second
)

// errGenerateAI wraps every error of the AI call so handlers can tell it from storage errors.
var errGenerateAI = errors.New("lesson: generate")

// GenerateInput is what an admin asks for when generating lessons for a topic.
type GenerateInput struct {
	Count int
	Words int
	Kind  string
	Idea  string
	// TargetWords lists the topic words each lesson must use (F18): empty, or one group per lesson.
	TargetWords [][]string
}

// Draft is a generated lesson that passed the checks; it is never stored.
type Draft struct {
	Title   string
	Content string
	Words   int
	// TargetWords are the words asked for this draft; MissingWords those its content lacks (F18).
	TargetWords  []string
	MissingWords []string
	// index is the position of the draft in the AI answer, which matches its target group.
	index int
}

// GenerateResult is one batch of drafts. Dropped counts drafts the AI returned that were
// rejected, by reason in DropReasons; Requested - len(Drafts) can be larger when the AI returned
// fewer.
type GenerateResult struct {
	Drafts      []Draft
	Requested   int
	Dropped     int
	DropReasons DropReasons
}

// DropReasons counts rejected drafts by reason. A draft off the asked length is kept: the admin
// sees a warning and decides.
type DropReasons struct {
	// DuplicateTitle: the title repeats another draft of the batch or a lesson of the topic.
	DuplicateTitle int
	// Empty: no title or no content.
	Empty int
	// TooLong: the title or content is over the lesson limits.
	TooLong int
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
	targets, err := cleanTargets(in.TargetWords, in.Count, topic.Words)
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
		Kind: ai.LessonKind(in.Kind), Idea: in.Idea, ExistingTitles: titles, TargetWords: targets,
	})
	if err != nil {
		return GenerateResult{}, fmt.Errorf("%w: %w", errGenerateAI, err)
	}

	out := filterDrafts(drafts, titles, in.Count)
	if len(out.Drafts) == 0 {
		return GenerateResult{}, ErrUnusableDraft
	}
	markTargets(out.Drafts, targets)
	return out, nil
}

// filterDrafts drops empty drafts, repeated titles (within the batch or of existing lessons) and
// drafts over the lesson limits, counting each reason, then keeps at most count. Drafts off the
// asked length are kept (the admin page warns about them).
func filterDrafts(drafts []ai.LessonDraft, existing []string, count int) GenerateResult {
	seen := map[string]bool{}
	for _, t := range existing {
		seen[titleKey(t)] = true
	}

	out := GenerateResult{Drafts: []Draft{}, Requested: count}
	for i, d := range drafts {
		title := strings.TrimSpace(d.Title)
		content := strings.TrimSpace(strings.ReplaceAll(d.Content, "\r\n", "\n"))
		n := len(strings.Fields(content))
		key := titleKey(title)
		reason := &out.DropReasons
		switch {
		case title == "" || content == "":
			reason.Empty++
		case utf8.RuneCountInString(title) > maxTitle || utf8.RuneCountInString(content) > maxContent:
			reason.TooLong++
		case seen[key]:
			reason.DuplicateTitle++
		default:
			reason = nil
		}
		if reason != nil {
			out.Dropped++
			continue
		}
		seen[key] = true
		if len(out.Drafts) < count {
			out.Drafts = append(out.Drafts, Draft{Title: title, Content: content, Words: n, index: i})
		}
	}
	return out
}

// titleKey compares titles ignoring case, extra spaces and trailing punctuation.
func titleKey(title string) string {
	key := strings.ToLower(strings.Join(strings.Fields(title), " "))
	return strings.TrimRightFunc(key, unicode.IsPunct)
}

// maxTargetWords is the largest target word group of one lesson (F18).
const maxTargetWords = 15

// cleanTargets checks the target word groups of a generation against the topic words and
// rewrites each word with the topic's spelling. No groups gives nil (generate as before F18).
func cleanTargets(groups [][]string, count int, topicWords []string) ([][]string, error) {
	if len(groups) == 0 {
		return nil, nil
	}
	if len(groups) != count {
		return nil, &ValidationError{Fields: map[string]string{"targetWords": "Số nhóm từ phải bằng số bài"}}
	}
	known := make(map[string]string, len(topicWords))
	for _, w := range topicWords {
		known[strings.ToLower(w)] = w
	}
	fields := map[string]string{}
	out := make([][]string, len(groups))
	for i, g := range groups {
		if len(g) > maxTargetWords {
			fields[fmt.Sprintf("targetWords.%d", i)] = fmt.Sprintf("Tối đa %d từ mỗi bài", maxTargetWords)
			continue
		}
		out[i] = []string{}
		seen := map[string]bool{}
		for j, w := range g {
			key := strings.ToLower(strings.Join(strings.Fields(w), " "))
			word, ok := known[key]
			switch {
			case !ok:
				fields[fmt.Sprintf("targetWords.%d.%d", i, j)] = "Từ không có trong danh sách của chủ đề"
			case seen[key]:
				fields[fmt.Sprintf("targetWords.%d.%d", i, j)] = "Từ bị trùng trong bài"
			default:
				seen[key] = true
				out[i] = append(out[i], word)
			}
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return out, nil
}

// markTargets sets each draft's target words (by its position in the AI answer) and the ones
// its content lacks.
func markTargets(drafts []Draft, targets [][]string) {
	for i := range drafts {
		d := &drafts[i]
		d.TargetWords, d.MissingWords = []string{}, []string{}
		if d.index >= len(targets) {
			continue
		}
		d.TargetWords = targets[d.index]
		content := wordmatch.New(d.Content, nil)
		for _, w := range d.TargetWords {
			if !content.Contains(w) {
				d.MissingWords = append(d.MissingWords, w)
			}
		}
	}
}
