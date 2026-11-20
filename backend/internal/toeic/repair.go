package toeic

import (
	"fmt"
	"strings"
	"unicode"
)

// repairEvidence fixes the evidence of a group the AI wrote, before it is checked: a span that is
// in the material but under the wrong turn, text or block is pointed at the right place (with the
// material's own spelling), and a span that is nowhere in the material is dropped. Evidence only
// marks where the answer is; a wrong index or a paraphrased quote should not throw away a group
// whose questions are fine. Imports are never repaired: a file must be right.
func repairEvidence(g *Group) {
	for qi := range g.Questions {
		q := &g.Questions[qi]
		kept := q.Evidence[:0]
		for _, r := range q.Evidence {
			if checkRef(*g, r) == "" {
				kept = append(kept, r)
				continue
			}
			if fixed, ok := findRef(*g, r.Text); ok {
				kept = append(kept, fixed)
			}
		}
		q.Evidence = kept
	}
}

// findRef looks for a span in the script, then in the texts, ignoring case, spacing and the kind of
// quotes and dashes; it returns a Ref with the span as the material writes it.
func findRef(g Group, span string) (Ref, bool) {
	want := fold(span)
	if want == "" {
		return Ref{}, false
	}
	for i, t := range g.Script {
		if exact, ok := locate(t.Text, want); ok {
			return Ref{Doc: -1, Turn: i, Text: exact}, true
		}
	}
	for d, doc := range g.Docs {
		for b, block := range doc.Body {
			if exact, ok := locate(blockText(block), want); ok {
				return Ref{Doc: d, Block: b, Text: exact}, true
			}
		}
	}
	return Ref{}, false
}

// locate finds the folded span in text and returns the matching part of text as written.
func locate(text, want string) (string, bool) {
	runes := []rune(text)
	// folded[i] is the folded form of runes[i]; spans of spaces fold to one space.
	var b strings.Builder
	var starts []int
	prevSpace := false
	for i, r := range runes {
		f := foldRune(r)
		if f == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(f)
		starts = append(starts, i)
	}
	hay := []rune(b.String())
	need := []rune(want)
	at := strings.Index(string(hay), want)
	if at < 0 {
		return "", false
	}
	first := len([]rune(string(hay)[:at]))
	last := first + len(need) - 1
	if last >= len(starts) {
		return "", false
	}
	end := starts[last] + 1
	return string(runes[starts[first]:end]), true
}

// fold lowers a span, makes every quote and dash plain and keeps one space between words.
func fold(s string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range strings.TrimSpace(s) {
		f := foldRune(r)
		if f == ' ' {
			if prevSpace {
				continue
			}
			prevSpace = true
		} else {
			prevSpace = false
		}
		b.WriteRune(f)
	}
	return strings.TrimRight(b.String(), " ")
}

func foldRune(r rune) rune {
	switch r {
	case '‘', '’', '‛', '`', '´':
		return '\''
	case '“', '”', '„':
		return '"'
	case '–', '—', '‒', '−':
		return '-'
	}
	if unicode.IsSpace(r) {
		return ' '
	}
	return unicode.ToLower(r)
}

// kindSynonyms maps the names the AI may use for a kind of text, block or graphic to the import
// format's.
var kindSynonyms = map[string]string{
	"advertisement": "ad", "advert": "ad", "web page": "web", "webpage": "web", "website": "web", "web-page": "web",
	"e-mail": "email", "notice board": "notice", "announcement": "notice", "news article": "article",
	"text message": "chat", "text messages": "chat", "text-message": "chat", "message chain": "chat", "online chat": "chat",
	"customer review": "review", "receipt": "invoice", "bill": "invoice", "application form": "form",
	"timetable": "schedule", "agenda": "schedule", "itinerary": "schedule", "letter of": "letter",
	"paragraph": "p", "para": "p", "text": "p", "body": "p", "heading": "h", "header": "h", "title": "h",
	"bullet": "list", "bullets": "list", "list items": "list", "message": "msg", "chat message": "msg",
	"pricelist": "price-list", "price list": "price-list", "floorplan": "floor-plan", "floor plan": "floor-plan",
	"bar chart": "bar-chart", "barchart": "bar-chart", "chart": "bar-chart",
}

// repairKinds puts the kinds of an AI-written group into the import format's words (lower case,
// known synonyms), so a group is not thrown away for writing "paragraph" instead of "p".
func repairKinds(g *Group) {
	norm := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		if v, ok := kindSynonyms[s]; ok {
			return v
		}
		return s
	}
	for i := range g.Docs {
		g.Docs[i].Kind = norm(g.Docs[i].Kind)
		for j := range g.Docs[i].Body {
			g.Docs[i].Body[j].Kind = norm(g.Docs[i].Body[j].Kind)
		}
	}
	if g.Graphic != nil {
		g.Graphic.Kind = norm(g.Graphic.Kind)
	}
}

// vietnameseOnly are letters that appear in Vietnamese but not in English or in the usual loanwords
// (café, résumé): a test field holding one of them was written in Vietnamese.
const vietnameseOnly = "ăâđơưĂÂĐƠƯạảấầẩẫậắằẳẵặẹẻẽếềểễệỉịọỏốồổỗộớờởỡợụủứừửữựỳỵỷỹẠẢẤẦẨẪẬẮẰẲẴẶẸẺẼẾỀỂỄỆỈỊỌỎỐỒỔỖỘỚỜỞỠỢỤỦỨỪỬỮỰỲỴỶỸ"

// notEnglish returns the test fields of a group (what the learner reads or hears) that contain
// Vietnamese, keyed like the import's errors; empty when the material is in English.
func notEnglish(g Group) map[string]string {
	const msg = "Nội dung đề phải bằng tiếng Anh (chỉ giải thích, bản dịch bằng tiếng Việt)"
	out := map[string]string{}
	vi := func(s string) bool { return strings.ContainsAny(s, vietnameseOnly) }
	if vi(g.Scene) {
		out["scene"] = msg
	}
	for i, t := range g.Script {
		if vi(t.Text) {
			out[fmt.Sprintf("script[%d].text", i)] = msg
		}
	}
	for i, d := range g.Docs {
		texts := []string{d.Title, docText(d)}
		for _, kv := range d.Meta {
			texts = append(texts, kv.K, kv.V)
		}
		for _, s := range texts {
			if vi(s) {
				out[fmt.Sprintf("docs[%d]", i)] = msg
				break
			}
		}
	}
	if g.Graphic != nil {
		parts := append([]string{g.Graphic.Title}, g.Graphic.Columns...)
		for _, r := range g.Graphic.Rows {
			parts = append(parts, r...)
		}
		for _, c := range g.Graphic.Cells {
			parts = append(parts, c.Label)
		}
		for _, b := range g.Graphic.Bars {
			parts = append(parts, b.Label)
		}
		if vi(strings.Join(parts, " ")) {
			out["graphic"] = msg
		}
	}
	for i, q := range g.Questions {
		if vi(q.Prompt) || vi(strings.Join(q.Options, " ")) {
			out[fmt.Sprintf("questions[%d]", i)] = msg
		}
	}
	return out
}

// splitSingles splits a group of Part 5 that the AI wrote with several questions into one group per
// question: a Part 5 group is one sentence. Other parts are left as they are.
func splitSingles(gs []Group) []Group {
	out := make([]Group, 0, len(gs))
	for _, g := range gs {
		if g.Part != 5 || len(g.Questions) <= 1 {
			out = append(out, g)
			continue
		}
		for _, q := range g.Questions {
			one := g
			q.No = 1
			one.Questions = []Question{q}
			out = append(out, one)
		}
	}
	return out
}

// headerPrefixes are header lines the AI sometimes repeats at the top of a text's body.
var headerPrefixes = []string{"to:", "from:", "subject:", "date:", "re:", "cc:", "url:", "sent:", "memorandum", "memo"}

// dropHeaderBlocks removes the first blocks of a text that only repeat its header ("To: ...",
// "From: ...", "MEMORANDUM"): the header is in "meta" and shown above the body.
func dropHeaderBlocks(g *Group) {
	for i := range g.Docs {
		d := &g.Docs[i]
		body := d.Body
		for len(body) > 1 && isHeaderBlock(body[0]) {
			body = body[1:]
		}
		// A first block may hold all the header lines and then the text.
		if len(body) > 0 && body[0].Kind == "p" {
			lines := strings.Split(body[0].Text, "\n")
			k := 0
			for k < len(lines)-1 && isHeaderLine(lines[k]) {
				k++
			}
			if k > 0 {
				body[0].Text = strings.TrimSpace(strings.Join(lines[k:], "\n"))
			}
		}
		d.Body = body
	}
}

func isHeaderBlock(b Block) bool {
	if b.Kind != "p" && b.Kind != "h" {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(b.Text), "\n") {
		if !isHeaderLine(line) {
			return false
		}
	}
	return strings.TrimSpace(b.Text) != ""
}

func isHeaderLine(line string) bool {
	l := strings.ToLower(strings.TrimSpace(line))
	if l == "" {
		return false
	}
	for _, p := range headerPrefixes {
		if strings.HasPrefix(l, p) && (strings.HasSuffix(p, ":") || l == p) {
			return true
		}
	}
	return false
}
