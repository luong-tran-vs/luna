package lesson

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxDistractorLen bounds one wrong tile.
const maxDistractorLen = 50

var (
	// ErrNoPractice means the lesson has no practice to edit.
	ErrNoPractice = errors.New("lesson: lesson has no practice")
	// ErrPracticeChanged means the practice changed while an admin edited it.
	ErrPracticeChanged = errors.New("lesson: practice changed")
)

// TranslationInput is one translation sentence as edited by an admin.
type TranslationInput struct {
	Vi          string
	En          string
	Distractors []string
}

// ValidateTranslations applies the rules of cleanTranslations but reports each problem by field
// ("translations.1.en") instead of dropping the sentence.
func ValidateTranslations(in []TranslationInput) ([]Translation, error) {
	fields := map[string]string{}
	if len(in) > maxTranslations {
		fields["translations"] = fmt.Sprintf("Tối đa %d câu dịch", maxTranslations)
	}
	out := make([]Translation, 0, len(in))
	seen := map[string]bool{}
	for i, it := range in {
		key := "translations." + strconv.Itoa(i) + "."
		vi, en := collapse(it.Vi), collapse(it.En)
		switch n := utf8.RuneCountInString(vi); {
		case n == 0:
			fields[key+"vi"] = "Vui lòng nhập câu tiếng Việt"
		case n > maxPracticeText:
			fields[key+"vi"] = fmt.Sprintf("Tối đa %d ký tự", maxPracticeText)
		}
		tiles := answerTiles(en)
		switch n := utf8.RuneCountInString(en); {
		case n == 0:
			fields[key+"en"] = "Vui lòng nhập câu tiếng Anh"
		case n > maxPracticeText:
			fields[key+"en"] = fmt.Sprintf("Tối đa %d ký tự", maxPracticeText)
		case len(tiles) < minAnswerTiles || len(tiles) > maxAnswerTiles:
			fields[key+"en"] = fmt.Sprintf("Câu tiếng Anh cần từ %d đến %d ô chữ", minAnswerTiles, maxAnswerTiles)
		case seen[textKey(en)]:
			fields[key+"en"] = "Câu bị trùng"
		}
		seen[textKey(en)] = true

		taken := map[string]bool{}
		for _, t := range tiles {
			taken[strings.ToLower(t)] = true
			taken[strings.ToLower(strings.TrimFunc(t, isEdgePunct))] = true
		}
		distractors := make([]string, 0, len(it.Distractors))
		for _, d := range it.Distractors {
			d = collapse(d)
			switch k := strings.ToLower(d); {
			case d == "":
				fields[key+"distractors"] = "Từ gây nhiễu không được để trống"
			case utf8.RuneCountInString(d) > maxDistractorLen:
				fields[key+"distractors"] = fmt.Sprintf("Mỗi từ gây nhiễu tối đa %d ký tự", maxDistractorLen)
			case taken[k]:
				fields[key+"distractors"] = "Từ gây nhiễu bị trùng hoặc trùng với đáp án"
			default:
				taken[k] = true
			}
			distractors = append(distractors, d)
		}
		if len(distractors) > maxDistractors {
			fields[key+"distractors"] = fmt.Sprintf("Tối đa %d từ gây nhiễu", maxDistractors)
		}
		out = append(out, Translation{Vi: vi, En: en, Distractors: distractors})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return out, nil
}

// UpdateTranslations replaces the translation sentences of the practice with an admin-edited list
// (an empty list removes them). The other parts of the practice stay.
func (s *Service) UpdateTranslations(ctx context.Context, id string, in []TranslationInput) (Lesson, error) {
	l, err := s.Lessons.Get(ctx, id)
	if err != nil {
		return Lesson{}, err
	}
	if l.PracticeStatus == StatusRunning {
		return Lesson{}, ErrPracticeRunning
	}
	if l.Practice == nil {
		return Lesson{}, ErrNoPractice
	}
	ts, err := ValidateTranslations(in)
	if err != nil {
		return Lesson{}, err
	}
	p := *l.Practice
	p.Translations = ts
	ok, err := s.Lessons.SavePractice(ctx, id, l.Revision, l.PracticeVersion, p)
	if err != nil {
		return Lesson{}, fmt.Errorf("lesson: save practice: %w", err)
	}
	if !ok {
		return Lesson{}, ErrPracticeChanged
	}
	s.keepReview(ctx, l, AreaTranslation, translationKeys(l.Practice.Translations), translationKeys(ts))
	return s.Lessons.Get(ctx, id)
}
