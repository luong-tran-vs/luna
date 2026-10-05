package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/job"
	"github.com/luongtran/luna/backend/internal/lesson"
	"github.com/luongtran/luna/backend/internal/storage"
	"github.com/luongtran/luna/backend/internal/topic"
)

// Lessons implements storage.LessonStore on the "lessons" table. The nested parts that are only
// read and written whole (sentences, annotations, questions, grammar note, practice) are JSON
// columns of the lesson row.
type Lessons struct {
	db *sql.DB
}

var _ storage.LessonStore = (*Lessons)(nil)

// NewLessons returns the Lessons repository.
func NewLessons(db *sql.DB) *Lessons { return &Lessons{db: db} }

// lessonsSchema is the DDL of this domain (see migrate.go).
func lessonsSchema() []string {
	return []string{`CREATE TABLE IF NOT EXISTS lessons (
	id                CHAR(24)     NOT NULL PRIMARY KEY,
	title             TEXT         NOT NULL,
	content           MEDIUMTEXT   NOT NULL,
	level             VARCHAR(16)  NOT NULL,
	topic_id          CHAR(24)     NOT NULL,
	source            TEXT         NOT NULL,
	license           TEXT         NOT NULL,
	grammar_point_id  VARCHAR(64)  NOT NULL DEFAULT '',
	revision         INT          NOT NULL,
	sentences         JSON         NOT NULL,
	annotation_status VARCHAR(16)  NOT NULL,
	annotation_error  TEXT         NOT NULL,
	annotations       JSON         NOT NULL,
	questions         JSON         NOT NULL,
	grammar_note      JSON         NULL,
	writing_prompt    TEXT         NOT NULL,
	extras_edited     TINYINT(1)   NOT NULL,
	quiz_version      INT          NOT NULL,
	practice          JSON         NULL,
	practice_status   VARCHAR(16)  NOT NULL,
	practice_error    TEXT         NOT NULL,
	practice_version  INT          NOT NULL,
	created_at        DATETIME(6)  NULL,
	updated_at        DATETIME(6)  NULL,
	KEY lessons_created_at (created_at),
	KEY lessons_level (level),
	KEY lessons_topic_id (topic_id)
) ` + tableOptions}
}

// lessonCols lists the columns lessonScan reads, in order.
const lessonCols = `id, title, content, level, topic_id, source, license, grammar_point_id, revision, sentences,
	annotation_status, annotation_error, annotations, questions, grammar_note, writing_prompt,
	extras_edited, quiz_version, practice, practice_status, practice_error, practice_version,
	created_at, updated_at, review`

// lessonSummaryCols lists the columns lessonScanSummary reads.
const lessonSummaryCols = `id, title, level, topic_id, annotation_status, created_at, review`

// lessonValidID reports whether id has the shape newID makes (24 lowercase hex characters).
func lessonValidID(id string) bool {
	if len(id) != 24 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// lessonTopicKey is the stored topic id: a malformed id is stored as "" (no topic), like Mongo's zero id.
func lessonTopicKey(id string) string {
	if lessonValidID(id) {
		return id
	}
	return ""
}

// lessonJSON encodes v for a JSON column as text (a []byte would be sent as binary, which MySQL
// refuses for JSON).
func lessonJSON(v any) (string, error) {
	b, err := toJSON(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// lessonExtrasValues encodes Extras as the values of questions, grammar_note and writing_prompt.
func lessonExtrasValues(x lesson.Extras) (questions string, note any, prompt string, err error) {
	qs := x.Questions
	if qs == nil {
		qs = []lesson.Question{}
	}
	if questions, err = lessonJSON(qs); err != nil {
		return "", nil, "", err
	}
	if x.GrammarNote != nil {
		n := *x.GrammarNote
		if n.Examples == nil {
			n.Examples = []string{}
		}
		s, err := lessonJSON(n)
		if err != nil {
			return "", nil, "", err
		}
		note = s
	}
	return questions, note, x.WritingPrompt, nil
}

func lessonAnnotationsJSON(anns []lesson.Annotation) (string, error) {
	if anns == nil {
		anns = []lesson.Annotation{}
	}
	return lessonJSON(anns)
}

// lessonWriteArgs holds the encoded columns shared by Create and ReplaceContent.
type lessonWriteArgs struct {
	sentences, annotations, questions, prompt string
	note, practice, review                    any
}

func lessonEncode(l lesson.Lesson) (lessonWriteArgs, error) {
	var a lessonWriteArgs
	var err error
	ss := l.Sentences
	if ss == nil {
		ss = []lesson.Sentence{}
	}
	if a.sentences, err = lessonJSON(ss); err != nil {
		return a, err
	}
	if a.annotations, err = lessonAnnotationsJSON(l.Annotations); err != nil {
		return a, err
	}
	if a.questions, a.note, a.prompt, err = lessonExtrasValues(l.Extras); err != nil {
		return a, err
	}
	if a.practice, err = lessonPracticeValue(l.Practice); err != nil {
		return a, err
	}
	a.review, err = lessonReviewValue(l.Review)
	return a, err
}

// lessonNormalize gives the slices a Lesson read from the database always has (never nil).
func lessonNormalize(l *lesson.Lesson) {
	if l.Sentences == nil {
		l.Sentences = []lesson.Sentence{}
	}
	if l.Annotations == nil {
		l.Annotations = []lesson.Annotation{}
	}
	if l.Extras.Questions == nil {
		l.Extras.Questions = []lesson.Question{}
	}
}

// Create inserts l and returns it with its new ID.
func (r *Lessons) Create(ctx context.Context, l lesson.Lesson) (lesson.Lesson, error) {
	enc, err := lessonEncode(l)
	if err != nil {
		return lesson.Lesson{}, err
	}
	l.ID = newID()
	l.TopicID = lessonTopicKey(l.TopicID)
	_, err = r.db.ExecContext(ctx, `INSERT INTO lessons (`+lessonCols+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		l.ID, l.Title, l.Content, string(l.Level), l.TopicID, l.Source, l.License, l.GrammarPointID, l.Revision, enc.sentences,
		string(l.AnnotationStatus), l.AnnotationError, enc.annotations, enc.questions, enc.note, enc.prompt,
		l.ExtrasEditedByAdmin, l.QuizVersion, enc.practice, string(l.PracticeStatus), l.PracticeError, l.PracticeVersion,
		nullTime(l.CreatedAt), nullTime(l.UpdatedAt), enc.review)
	if err != nil {
		return lesson.Lesson{}, fmt.Errorf("mysql insert lesson: %w", err)
	}
	lessonNormalize(&l)
	return l, nil
}

// lessonRowScanner is what *sql.Row and *sql.Rows share.
type lessonRowScanner interface{ Scan(dest ...any) error }

func lessonScan(s lessonRowScanner) (lesson.Lesson, error) {
	var (
		l                                    lesson.Lesson
		level, annStatus, pracStatus         string
		sentences, anns, questions, note, pr []byte
		reviewRaw                            []byte
		created, updated                     sql.NullTime
	)
	err := s.Scan(&l.ID, &l.Title, &l.Content, &level, &l.TopicID, &l.Source, &l.License, &l.GrammarPointID, &l.Revision, &sentences,
		&annStatus, &l.AnnotationError, &anns, &questions, &note, &l.Extras.WritingPrompt,
		&l.ExtrasEditedByAdmin, &l.QuizVersion, &pr, &pracStatus, &l.PracticeError, &l.PracticeVersion,
		&created, &updated, &reviewRaw)
	if err != nil {
		return lesson.Lesson{}, err
	}
	l.Level, l.AnnotationStatus, l.PracticeStatus = lesson.Level(level), lesson.Status(annStatus), lesson.Status(pracStatus)
	l.CreatedAt, l.UpdatedAt = timeOf(created), timeOf(updated)
	if err := fromJSON(sentences, &l.Sentences); err != nil {
		return lesson.Lesson{}, err
	}
	if err := fromJSON(anns, &l.Annotations); err != nil {
		return lesson.Lesson{}, err
	}
	if err := fromJSON(questions, &l.Extras.Questions); err != nil {
		return lesson.Lesson{}, err
	}
	if len(note) > 0 && string(note) != "null" {
		l.Extras.GrammarNote = &lesson.GrammarNote{}
		if err := fromJSON(note, l.Extras.GrammarNote); err != nil {
			return lesson.Lesson{}, err
		}
	}
	if l.Review, err = lessonReviewFrom(reviewRaw); err != nil {
		return lesson.Lesson{}, err
	}
	if l.Practice, err = lessonPracticeFrom(pr); err != nil {
		return lesson.Lesson{}, err
	}
	lessonNormalize(&l)
	return l, nil
}

func lessonScanSummary(s lessonRowScanner) (lesson.Summary, error) {
	var (
		sm            lesson.Summary
		level, status string
		created       sql.NullTime
		reviewRaw     []byte
	)
	if err := s.Scan(&sm.ID, &sm.Title, &level, &sm.TopicID, &status, &created, &reviewRaw); err != nil {
		return lesson.Summary{}, err
	}
	sm.Level, sm.AnnotationStatus, sm.CreatedAt = lesson.Level(level), lesson.Status(status), timeOf(created)
	rev, err := lessonReviewFrom(reviewRaw)
	if err != nil {
		return lesson.Summary{}, err
	}
	sm.Flags, sm.Checked, sm.Verified = lesson.SummaryOf(rev)
	return sm, nil
}

// Get returns lesson.ErrNotFound for unknown or malformed ids.
func (r *Lessons) Get(ctx context.Context, id string) (lesson.Lesson, error) {
	if !lessonValidID(id) {
		return lesson.Lesson{}, lesson.ErrNotFound
	}
	l, err := lessonScan(r.db.QueryRowContext(ctx, `SELECT `+lessonCols+` FROM lessons WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return lesson.Lesson{}, lesson.ErrNotFound
	}
	if err != nil {
		return lesson.Lesson{}, fmt.Errorf("mysql get lesson: %w", err)
	}
	return l, nil
}

// List returns summaries newest first, filtered by level and topic id.
func (r *Lessons) List(ctx context.Context, f lesson.Filter) ([]lesson.Summary, error) {
	var where []string
	var params []any
	if f.Level != "" {
		where = append(where, "level = ?")
		params = append(params, string(f.Level))
	}
	if f.TopicID != "" {
		if !lessonValidID(f.TopicID) {
			return []lesson.Summary{}, nil // a malformed topic id matches no lesson
		}
		where = append(where, "topic_id = ?")
		params = append(params, f.TopicID)
	}
	q := `SELECT ` + lessonSummaryCols + ` FROM lessons`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY created_at DESC, id DESC`
	return r.summaries(ctx, q, params)
}

// Summaries returns the existing lessons among ids.
func (r *Lessons) Summaries(ctx context.Context, ids []string) ([]lesson.Summary, error) {
	valid := make([]string, 0, len(ids))
	for _, id := range ids {
		if lessonValidID(id) {
			valid = append(valid, id)
		}
	}
	if len(valid) == 0 {
		return nil, nil
	}
	return r.summaries(ctx, `SELECT `+lessonSummaryCols+` FROM lessons WHERE id IN (`+placeholders(len(valid))+`)`, args(valid))
}

func (r *Lessons) summaries(ctx context.Context, query string, params []any) ([]lesson.Summary, error) {
	rows, err := r.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("mysql find lessons: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []lesson.Summary{}
	for rows.Next() {
		sm, err := lessonScanSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("mysql read lessons: %w", err)
		}
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read lessons: %w", err)
	}
	return out, nil
}

// UpdateInfo changes only the descriptive fields, leaving content and work results intact.
func (r *Lessons) UpdateInfo(ctx context.Context, id string, in lesson.Info) error {
	return r.updateOne(ctx, id, `title = ?, level = ?, topic_id = ?, source = ?, license = ?, grammar_point_id = ?, updated_at = ?`,
		in.Title, string(in.Level), lessonTopicKey(in.TopicID), in.Source, in.License, in.GrammarPointID, utc(time.Now()))
}

// ReplaceContent overwrites every field except the id and creation time.
func (r *Lessons) ReplaceContent(ctx context.Context, l lesson.Lesson) error {
	enc, err := lessonEncode(l)
	if err != nil {
		return err
	}
	return r.updateOne(ctx, l.ID, `title = ?, content = ?, level = ?, topic_id = ?, source = ?, license = ?, grammar_point_id = ?,
		revision = ?, sentences = ?, annotation_status = ?, annotation_error = ?, annotations = ?,
		extras_edited = ?, quiz_version = ?, practice = ?, practice_status = ?, practice_error = ?,
		updated_at = ?, questions = ?, grammar_note = ?, writing_prompt = ?, review = NULL`,
		l.Title, l.Content, string(l.Level), lessonTopicKey(l.TopicID), l.Source, l.License, l.GrammarPointID,
		l.Revision, enc.sentences, string(l.AnnotationStatus), l.AnnotationError, enc.annotations,
		l.ExtrasEditedByAdmin, l.QuizVersion, enc.practice, string(l.PracticeStatus), l.PracticeError,
		nullTime(l.UpdatedAt), enc.questions, enc.note, enc.prompt)
}

// SetStatus changes one work status if the lesson is still at revision.
func (r *Lessons) SetStatus(ctx context.Context, id string, revision int, t job.Type, st lesson.Status, errMsg string) (bool, error) {
	var set string
	switch t {
	case job.TypeAnnotate:
		set = `annotation_status = ?, annotation_error = ?`
	case job.TypePractice:
		set = `practice_status = ?, practice_error = ?`
	default:
		return false, fmt.Errorf("mysql set status: unknown job type %q", t)
	}
	return r.updateAtRevision(ctx, id, revision, "", set, string(st), errMsg)
}

// SaveAnnotations stores AI annotations and extras, marks them done and bumps quiz_version if
// still at revision. The practice is dropped and marked running: a new one is queued (F17).
func (r *Lessons) SaveAnnotations(ctx context.Context, id string, revision int, anns []lesson.Annotation,
	x lesson.Extras,
) (bool, error) {
	a, err := lessonAnnotationsJSON(anns)
	if err != nil {
		return false, err
	}
	questions, note, prompt, err := lessonExtrasValues(x)
	if err != nil {
		return false, err
	}
	return r.updateAtRevision(ctx, id, revision, `quiz_version = quiz_version + 1`,
		`annotations = ?, annotation_status = ?, annotation_error = '', extras_edited = 0,
		practice = NULL, practice_status = ?, practice_error = '',
		questions = ?, grammar_note = ?, writing_prompt = ?, review = NULL`,
		a, string(lesson.StatusDone), string(lesson.StatusRunning), questions, note, prompt)
}

// ReplaceExtras stores admin-edited extras and marks them edited; bumpQuiz bumps quiz_version.
func (r *Lessons) ReplaceExtras(ctx context.Context, id string, x lesson.Extras, bumpQuiz bool) error {
	questions, note, prompt, err := lessonExtrasValues(x)
	if err != nil {
		return err
	}
	set := `questions = ?, grammar_note = ?, writing_prompt = ?, extras_edited = 1, updated_at = ?, review = NULL`
	if bumpQuiz {
		set += `, quiz_version = quiz_version + 1`
	}
	return r.updateOne(ctx, id, set, questions, note, prompt, utc(time.Now()))
}

// ReplaceAnnotations stores admin-edited annotations and marks them done.
func (r *Lessons) ReplaceAnnotations(ctx context.Context, id string, anns []lesson.Annotation) error {
	a, err := lessonAnnotationsJSON(anns)
	if err != nil {
		return err
	}
	return r.updateOne(ctx, id, `annotations = ?, annotation_status = ?, annotation_error = '', updated_at = ?, review = NULL`,
		a, string(lesson.StatusDone), utc(time.Now()))
}

// Delete removes a lesson; deleting a missing lesson is not an error.
func (r *Lessons) Delete(ctx context.Context, id string) error {
	if !lessonValidID(id) {
		return lesson.ErrNotFound
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM lessons WHERE id = ?`, id); err != nil {
		return fmt.Errorf("mysql delete lesson: %w", err)
	}
	return nil
}

// updateOne runs UPDATE lessons SET set WHERE id = ?; params are the values of set.
func (r *Lessons) updateOne(ctx context.Context, id, set string, params ...any) error {
	if !lessonValidID(id) {
		return lesson.ErrNotFound
	}
	res, err := r.db.ExecContext(ctx, `UPDATE lessons SET `+set+` WHERE id = ?`, append(params, id)...)
	if err != nil {
		return fmt.Errorf("mysql update lesson: %w", err)
	}
	ok, err := changed(res)
	if err != nil {
		return err
	}
	if !ok {
		return lesson.ErrNotFound
	}
	return nil
}

// updateAtRevision applies set (plus the raw expression extra, when not empty) only while the
// lesson is still at revision; it also refreshes updated_at. A missing lesson is ok=false.
func (r *Lessons) updateAtRevision(ctx context.Context, id string, revision int, extra, set string, params ...any) (bool, error) {
	if !lessonValidID(id) {
		return false, lesson.ErrNotFound
	}
	q := `UPDATE lessons SET ` + set + `, updated_at = ?`
	if extra != "" {
		q += `, ` + extra
	}
	q += ` WHERE id = ? AND revision = ?`
	params = append(params, utc(time.Now()), id, revision)
	res, err := r.db.ExecContext(ctx, q, params...)
	if err != nil {
		return false, fmt.Errorf("mysql update lesson: %w", err)
	}
	return changed(res)
}

// --- topic.Lessons (F14), adapted in main ---

// CountByTopic counts lessons per topic id.
func (r *Lessons) CountByTopic(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT topic_id, COUNT(*) FROM lessons GROUP BY topic_id`)
	if err != nil {
		return nil, fmt.Errorf("mysql count lessons: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("mysql read lesson counts: %w", err)
		}
		out[id] += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read lesson counts: %w", err)
	}
	return out, nil
}

// TopicOf returns the topic id of each existing lesson among ids.
func (r *Lessons) TopicOf(ctx context.Context, ids []string) (map[string]string, error) {
	sums, err := r.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(sums))
	for _, s := range sums {
		out[s.ID] = s.TopicID
	}
	return out, nil
}

// SetLevelByTopic sets the level of every lesson of a topic. A malformed or empty topic id has no
// lessons, so it changes nothing. A grammar point belongs to one level, so lessons that change
// level lose theirs (MySQL reads the old level in the first assignment).
func (r *Lessons) SetLevelByTopic(ctx context.Context, topicID, level string) error {
	if !lessonValidID(topicID) {
		return nil
	}
	if _, err := r.db.ExecContext(ctx,
		`UPDATE lessons SET grammar_point_id = IF(level <> ?, '', grammar_point_id), level = ? WHERE topic_id = ?`,
		level, level, topicID); err != nil {
		return fmt.Errorf("mysql update lesson levels: %w", err)
	}
	return nil
}

// CountByGrammarPoint counts lessons per grammar point; topicID "" counts every topic.
func (r *Lessons) CountByGrammarPoint(ctx context.Context, topicID string) (map[string]int, error) {
	q := `SELECT grammar_point_id, COUNT(*) FROM lessons WHERE grammar_point_id <> ''`
	var params []any
	if topicID != "" {
		if !lessonValidID(topicID) {
			return map[string]int{}, nil // a malformed topic id matches no lesson
		}
		q += ` AND topic_id = ?`
		params = append(params, topicID)
	}
	rows, err := r.db.QueryContext(ctx, q+` GROUP BY grammar_point_id`, params...)
	if err != nil {
		return nil, fmt.Errorf("mysql count grammar points: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("mysql read grammar point counts: %w", err)
		}
		out[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql read grammar point counts: %w", err)
	}
	return out, nil
}

// TopicTexts returns the content and single-word annotation lemmas of every lesson of the
// given topics, by topic id.
func (r *Lessons) TopicTexts(ctx context.Context, topicIDs []string) (map[string][]topic.LessonText, error) {
	valid := make([]string, 0, len(topicIDs))
	for _, id := range topicIDs {
		if lessonValidID(id) {
			valid = append(valid, id)
		}
	}
	out := map[string][]topic.LessonText{}
	if len(valid) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT topic_id, content, annotations FROM lessons WHERE topic_id IN (`+placeholders(len(valid))+`) ORDER BY created_at, id`,
		args(valid)...)
	if err != nil {
		return nil, fmt.Errorf("mysql find topic lessons: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, content string
		var raw []byte
		if err := rows.Scan(&id, &content, &raw); err != nil {
			return nil, fmt.Errorf("mysql decode topic lessons: %w", err)
		}
		var anns []struct{ Text, Lemma string }
		if err := fromJSON(raw, &anns); err != nil {
			return nil, err
		}
		t := topic.LessonText{Content: content, Lemmas: map[string]string{}}
		for _, a := range anns {
			text := strings.ToLower(strings.TrimSpace(a.Text))
			if text != "" && !strings.Contains(text, " ") && a.Lemma != "" {
				t.Lemmas[text] = strings.ToLower(strings.TrimSpace(a.Lemma))
			}
		}
		out[id] = append(out[id], t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql decode topic lessons: %w", err)
	}
	return out, nil
}
