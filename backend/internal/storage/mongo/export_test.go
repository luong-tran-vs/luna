package mongo

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	oid := bson.NewObjectID()
	at := time.Date(2026, 9, 30, 7, 0, 0, 123_000_000, time.UTC)
	in := bson.M{
		"_id":    oid,
		"userId": oid,
		"due":    bson.NewDateTimeFromTime(at),
		"when":   at,
		"steps":  bson.D{{Key: "read", Value: true}, {Key: "at", Value: bson.NewDateTimeFromTime(at)}},
		"nested": bson.M{"cardId": oid},
		"list":   bson.A{oid, "x", bson.D{{Key: "n", Value: int32(2)}}},
		"plain":  []any{int64(3)},
		"text":   "go",
	}
	want := map[string]any{
		"_id":    oid.Hex(),
		"userId": oid.Hex(),
		"due":    "2026-09-30T07:00:00.123Z",
		"when":   "2026-09-30T07:00:00.123Z",
		"steps":  map[string]any{"read": true, "at": "2026-09-30T07:00:00.123Z"},
		"nested": map[string]any{"cardId": oid.Hex()},
		"list":   []any{oid.Hex(), "x", map[string]any{"n": int32(2)}},
		"plain":  []any{int64(3)},
		"text":   "go",
	}
	if got := normalizeDoc(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalize =\n%#v\nwant\n%#v", got, want)
	}
}
