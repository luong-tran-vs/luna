package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/luongtran/luna/backend/internal/lesson"
)

// The practice (F17) is one JSON column of the lesson row (practice), with its status, error and
// version in plain columns, since the version is compared in an UPDATE.

// lessonPracticeValue encodes a practice for the practice column: NULL for none.
func lessonPracticeValue(p *lesson.Practice) (any, error) {
	if p == nil {
		return nil, nil
	}
	c := *p
	if c.Examples == nil {
		c.Examples = []lesson.Example{}
	}
	if c.Translations == nil {
		c.Translations = []lesson.Translation{}
	}
	s, err := lessonJSON(c)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// lessonPracticeFrom decodes the practice column; NULL is no practice.
func lessonPracticeFrom(raw []byte) (*lesson.Practice, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	p := &lesson.Practice{}
	if err := fromJSON(raw, p); err != nil {
		return nil, err
	}
	if p.Examples == nil {
		p.Examples = []lesson.Example{}
	}
	if p.Translations == nil {
		p.Translations = []lesson.Translation{}
	}
	return p, nil
}

// SavePractice stores a practice, marks it done and bumps practice_version, if the lesson is
// still at revision and practice_version still equals prevVersion.
func (r *Lessons) SavePractice(ctx context.Context, id string, revision, prevVersion int, p lesson.Practice) (bool, error) {
	if !lessonValidID(id) {
		return false, lesson.ErrNotFound
	}
	v, err := lessonPracticeValue(&p)
	if err != nil {
		return false, err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE lessons
		SET practice = ?, practice_status = ?, practice_error = '', updated_at = ?, practice_version = practice_version + 1
		WHERE id = ? AND revision = ? AND practice_version = ?`,
		v, string(lesson.StatusDone), utc(time.Now()), id, revision, prevVersion)
	if err != nil {
		return false, fmt.Errorf("mysql save practice: %w", err)
	}
	return changed(res)
}

// WithoutPractice lists the lessons whose annotations are done but whose practice was never queued.
func (r *Lessons) WithoutPractice(ctx context.Context) ([]lesson.RevisionRef, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, revision FROM lessons
		WHERE annotation_status = ? AND practice_status = ? ORDER BY created_at, id`,
		string(lesson.StatusDone), string(lesson.StatusNone))
	if err != nil {
		return nil, fmt.Errorf("mysql find lessons without practice: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []lesson.RevisionRef{}
	for rows.Next() {
		var ref lesson.RevisionRef
		if err := rows.Scan(&ref.ID, &ref.Revision); err != nil {
			return nil, fmt.Errorf("mysql read lessons without practice: %w", err)
		}
		out = append(out, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read lessons without practice: %w", err)
	}
	return out, nil
}
