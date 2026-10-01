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
	Source  string
	License string
	// AppendToRoadmap adds a new lesson at the end of its topic roadmap (F7); Update ignores it.
	AppendToRoadmap bool
}

// ValidateInput trims every field, checks limits and splits the content. It returns the
// cleaned input and the sentences, or a *ValidationError listing every invalid field.
func ValidateInput(in Input) (Input, []string, error) {
	in = Input{
		Title:           strings.TrimSpace(in.Title),
		Content:         strings.TrimSpace(in.Content),
		TopicID:         strings.TrimSpace(in.TopicID),
		Source:          strings.TrimSpace(in.Source),
		License:         strings.TrimSpace(in.License),
		AppendToRoadmap: in.AppendToRoadmap,
	}
	fields := map[string]string{}

	required(fields, "title", in.Title, "Vui lòng nhập tiêu đề", maxTitle)
	required(fields, "source", in.Source, "Vui lòng nhập nguồn", maxSource)
	required(fields, "license", in.License, "Vui lòng nhập giấy phép", maxLicense)
	if in.TopicID == "" {
		fields["topicId"] = "Vui lòng chọn chủ đề"
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
