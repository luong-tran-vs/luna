package grammar

import (
	"slices"
	"testing"
)

func TestPatternPieces(t *testing.T) {
	t.Parallel()
	got := PatternPieces(" S + am / is / are +  V-ing ")
	if want := []string{"S", "am / is / are", "V-ing"}; !slices.Equal(got, want) {
		t.Errorf("pieces = %q, want %q", got, want)
	}
	got = PatternPieces("Am + I ...? / Is + he, she, it ...? -> Yes, I am. / No, I'm not.")
	if want := []string{"Am", "I ...?", "Is", "he, she, it ...?"}; !slices.Equal(got, want) {
		t.Errorf("pieces = %q, want %q", got, want)
	}
	got = PatternPieces("I + am / he, she, it + is / you, we, they + are + ...")
	if want := []string{"I", "am", "he, she, it", "is", "you, we, they", "are", "..."}; !slices.Equal(got, want) {
		t.Errorf("pieces = %q, want %q", got, want)
	}
	got = PatternPieces("S + am / is / are + danh từ hoặc tính từ")
	if want := []string{"S", "am / is / are", "danh từ hoặc tính từ"}; !slices.Equal(got, want) {
		t.Errorf("pieces = %q, want %q", got, want)
	}
}

func TestValidateChecksTheStructureAndMarksOfExamples(t *testing.T) {
	t.Parallel()
	// The fixture has one structure, "S + am/is/are": two pieces.
	tests := []struct {
		name      string
		structure int
		marked    string
		key, msg  string
	}{
		{name: "unmarked old example", key: ""},
		{name: "structure only", structure: 1, key: ""},
		{name: "marked", structure: 1, marked: "{1:I} {2:am} a student.", key: ""},
		{name: "unknown structure", structure: 2, key: "structure", msg: "Không có cấu trúc số 2"},
		{name: "marks without structure", marked: "{1:I} am a student.", key: "marked", msg: "Câu có đánh dấu phải ghi số cấu trúc"},
		{name: "other sentence", structure: 1, marked: "{1:I} {2:was} a student.", key: "marked", msg: "Bỏ dấu đi phải ra đúng câu tiếng Anh"},
		{name: "piece too high", structure: 1, marked: "{1:I} {3:am} a student.", key: "marked", msg: "Số đánh dấu lớn hơn số phần của công thức"},
		{name: "unclosed", structure: 1, marked: "{1:I am a student.", key: "marked", msg: "Đánh dấu sai cú pháp, phải là {số:từ}"},
		{name: "no number", structure: 1, marked: "{I} am a student.", key: "marked", msg: "Đánh dấu sai cú pháp, phải là {số:từ}"},
		{name: "empty words", structure: 1, marked: "{1:}I am a student.", key: "marked", msg: "Đánh dấu sai cú pháp, phải là {số:từ}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := validContent()
			c.Examples[0].Structure, c.Examples[0].Marked = tt.structure, tt.marked
			f := Validate(c)
			if tt.key == "" {
				if f != nil {
					t.Fatalf("rejected: %v", f)
				}
				return
			}
			if got := f["content.examples[0]."+tt.key]; got != tt.msg {
				t.Errorf("message = %q, want %q (all: %v)", got, tt.msg, f)
			}
		})
	}
}

func TestFilterGeneratedDropsOnlyWrongMarks(t *testing.T) {
	t.Parallel()
	c := validContent()
	c.Examples[0].Structure, c.Examples[0].Marked = 1, "{1:I} {2:am} a student."
	c.Examples[1].Structure, c.Examples[1].Marked = 1, "{1:She} {2:is} a doctor."
	c.Examples[2].Structure, c.Examples[2].Marked = 5, "{1:They} {2:are} friends."
	c = FilterGenerated(c)

	if e := c.Examples[0]; e.Structure != 1 || e.Marked != "{1:I} {2:am} a student." {
		t.Errorf("good example changed: %+v", e)
	}
	if e := c.Examples[1]; e.Structure != 1 || e.Marked != "" {
		t.Errorf("marks of another sentence kept: %+v", e)
	}
	if e := c.Examples[2]; e.Structure != 0 || e.Marked != "" || e.En != "They are friends." {
		t.Errorf("unknown structure kept: %+v", e)
	}
}

func TestFilterGeneratedForgetsStructuresOnceOneIsDropped(t *testing.T) {
	t.Parallel()
	c := validContent()
	c.Structures = append([]Structure{{Label: "Hỏng", Pattern: "", Example: "x"}}, c.Structures...)
	c.Examples[0].Structure, c.Examples[0].Marked = 2, "{1:I} {2:am} a student."
	c = FilterGenerated(c)
	if e := c.Examples[0]; e.Structure != 0 || e.Marked != "" {
		t.Errorf("example still points at a renumbered structure: %+v", e)
	}
}
