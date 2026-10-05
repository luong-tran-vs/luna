package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// reviewDoc is the AI check of a lesson (F22), nested in the lesson document as "review".
type reviewDoc struct {
	CheckedAt  time.Time `bson:"checkedAt"`
	VerifiedAt time.Time `bson:"verifiedAt,omitempty"`
	Flags      []flagDoc `bson:"flags"`
}

type flagDoc struct {
	Area      string `bson:"area"`
	Index     int    `bson:"index"`
	Kind      string `bson:"kind"`
	NoteVi    string `bson:"noteVi"`
	Confirmed bool   `bson:"confirmed"`
}

func fromReview(r *lesson.Review) *reviewDoc {
	if r == nil {
		return nil
	}
	d := &reviewDoc{CheckedAt: r.CheckedAt.UTC(), Flags: make([]flagDoc, len(r.Flags))}
	if !r.VerifiedAt.IsZero() {
		d.VerifiedAt = r.VerifiedAt.UTC()
	}
	for i, f := range r.Flags {
		d.Flags[i] = flagDoc{Area: string(f.Area), Index: f.Index, Kind: string(f.Kind), NoteVi: f.NoteVi, Confirmed: f.Confirmed}
	}
	return d
}

func (d *reviewDoc) toReview() *lesson.Review {
	if d == nil {
		return nil
	}
	r := &lesson.Review{CheckedAt: d.CheckedAt, Flags: make([]lesson.Flag, len(d.Flags))}
	if !d.VerifiedAt.IsZero() {
		r.VerifiedAt = d.VerifiedAt
	}
	for i, f := range d.Flags {
		r.Flags[i] = lesson.Flag{
			Area: lesson.FlagArea(f.Area), Index: f.Index, Kind: lesson.FlagKind(f.Kind), NoteVi: f.NoteVi, Confirmed: f.Confirmed,
		}
	}
	return r
}

// SaveReview stores (or, for nil, removes) the AI check if the lesson is still at revision. It does
// not touch updatedAt.
func (r *Lessons) SaveReview(ctx context.Context, id string, revision int, rev *lesson.Review) (bool, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false, lesson.ErrNotFound
	}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "review", Value: fromReview(rev)}}}}
	if rev == nil {
		update = unsetReview
	}
	res, err := r.coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: oid}, {Key: "revision", Value: revision}}, update)
	if err != nil {
		return false, fmt.Errorf("save review: %w", err)
	}
	return res.MatchedCount == 1, nil
}

// unsetReview is the update that drops the review of a lesson.
var unsetReview = bson.D{{Key: "$unset", Value: reviewUnset}}

// reviewUnset is the $unset document that drops the review.
var reviewUnset = bson.D{{Key: "review", Value: ""}}
