package llm

var (
	schemaString  = map[string]any{"type": "STRING"}
	schemaInt     = map[string]any{"type": "INTEGER"}
	schemaStrings = map[string]any{"type": "ARRAY", "items": schemaString}
)

func schemaObject(props map[string]any, required ...string) map[string]any {
	o := map[string]any{"type": "OBJECT", "properties": props}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}

func schemaArray(items map[string]any) map[string]any {
	return map[string]any{"type": "ARRAY", "items": items}
}

// schemaEnum is a string that must be one of values.
func schemaEnum(values ...string) map[string]any {
	return map[string]any{"type": "STRING", "enum": values}
}

// toeicSchema is the import format of a TOEIC group (contracts/content-format.md); a part uses only
// some of the fields.
var toeicSchema = schemaObject(map[string]any{
	"groups": schemaArray(schemaObject(map[string]any{
		"level": schemaInt,
		"scene": schemaString,
		"script": schemaArray(schemaObject(map[string]any{
			"speaker": schemaString, "accent": schemaEnum("us", "gb", "au", "ca"), "text": schemaString, "translateVi": schemaString,
		}, "speaker", "accent", "text")),
		"graphic": schemaObject(map[string]any{
			"kind": schemaEnum("table", "schedule", "price-list", "floor-plan", "bar-chart"), "title": schemaString, "columns": schemaStrings, "rows": schemaArray(schemaStrings),
			"cells": schemaArray(schemaObject(map[string]any{"label": schemaString, "row": schemaInt, "col": schemaInt, "span": schemaInt}, "label", "row", "col")),
			"bars":  schemaArray(schemaObject(map[string]any{"label": schemaString, "value": map[string]any{"type": "NUMBER"}}, "label", "value")),
		}, "kind"),
		"docs": schemaArray(schemaObject(map[string]any{
			"kind": schemaEnum("email", "letter", "memo", "notice", "ad", "article", "web", "invoice", "form", "schedule", "chat", "review"), "title": schemaString, "translateVi": schemaString,
			"meta": schemaArray(schemaObject(map[string]any{"k": schemaString, "v": schemaString}, "k", "v")),
			"body": schemaArray(schemaObject(map[string]any{
				"kind": schemaEnum("p", "h", "list", "table", "msg"), "text": schemaString, "items": schemaStrings, "who": schemaString, "time": schemaString,
				"table": schemaObject(map[string]any{"columns": schemaStrings, "rows": schemaArray(schemaStrings)}, "columns", "rows"),
			}, "kind")),
		}, "kind", "body")),
		"questions": schemaArray(schemaObject(map[string]any{
			"no": schemaInt, "prompt": schemaString, "options": schemaStrings, "answer": schemaInt,
			"evidence":    schemaArray(schemaObject(map[string]any{"doc": schemaInt, "block": schemaInt, "turn": schemaInt, "text": schemaString}, "doc", "text")),
			"explainVi":   schemaString,
			"translateVi": schemaString,
			"tags":        schemaStrings,
			"vocab":       schemaArray(schemaObject(map[string]any{"word": schemaString, "ipa": schemaString, "meaningVi": schemaString}, "word", "meaningVi")),
		}, "no", "options", "answer", "explainVi", "tags")),
	}, "level", "questions")),
}, "groups")
