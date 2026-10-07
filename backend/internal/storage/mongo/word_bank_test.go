package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/wordbank"
)

func TestBankWordDoc(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)
	w := wordbank.Word{Lemma: "give up", MeaningVi: "từ bỏ", IPA: "/ɡɪv ʌp/", CreatedAt: at, UpdatedAt: at}

	// A new word is stored with imageAt null, so the "missing image" filter finds it.
	raw, err := bson.Marshal(toBankWordDoc(w))
	if err != nil {
		t.Fatal(err)
	}
	if v := bson.Raw(raw).Lookup("imageAt"); v.Type != bson.TypeNull {
		t.Fatalf("imageAt = %v", v)
	}
	if id := bson.Raw(raw).Lookup("_id").StringValue(); id != "give up" {
		t.Fatalf("_id = %q", id)
	}
	var d bankWordDoc
	if err := bson.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if got := d.toWord(); got != w || got.HasImage() {
		t.Fatalf("word = %+v", got)
	}
	d.ImageAt = &at
	if got := d.toWord(); !got.ImageAt.Equal(at) || !got.HasImage() {
		t.Fatalf("with image = %+v", got)
	}
}
