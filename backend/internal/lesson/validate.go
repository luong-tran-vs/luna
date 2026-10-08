package lesson

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxTitle     = 200
	maxContent   = 10000
	maxSource    = 200
	maxLicense   = 100
	maxSentences = 200
)

// Input is what an admin submits to create or edit a lesson.
type Input struct {
	Title   string
	Content string
	TopicID string
	// Level is the lesson's own level; a topic is shared by every level.
	Level   Level
	Source  string
	License string
	// GrammarPointID is an optional syllabus point of the lesson's level.
	GrammarPointID string
	// KeepGrammarPoint makes Update keep the lesson's current point and ignore GrammarPointID (the
	// request left the field out). Create ignores it.
	KeepGrammarPoint bool
	// AppendToRoadmap adds a new lesson at the end of its topic roadmap (F7); Update ignores it.
	AppendToRoadmap bool
	// Images are the word pictures of a new lesson (F23); nil leaves them off. Update ignores it.
	Images *ImageInput
	// TargetWords are the topic words a generated draft was asked to use (F18); at most
	// maxTargetWords. Update ignores it.
	TargetWords []string
	// Draft saves a new lesson hidden from learners until it is published. Update ignores it.
	Draft bool
}

// ValidateInput trims every field, checks limits and splits the content. It returns the
// cleaned input and the sentences, or a *ValidationError listing every invalid field.
func ValidateInput(in Input) (Input, []string, error) {
	in = Input{
		Title:            strings.TrimSpace(in.Title),
		Content:          strings.TrimSpace(in.Content),
		TopicID:          strings.TrimSpace(in.TopicID),
		Level:            Level(strings.ToUpper(strings.TrimSpace(string(in.Level)))),
		Source:           strings.TrimSpace(in.Source),
		License:          strings.TrimSpace(in.License),
		GrammarPointID:   strings.TrimSpace(in.GrammarPointID),
		KeepGrammarPoint: in.KeepGrammarPoint,
		AppendToRoadmap:  in.AppendToRoadmap,
		Images:           in.Images,
		TargetWords:      cleanWordList(in.TargetWords),
		Draft:            in.Draft,
	}
	fields := map[string]string{}
	if len(in.TargetWords) > maxTargetWords {
		fields["targetWords"] = fmt.Sprintf("Tối đa %d từ mục tiêu", maxTargetWords)
	}

	required(fields, "title", in.Title, "Vui lòng nhập tiêu đề", maxTitle)
	required(fields, "source", in.Source, "Vui lòng nhập nguồn", maxSource)
	required(fields, "license", in.License, "Vui lòng nhập giấy phép", maxLicense)
	if in.TopicID == "" {
		fields["topicId"] = "Vui lòng chọn chủ đề"
	}
	if !ValidLevel(string(in.Level)) {
		fields["level"] = "Vui lòng chọn trình độ"
	}

	var sentences []string
	switch n := utf8.RuneCountInString(in.Content); {
	case n == 0:
		fields["content"] = "Vui lòng dán nội dung bài"
	case n > maxContent:
		fields["content"] = "Nội dung tối đa 10.000 ký tự"
	default:
		sentences = SplitSentences(in.Content)
		switch {
		case len(sentences) == 0:
			fields["content"] = "Vui lòng dán nội dung bài"
		case len(sentences) > maxSentences:
			fields["content"] = fmt.Sprintf("Bài có %d câu, tối đa %d câu", len(sentences), maxSentences)
		}
	}

	if len(fields) > 0 {
		return in, nil, &ValidationError{Fields: fields}
	}
	return in, sentences, nil
}

func required(fields map[string]string, key, value, emptyMsg string, limit int) {
	switch n := utf8.RuneCountInString(value); {
	case n == 0:
		fields[key] = emptyMsg
	case n > limit:
		fields[key] = fmt.Sprintf("Tối đa %d ký tự", limit)
	}
}
