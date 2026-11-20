package toeic

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contentDir is where Claude writes the TOEIC content (contracts/content-format.md).
const contentDir = "../../../content/toeic"

// TestContentFiles runs the import checks on every file of content/toeic/ offline, the same ones
// as POST /api/admin/toeic/import?dryRun=true, so a file is known to import before it is sent; and
// checks that each test file names existing groups that fill its parts.
func TestContentFiles(t *testing.T) {
	t.Parallel()
	files, _ := filepath.Glob(filepath.Join(contentDir, "part*", "*.json"))
	if len(files) == 0 {
		t.Skip("no content files")
	}
	all := map[string]Group{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Groups []Group `json:"groups"`
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		for i := range body.Groups {
			Normalize(&body.Groups[i])
		}
		if err := Validate(body.Groups); err != nil {
			t.Errorf("%s: %v", f, err)
		}
		part := strings.TrimPrefix(filepath.Base(filepath.Dir(f)), "part")
		for _, g := range body.Groups {
			if _, dup := all[g.Key]; dup {
				t.Errorf("%s: key %q is in another file too", f, g.Key)
			}
			if part != "" && string(rune('0'+int(g.Part))) != part {
				t.Errorf("%s: %q is Part %d", f, g.Key, g.Part)
			}
			g.Status = StatusPublished
			all[g.Key] = g
		}
	}

	tests, _ := filepath.Glob(filepath.Join(contentDir, "tests", "*.json"))
	for _, f := range tests {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var in TestInput
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		test := Test{Key: in.Key, Kind: in.Kind, Groups: in.Groups}
		groups := map[string]Group{}
		for _, k := range in.Groups {
			if g, ok := all[k]; ok {
				groups[k] = g
			}
		}
		photos := map[string][]string{}
		for k, g := range groups {
			if g.Part == 1 {
				photos[k] = []string{SlotPhoto} // drawn at publishing time, not part of the file
			}
		}
		c := check(test, groups, photos)
		if len(c.MissingGroups) > 0 || !c.Ready {
			t.Errorf("%s: missing %v, parts %+v", f, c.MissingGroups, c.Parts)
		}
	}
}
