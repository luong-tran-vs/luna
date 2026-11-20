package toeic

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9-]{3,48}$`)

// Levels are the difficulty marks of a group (TOEIC scores).
var Levels = []int{450, 600, 750, 900}

var (
	docKinds     = []string{"email", "letter", "memo", "notice", "ad", "article", "web", "invoice", "form", "schedule", "chat", "review"}
	blockKinds   = []string{"p", "h", "list", "table", "msg"}
	graphicKinds = []string{"table", "schedule", "price-list", "floor-plan", "bar-chart"}
)

// Normalize trims the text fields of g and lowercases its key.
func Normalize(g *Group) {
	g.Key = strings.ToLower(strings.TrimSpace(g.Key))
	g.Scene = strings.TrimSpace(g.Scene)
	for i := range g.Script {
		t := &g.Script[i]
		t.Speaker = strings.TrimSpace(t.Speaker)
		t.Accent = Accent(strings.ToLower(strings.TrimSpace(string(t.Accent))))
		t.Text = strings.TrimSpace(t.Text)
		t.TranslateVi = strings.TrimSpace(t.TranslateVi)
	}
	for i := range g.Docs {
		d := &g.Docs[i]
		d.Kind = strings.TrimSpace(d.Kind)
		d.Title = strings.TrimSpace(d.Title)
		d.ImageSlot = strings.TrimSpace(d.ImageSlot)
	}
	for i := range g.Questions {
		q := &g.Questions[i]
		q.Prompt = strings.TrimSpace(q.Prompt)
		for j := range q.Options {
			q.Options[j] = strings.TrimSpace(q.Options[j])
		}
		q.ExplainVi = strings.TrimSpace(q.ExplainVi)
		q.TranslateVi = strings.TrimSpace(q.TranslateVi)
		for j := range q.Tags {
			q.Tags[j] = strings.ToLower(strings.TrimSpace(q.Tags[j]))
		}
	}
}

// Validate checks every group of an import file and returns a ValidationError whose keys are like
// "groups[3].questions[1].answer", nil when all are right.
func Validate(gs []Group) error {
	fields := map[string]string{}
	seen := map[string]int{}
	for i, g := range gs {
		prefix := fmt.Sprintf("groups[%d].", i)
		for k, v := range validateGroup(g) {
			fields[prefix+k] = v
		}
		if j, dup := seen[g.Key]; dup && g.Key != "" {
			fields[prefix+"key"] = fmt.Sprintf("Mã cụm trùng với groups[%d]", j)
		} else {
			seen[g.Key] = i
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// validateGroup returns the errors of one group, keyed by field inside the group.
func validateGroup(g Group) map[string]string {
	f := map[string]string{}
	if !keyPattern.MatchString(g.Key) {
		f["key"] = "Mã cụm gồm 3–48 ký tự: chữ thường, số, gạch ngang"
	}
	if !g.Part.Valid() {
		f["part"] = "Part phải từ 1 đến 7"
		return f
	}
	if !slices.Contains(Levels, g.Level) {
		f["level"] = "Mức độ phải là 450, 600, 750 hoặc 900"
	}

	n := len(g.Questions)
	wantQuestions := func(lo, hi int) {
		if n < lo || n > hi {
			if lo == hi {
				f["questions"] = fmt.Sprintf("Part %d cần đúng %d câu", g.Part, lo)
			} else {
				f["questions"] = fmt.Sprintf("Part %d cần %d–%d câu", g.Part, lo, hi)
			}
		}
	}
	noDocs := func() {
		if len(g.Docs) > 0 {
			f["docs"] = fmt.Sprintf("Part %d không có văn bản", g.Part)
		}
	}
	options := 4
	switch g.Part {
	case 1:
		wantQuestions(1, 1)
		noDocs()
		if g.Scene == "" {
			f["scene"] = "Part 1 cần mô tả cảnh (tiếng Anh) để vẽ ảnh"
		}
	case 2:
		wantQuestions(1, 1)
		noDocs()
		options = 3
		if n == 1 && g.Questions[0].Prompt == "" {
			f["questions[0].prompt"] = "Part 2 cần câu hỏi nói"
		}
		validateScript(f, g.Script, 0, 1)
	case 3:
		wantQuestions(3, 3)
		noDocs()
		if len(g.Script) == 0 {
			f["script"] = "Part 3 cần kịch bản hội thoại"
		} else {
			validateScript(f, g.Script, 2, 3)
		}
	case 4:
		wantQuestions(3, 3)
		noDocs()
		if len(g.Script) == 0 {
			f["script"] = "Part 4 cần kịch bản bài nói"
		} else {
			validateScript(f, g.Script, 1, 1)
		}
	case 5:
		wantQuestions(1, 1)
		noDocs()
		if n == 1 && strings.Count(g.Questions[0].Prompt, "___") != 1 {
			f["questions[0].prompt"] = "Câu Part 5 cần đúng một chỗ trống ___"
		}
	case 6:
		wantQuestions(4, 4)
		if len(g.Docs) != 1 {
			f["docs"] = "Part 6 cần đúng một văn bản"
		}
	case 7:
		wantQuestions(2, 5)
		if len(g.Docs) < 1 || len(g.Docs) > 3 {
			f["docs"] = "Part 7 cần 1–3 văn bản"
		}
	}
	if g.Part >= 6 {
		validateDocs(f, g.Docs)
	}
	if g.Part == 6 && len(g.Docs) == 1 {
		text := docText(g.Docs[0])
		for no := 1; no <= 4; no++ {
			if !strings.Contains(text, "[["+strconv.Itoa(no)+"]]") {
				f["docs[0].body"] = fmt.Sprintf("Văn bản Part 6 thiếu chỗ trống [[%d]]", no)
				break
			}
		}
	}
	if g.Graphic != nil {
		if g.Part != 3 && g.Part != 4 {
			f["graphic"] = "Chỉ Part 3/4 có biểu đồ"
		} else if !slices.Contains(graphicKinds, g.Graphic.Kind) {
			f["graphic.kind"] = "Loại biểu đồ không hợp lệ"
		}
	}

	for j, q := range g.Questions {
		p := fmt.Sprintf("questions[%d].", j)
		if q.No != j+1 {
			f[p+"no"] = fmt.Sprintf("Số câu trong cụm phải là %d", j+1)
		}
		if len(q.Options) != options {
			f[p+"options"] = fmt.Sprintf("Part %d cần %d phương án", g.Part, options)
		} else {
			seen := map[string]bool{}
			for _, o := range q.Options {
				k := strings.ToLower(o)
				if o == "" {
					f[p+"options"] = "Phương án không được để trống"
					break
				}
				if seen[k] {
					f[p+"options"] = "Các phương án không được trùng nhau"
					break
				}
				seen[k] = true
			}
		}
		if q.Answer < 0 || q.Answer >= options {
			f[p+"answer"] = fmt.Sprintf("Đáp án phải từ 0 đến %d", options-1)
		}
		if q.ExplainVi == "" {
			f[p+"explainVi"] = "Cần giải thích tiếng Việt"
		}
		if len(q.Tags) == 0 {
			f[p+"tags"] = "Cần ít nhất một dạng câu"
		}
		for _, tag := range q.Tags {
			if !TagAllowed(tag, g.Part) {
				f[p+"tags"] = fmt.Sprintf("Dạng câu %q không có hoặc không thuộc Part %d", tag, g.Part)
				break
			}
		}
		if slices.Contains(q.Tags, "graphic") && g.Graphic == nil {
			f["graphic"] = "Câu \"Look at the graphic\" cần biểu đồ của cụm"
		}
		for k, ref := range q.Evidence {
			if msg := checkRef(g, ref); msg != "" {
				f[fmt.Sprintf("%sevidence[%d].text", p, k)] = msg
			}
		}
		for k, v := range q.Vocab {
			if strings.TrimSpace(v.Word) == "" || strings.TrimSpace(v.MeaningVi) == "" {
				f[fmt.Sprintf("%svocab[%d]", p, k)] = "Từ vựng cần từ và nghĩa"
			}
		}
	}
	return f
}

// validateScript checks the lines and that there are between lo and hi speakers (lo = 0: no rule).
func validateScript(f map[string]string, script []Turn, lo, hi int) {
	speakers := map[string]bool{}
	for i, t := range script {
		p := fmt.Sprintf("script[%d].", i)
		if t.Speaker == "" {
			f[p+"speaker"] = "Cần người nói"
		}
		if !t.Accent.Valid() {
			f[p+"accent"] = "Giọng phải là us, gb, au hoặc ca"
		}
		if t.Text == "" {
			f[p+"text"] = "Lời nói không được để trống"
		}
		speakers[t.Speaker] = true
	}
	if lo > 0 && len(script) > 0 && (len(speakers) < lo || len(speakers) > hi) {
		if lo == hi {
			f["script"] = fmt.Sprintf("Cần đúng %d người nói", lo)
		} else {
			f["script"] = fmt.Sprintf("Cần %d–%d người nói", lo, hi)
		}
	}
}

func validateDocs(f map[string]string, docs []Doc) {
	for i, d := range docs {
		p := fmt.Sprintf("docs[%d].", i)
		if !slices.Contains(docKinds, d.Kind) {
			f[p+"kind"] = "Loại văn bản không hợp lệ"
		}
		if len(d.Body) == 0 {
			f[p+"body"] = "Văn bản không có nội dung"
		}
		for j, b := range d.Body {
			bp := fmt.Sprintf("%sbody[%d]", p, j)
			switch {
			case !slices.Contains(blockKinds, b.Kind):
				f[bp] = "Loại khối không hợp lệ (p, h, list, table, msg)"
			case b.Kind == "table" && (b.Table == nil || len(b.Table.Columns) == 0):
				f[bp] = "Khối bảng cần table.columns"
			case b.Kind == "list" && len(b.Items) == 0:
				f[bp] = "Khối danh sách cần items"
			case (b.Kind == "p" || b.Kind == "h" || b.Kind == "msg") && strings.TrimSpace(b.Text) == "":
				f[bp] = "Khối không có chữ"
			}
		}
	}
}

// checkRef returns why an evidence does not point at its words, "" when it does.
func checkRef(g Group, r Ref) string {
	if strings.TrimSpace(r.Text) == "" {
		return "Đoạn trích không được để trống"
	}
	if r.Doc < 0 {
		if r.Turn < 0 || r.Turn >= len(g.Script) {
			return "Lượt nói không có trong kịch bản"
		}
		if !strings.Contains(g.Script[r.Turn].Text, r.Text) {
			return "Đoạn trích không có nguyên văn trong lượt nói"
		}
		return ""
	}
	if r.Doc >= len(g.Docs) {
		return "Văn bản không có trong cụm"
	}
	body := g.Docs[r.Doc].Body
	if r.Block < 0 || r.Block >= len(body) {
		return "Khối không có trong văn bản"
	}
	if !strings.Contains(blockText(body[r.Block]), r.Text) {
		return "Đoạn trích không có nguyên văn trong khối"
	}
	return ""
}

// blockText is every word of a block, for searching.
func blockText(b Block) string {
	parts := []string{b.Text}
	parts = append(parts, b.Items...)
	if b.Table != nil {
		parts = append(parts, b.Table.Columns...)
		for _, row := range b.Table.Rows {
			parts = append(parts, row...)
		}
	}
	return strings.Join(parts, "\n")
}

func docText(d Doc) string {
	parts := make([]string, 0, len(d.Body))
	for _, b := range d.Body {
		parts = append(parts, blockText(b))
	}
	return strings.Join(parts, "\n")
}

// --- listening time (research R3) ---

const (
	wordsPerMinute   = 150.0
	directionsSecs   = 40.0
	part12PauseSecs  = 5.0
	part34PauseSecs  = 8.0
	leadWordsPart1   = 13 // "Number 3. Look at the picture marked number 3 in your test book."
	leadWordsPart2   = 2  // "Number 7."
	leadWordsPart34  = 10 // "Questions 41 through 43 refer to the following conversation."
	optionLeadWords  = 1  // "(A)"
	betweenTurnsSecs = 0.5
)

func wordCount(s string) int { return len(strings.Fields(s)) }

// estimateListening returns the minutes the browser voice takes to read the listening groups.
func estimateListening(gs []Group) float64 {
	secs := 0.0
	parts := map[Part]bool{}
	words := 0
	for _, g := range gs {
		if g.Part > 4 || !g.Part.Valid() {
			continue
		}
		parts[g.Part] = true
		switch g.Part {
		case 1:
			words += leadWordsPart1
			for _, q := range g.Questions {
				for _, o := range q.Options {
					words += optionLeadWords + wordCount(o)
				}
			}
			secs += part12PauseSecs
		case 2:
			words += leadWordsPart2
			if len(g.Script) > 0 {
				for _, t := range g.Script {
					words += wordCount(t.Text)
				}
			} else if len(g.Questions) > 0 {
				words += wordCount(g.Questions[0].Prompt)
			}
			for _, q := range g.Questions {
				for _, o := range q.Options {
					words += optionLeadWords + wordCount(o)
				}
			}
			secs += part12PauseSecs
		case 3, 4:
			words += leadWordsPart34
			for _, t := range g.Script {
				words += wordCount(t.Text)
				secs += betweenTurnsSecs
			}
			for _, q := range g.Questions {
				words += wordCount(q.Prompt)
				secs += part34PauseSecs
			}
		}
	}
	secs += float64(len(parts)) * directionsSecs
	secs += float64(words) / wordsPerMinute * 60
	return math.Round(secs/60*10) / 10
}
