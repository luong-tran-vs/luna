package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// lessonReviewSchema adds the column of the AI check of a lesson (F22). It reruns safely: the
// migration ignores "duplicate column name".
func lessonReviewSchema() []string {
	return []string{`ALTER TABLE lessons ADD COLUMN review JSON NULL`}
}

type lessonReviewJSON struct {
	CheckedAt  time.Time        `json:"checkedAt"`
	VerifiedAt *time.Time       `json:"verifiedAt"`
	Flags      []lessonFlagJSON `json:"flags"`
}

type lessonFlagJSON struct {
	Area      string `json:"area"`
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	NoteVi    string `json:"noteVi"`
	Confirmed bool   `json:"confirmed"`
}

// lessonReviewValue encodes a review for the review column; nil is SQL NULL.
func lessonReviewValue(r *lesson.Review) (any, error) {
	if r == nil {
		return nil, nil
	}
	j := lessonReviewJSON{CheckedAt: r.CheckedAt.UTC(), Flags: make([]lessonFlagJSON, len(r.Flags))}
	if !r.VerifiedAt.IsZero() {
		v := r.VerifiedAt.UTC()
		j.VerifiedAt = &v
	}
	for i, f := range r.Flags {
		j.Flags[i] = lessonFlagJSON{Area: string(f.Area), Index: f.Index, Kind: string(f.Kind), NoteVi: f.NoteVi, Confirmed: f.Confirmed}
	}
	return lessonJSON(j)
}

// lessonReviewFrom decodes the review column; NULL (or JSON null) is a lesson never checked.
func lessonReviewFrom(raw []byte) (*lesson.Review, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var j lessonReviewJSON
	if err := fromJSON(raw, &j); err != nil {
		return nil, err
	}
	r := &lesson.Review{CheckedAt: j.CheckedAt, Flags: make([]lesson.Flag, len(j.Flags))}
	if j.VerifiedAt != nil {
		r.VerifiedAt = *j.VerifiedAt
	}
	for i, f := range j.Flags {
		r.Flags[i] = lesson.Flag{
			Area: lesson.FlagArea(f.Area), Index: f.Index, Kind: lesson.FlagKind(f.Kind), NoteVi: f.NoteVi, Confirmed: f.Confirmed,
		}
	}
	return r, nil
}

// SaveReview stores (or, for nil, clears) the AI check if the lesson is still at revision. It does
// not touch updated_at. ok is false when the lesson is missing or no longer at revision.
func (r *Lessons) SaveReview(ctx context.Context, id string, revision int, rev *lesson.Review) (bool, error) {
	if !lessonValidID(id) {
		return false, lesson.ErrNotFound
	}
	v, err := lessonReviewValue(rev)
	if err != nil {
		return false, err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE lessons SET review = ? WHERE id = ? AND revision = ?`, v, id, revision)
	if err != nil {
		return false, fmt.Errorf("mysql save review: %w", err)
	}
	return changed(res)
}
