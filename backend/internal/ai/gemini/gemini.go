// Package gemini implements ai.Provider with the Gemini generateContent REST API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/ai"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	// maxResponse bounds the response body; an annotation with questions stays far below it.
	maxResponse = 2 << 20
)

// Client calls Gemini. BaseURL can be replaced in tests.
type Client struct {
	BaseURL string
	apiKey  string
	model   string
	client  *http.Client
	log     *slog.Logger
}

// New returns a Gemini provider. An empty apiKey makes every call return ai.ErrNotConfigured.
func New(apiKey, model string, client *http.Client, log *slog.Logger) *Client {
	return &Client{BaseURL: defaultBaseURL, apiKey: apiKey, model: model, client: client, log: log}
}

var _ ai.Provider = (*Client)(nil)

var stringArray = map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}

// annotateSchema forces an object with the annotations and, optionally, the questions, the
// grammar note and the writing prompt (F15), so a missing part never fails the request.
var annotateSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"annotations": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"text":          map[string]any{"type": "STRING"},
					"lemma":         map[string]any{"type": "STRING"},
					"meaningVi":     map[string]any{"type": "STRING"},
					"sentenceIndex": map[string]any{"type": "INTEGER"},
				},
				"required": []string{"text", "lemma", "meaningVi", "sentenceIndex"},
			},
		},
		"questions": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"prompt":        map[string]any{"type": "STRING"},
					"options":       stringArray,
					"answerIndex":   map[string]any{"type": "INTEGER"},
					"explanationVi": map[string]any{"type": "STRING"},
				},
				"required": []string{"prompt", "options", "answerIndex", "explanationVi"},
			},
		},
		"grammarNote": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"title":    map[string]any{"type": "STRING"},
				"bodyVi":   map[string]any{"type": "STRING"},
				"examples": stringArray,
			},
			"required": []string{"title", "bodyVi", "examples"},
		},
		"writingPrompt": map[string]any{"type": "STRING"},
	},
	"required": []string{"annotations"},
}

// generateSchema forces a JSON array of lesson drafts.
var generateSchema = map[string]any{
	"type": "ARRAY",
	"items": map[string]any{
		"type": "OBJECT",
		"properties": map[string]any{
			"title":   map[string]any{"type": "STRING"},
			"content": map[string]any{"type": "STRING"},
		},
		"required": []string{"title", "content"},
	},
}

var criterionSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"score":     map[string]any{"type": "INTEGER"},
		"commentVi": map[string]any{"type": "STRING"},
	},
	"required": []string{"score", "commentVi"},
}

// gradeSchema forces the four fixed criteria, an overall comment and the corrected text (F8).
var gradeSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"task":          criterionSchema,
		"grammar":       criterionSchema,
		"vocabulary":    criterionSchema,
		"coherence":     criterionSchema,
		"overallVi":     map[string]any{"type": "STRING"},
		"correctedText": map[string]any{"type": "STRING"},
	},
	"required": []string{"task", "grammar", "vocabulary", "coherence", "overallVi", "correctedText"},
}

// explainSchema forces the base form, the meaning in context and a short note (F9).
var explainSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"lemma":     map[string]any{"type": "STRING"},
		"meaningVi": map[string]any{"type": "STRING"},
		"noteVi":    map[string]any{"type": "STRING"},
	},
	"required": []string{"lemma", "meaningVi", "noteVi"},
}

// practiceSchema forces every part of the vocabulary practice (F17); the lesson service drops
// the parts that fail its checks.
var practiceSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"objectiveVi": map[string]any{"type": "STRING"},
		"examples": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"lemma":    map[string]any{"type": "STRING"},
					"sentence": map[string]any{"type": "STRING"},
				},
				"required": []string{"lemma", "sentence"},
			},
		},
		"dialogue": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"speakers": stringArray,
				"turns": map[string]any{
					"type": "ARRAY",
					"items": map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"speaker":   map[string]any{"type": "INTEGER"},
							"text":      map[string]any{"type": "STRING"},
							"meaningVi": map[string]any{"type": "STRING"},
						},
						"required": []string{"speaker", "text", "meaningVi"},
					},
				},
			},
			"required": []string{"speakers", "turns"},
		},
		"grammarTipVi": map[string]any{"type": "STRING"},
		"translations": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"vi":          map[string]any{"type": "STRING"},
					"en":          map[string]any{"type": "STRING"},
					"distractors": stringArray,
				},
				"required": []string{"vi", "en", "distractors"},
			},
		},
	},
	"required": []string{"objectiveVi", "examples", "dialogue", "grammarTipVi", "translations"},
}

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type request struct {
	Contents         []content      `json:"contents"`
	GenerationConfig map[string]any `json:"generationConfig"`
}

type response struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
}

// Annotate sends one generateContent request for the whole lesson: annotations, questions,
// grammar note and writing prompt.
func (c *Client) Annotate(ctx context.Context, req ai.AnnotateRequest) (ai.LessonExtras, error) {
	text, err := c.generate(ctx, "annotate", annotatePrompt(req), annotateSchema, 0.2,
		slog.Int("sentences", len(req.Sentences)), slog.Int("focus_words", len(req.FocusWords)))
	if err != nil {
		return ai.LessonExtras{}, err
	}
	var out ai.LessonExtras
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.LessonExtras{}, fmt.Errorf("gemini: decode annotations: %w", err)
	}
	return out, nil
}

// GenerateLessons sends one generateContent request for the whole batch. A high temperature
// keeps the lessons of a batch different from each other.
func (c *Client) GenerateLessons(ctx context.Context, req ai.GenerateRequest) ([]ai.LessonDraft, error) {
	text, err := c.generate(ctx, "generate", generatePrompt(req), generateSchema, 0.9,
		slog.Int("count", req.Count), slog.Int("words", req.Words))
	if err != nil {
		return nil, err
	}
	var out []ai.LessonDraft
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode drafts: %w", err)
	}
	return out, nil
}

// GradeWriting sends one generateContent request that grades a writing on four criteria.
// Only the word count is logged, never the text.
func (c *Client) GradeWriting(ctx context.Context, req ai.GradeRequest) (ai.Grade, error) {
	text, err := c.generate(ctx, "grade", gradePrompt(req), gradeSchema, 0.3,
		slog.Int("words", len(strings.Fields(req.Text))))
	if err != nil {
		return ai.Grade{}, err
	}
	var out ai.Grade
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.Grade{}, fmt.Errorf("gemini: decode grade: %w", err)
	}
	return out, nil
}

// Explain sends one generateContent request for the meaning of a word or phrase in its
// sentence. Only the word count is logged, never the sentence.
func (c *Client) Explain(ctx context.Context, req ai.ExplainRequest) (ai.Explanation, error) {
	text, err := c.generate(ctx, "explain", explainPrompt(req), explainSchema, 0.2,
		slog.Int("words", len(strings.Fields(req.Text))))
	if err != nil {
		return ai.Explanation{}, err
	}
	var out ai.Explanation
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.Explanation{}, fmt.Errorf("gemini: decode explanation: %w", err)
	}
	return out, nil
}

// Practice sends one generateContent request for the vocabulary practice of a lesson. Only
// counts are logged, never the lesson text.
func (c *Client) Practice(ctx context.Context, req ai.PracticeRequest) (ai.Practice, error) {
	text, err := c.generate(ctx, "practice", practicePrompt(req), practiceSchema, 0.5,
		slog.Int("words", len(req.Words)), slog.Int("sentences", len(req.Sentences)))
	if err != nil {
		return ai.Practice{}, err
	}
	var out ai.Practice
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return ai.Practice{}, fmt.Errorf("gemini: decode practice: %w", err)
	}
	return out, nil
}

// generate sends a JSON-mode request and returns the text of the first candidate.
func (c *Client) generate(ctx context.Context, op, prompt string, schema map[string]any, temperature float64,
	attrs ...slog.Attr,
) (string, error) {
	if c.apiKey == "" {
		return "", ai.ErrNotConfigured
	}

	body, err := json.Marshal(request{
		Contents: []content{{Role: "user", Parts: []part{{Text: prompt}}}},
		GenerationConfig: map[string]any{
			"temperature":      temperature,
			"responseMimeType": "application/json",
			"responseSchema":   schema,
		},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: encode request: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.BaseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logRequest(ctx, op, start, 0, attrs)
		return "", fmt.Errorf("gemini: request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read below
	c.logRequest(ctx, op, start, resp.StatusCode, attrs)

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}
	if err := statusError(resp.StatusCode, raw); err != nil {
		return "", err
	}

	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini: empty response")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}

func statusError(status int, body []byte) error {
	switch {
	case status == http.StatusOK:
		return nil
	case status == http.StatusTooManyRequests:
		return ai.ErrQuota
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ai.ErrInvalidKey
	case status == http.StatusBadRequest && strings.Contains(string(body), "API key"):
		return ai.ErrInvalidKey
	default:
		return fmt.Errorf("gemini: status %d", status)
	}
}

// logRequest records one line per call so requests can be counted. It never logs the key
// or the lesson text.
func (c *Client) logRequest(ctx context.Context, op string, start time.Time, status int, extra []slog.Attr) {
	attrs := []slog.Attr{
		slog.String("provider", "gemini"),
		slog.String("model", c.model),
		slog.String("op", op),
		slog.Int("status", status),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	}
	c.log.LogAttrs(ctx, slog.LevelInfo, "ai request", append(attrs, extra...)...)
}

func annotatePrompt(req ai.AnnotateRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You help Vietnamese learners of English at CEFR level %[1]s.
Read the numbered lesson sentences below and return a JSON object with:

1. annotations: pick 8 to 25 words or phrases worth learning at this level
(phrasal verbs, collocations and idioms count as one item, e.g. "give up").
For each item return:
- text: copied exactly as it appears in the sentence (same spelling and inflection),
- lemma: the dictionary form (e.g. "went" -> "go", "gave up" -> "give up"),
- meaningVi: a short Vietnamese meaning that fits this context,
- sentenceIndex: the number of the sentence containing it.
Do not invent text that is not in the sentences.

2. questions: 3 to 5 multiple-choice questions in English that check understanding of the lesson,
at CEFR %[1]s. Each has a prompt, exactly 4 different options, answerIndex (0 to 3) of the only
correct option, and explanationVi: a short Vietnamese explanation pointing to the lesson.

3. grammarNote: one notable grammar point used in the lesson, for this level:
- title: a short Vietnamese title (e.g. "Thì quá khứ đơn"),
- bodyVi: a short explanation in Vietnamese (at most 120 words),
- examples: 1 to 3 sentences or phrases copied exactly from the lesson that use this point.

4. writingPrompt: one short English writing task related to the lesson, suitable for CEFR %[1]s.

`, req.Level)
	if len(req.FocusWords) > 0 {
		fmt.Fprintf(&b, "Always include each of these topic words or phrases as annotations, with text copied exactly "+
			"as it appears in the sentence; they count toward the 25 items: %s\n\n", strings.Join(req.FocusWords, ", "))
	}
	b.WriteString("Sentences:\n")
	for i, s := range req.Sentences {
		fmt.Fprintf(&b, "%d: %s\n", i, s)
	}
	return b.String()
}

// generatePrompt asks for 0.9–1.1× the target length so drafts stay inside the ±20% the
// lesson service accepts.
func generatePrompt(req ai.GenerateRequest) string {
	low, high := req.Words*9/10, req.Words*11/10
	var b strings.Builder
	fmt.Fprintf(&b, `You write English lessons for Vietnamese learners at CEFR level %s.
Topic: %s.
Write exactly %d different lessons about this topic. Each lesson must have about %d words
(between %d and %d words; count every word of the content).
Use only vocabulary and grammar suitable for CEFR %s.
Each lesson needs a short title (at most 8 words). Titles must be different from each other,
and each lesson must tell a different story or situation.
`, req.Level, req.TopicName, req.Count, req.Words, low, high, req.Level)
	if req.Kind == ai.KindDialogue {
		b.WriteString(`Form: a dialogue between two or three people with first names.
Write one turn per line in the form "Name: what they say". No empty lines, no narration,
no stage directions.
`)
	} else {
		b.WriteString("Form: a reading text in plain paragraphs (no dialogue lines).\n")
	}
	if req.Idea != "" {
		fmt.Fprintf(&b, "Use this idea as inspiration: %s\n", req.Idea)
	}
	for i, words := range req.TargetWords {
		if len(words) > 0 {
			fmt.Fprintf(&b, "Lesson %d must use every one of these words or phrases "+
				"(any natural form, e.g. plural or past tense): %s\n", i+1, strings.Join(words, ", "))
		}
	}
	if len(req.ExistingTitles) > 0 {
		b.WriteString("The topic already has these lessons; do not repeat their titles or stories:\n")
		for _, t := range req.ExistingTitles {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	b.WriteString("Return plain text only, no markdown, no headings inside the content.\n")
	return b.String()
}

func gradePrompt(req ai.GradeRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You are an English teacher grading a short writing by a Vietnamese learner at CEFR level %s.
The learner read the lesson below, then wrote about the writing task.
Grade each criterion with an integer score from 1 to 5 (1 = very weak, 5 = excellent for this level):
- task: does the writing answer the task and use the lesson?
- grammar: accuracy of grammar,
- vocabulary: range and correct use of words,
- coherence: organisation and linking of ideas.
For each criterion write commentVi: a short, specific comment in Vietnamese with one or two examples
quoted from the learner's writing. Also write overallVi: two or three sentences of advice in Vietnamese.
correctedText: the learner's writing with minimal corrections, keeping their ideas and structure.
Plain text only, no markdown.

Writing task: %s

Lesson:
%s

Learner's writing:
%s
`, req.Level, req.Prompt, req.LessonText, req.Text)
	return b.String()
}

func explainPrompt(req ai.ExplainRequest) string {
	return fmt.Sprintf(`You help a Vietnamese learner of English at CEFR level %s.
In the sentence below, explain the words "%s" exactly as they are used in this sentence.
Return:
- lemma: the dictionary form (e.g. "made up for" -> "make up for"),
- meaningVi: a short Vietnamese meaning that fits this sentence,
- noteVi: one short Vietnamese sentence explaining how it is used here.
Plain text only, no markdown.

Sentence: %s
`, req.Level, req.Text, req.Sentence)
}

func practicePrompt(req ai.PracticeRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, `You write vocabulary practice for Vietnamese learners of English at CEFR level %[1]s.
The lesson "%[2]s" and its vocabulary list are below. Everything must use simple English suitable for CEFR %[1]s.
Return a JSON object with:

1. objectiveVi: one Vietnamese sentence starting with "Bạn có thể" that says what the learner can do after this lesson.

2. examples: for each vocabulary item, one short English sentence (at most 12 words) that contains the item
exactly as written in its "text" or "lemma" column. Return the item's lemma with each sentence.

3. dialogue: a natural conversation between two people.
- speakers: exactly two first names, one Vietnamese and one English (e.g. "Minh", "Anna"),
- turns: 6 to 10 turns that alternate between the speakers; speaker is 0 or 1 (index in speakers),
  text is the English line, meaningVi its Vietnamese translation.
Use as many vocabulary items as possible, each written exactly as in the list.

4. grammarTipVi: one or two Vietnamese sentences about one useful way of saying something in the dialogue.

5. translations: 3 to 5 short, simple Vietnamese sentences (vi), each with one correct English translation (en)
that uses at least one vocabulary item, and 3 or 4 distractors: single English words that are not in en
but could tempt the learner.

Plain text only, no markdown.

Lesson sentences:
`, req.Level, req.Title)
	for i, s := range req.Sentences {
		fmt.Fprintf(&b, "%d: %s\n", i, s)
	}
	b.WriteString("\nVocabulary (lemma | text | Vietnamese meaning):\n")
	for _, w := range req.Words {
		fmt.Fprintf(&b, "- %s | %s | %s\n", w.Lemma, w.Text, w.MeaningVi)
	}
	return b.String()
}

// suggestWordsSchema is a plain list of words (F18).
var suggestWordsSchema = map[string]any{
	"type":       "OBJECT",
	"properties": map[string]any{"words": stringArray},
	"required":   []string{"words"},
}

// SuggestWords sends one generateContent request for new core words of a topic. The topic
// service checks and deduplicates the answer.
func (c *Client) SuggestWords(ctx context.Context, req ai.SuggestWordsRequest) ([]string, error) {
	text, err := c.generate(ctx, "suggest_words", suggestWordsPrompt(req), suggestWordsSchema, 0.4,
		slog.Int("count", req.Count), slog.Int("existing", len(req.Existing)))
	if err != nil {
		return nil, err
	}
	var out struct {
		Words []string `json:"words"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("gemini: decode words: %w", err)
	}
	return out.Words, nil
}

func suggestWordsPrompt(req ai.SuggestWordsRequest) string {
	existing := "(none)"
	if len(req.Existing) > 0 {
		existing = strings.Join(req.Existing, ", ")
	}
	return fmt.Sprintf(`You build vocabulary lists for Vietnamese learners of English at CEFR level %s.
Topic: %s
List %d new, common English words or short phrases that belong to this topic and suit this level.
Rules:
- Not in this list (ignoring case): %s
- Dictionary form, lower case except proper nouns, English letters, spaces, hyphens or apostrophes only.
- At most 3 words per phrase; no duplicates; no translations or explanations.
`, req.Level, req.TopicName, req.Count, existing)
}
