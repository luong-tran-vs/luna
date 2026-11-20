package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luongtran/luna/backend/internal/toeic"
)

// toeicSchema is the DDL of the TOEIC practice (F27, migration 047): the groups (content as JSON), their
// pictures, the tests, the attempts (state and result as JSON) with the groups each has (to grade them
// again when a group is replaced), the mistakes notebooks and the goals.
func toeicSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS toeic_groups (
  id         CHAR(24)     NOT NULL PRIMARY KEY,
  group_key  VARCHAR(48)  NOT NULL,
  part       TINYINT      NOT NULL,
  level      INT          NOT NULL,
  status     VARCHAR(10)  NOT NULL,
  source     VARCHAR(255) NOT NULL,
  version    INT          NOT NULL,
  body       MEDIUMTEXT   NOT NULL,
  created_at DATETIME(6)  NOT NULL,
  updated_at DATETIME(6)  NOT NULL,
  UNIQUE KEY uq_toeic_group_key (group_key),
  KEY idx_toeic_group_part_status (part, status)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_images (
  group_key  VARCHAR(48) NOT NULL,
  slot       VARCHAR(20) NOT NULL,
  mime       VARCHAR(40) NOT NULL,
  data       MEDIUMBLOB  NOT NULL,
  origin     VARCHAR(10) NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY (group_key, slot)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_tests (
  id         CHAR(24)    NOT NULL PRIMARY KEY,
  test_key   VARCHAR(48) NOT NULL,
  title      VARCHAR(80) NOT NULL,
  kind       VARCHAR(10) NOT NULL,
  ord        INT         NOT NULL,
  status     VARCHAR(10) NOT NULL,
  group_keys TEXT        NOT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  UNIQUE KEY uq_toeic_test_key (test_key)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_attempts (
  id           CHAR(24)    NOT NULL PRIMARY KEY,
  user_id      CHAR(24)    NOT NULL,
  status       VARCHAR(12) NOT NULL,
  test_key     VARCHAR(48) NOT NULL,
  mode         VARCHAR(10) NOT NULL,
  counted      TINYINT(1)  NOT NULL,
  body         MEDIUMTEXT  NOT NULL,
  result       MEDIUMTEXT  NULL,
  started_at   DATETIME(6) NOT NULL,
  saved_at     DATETIME(6) NOT NULL,
  submitted_at DATETIME(6) NULL,
  KEY idx_toeic_attempt_user_status (user_id, status, started_at),
  KEY idx_toeic_attempt_user_test (user_id, test_key)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_attempt_groups (
  attempt_id CHAR(24)    NOT NULL,
  group_key  VARCHAR(48) NOT NULL,
  PRIMARY KEY (attempt_id, group_key),
  KEY idx_toeic_attempt_group (group_key)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_mistakes (
  user_id       CHAR(24)    NOT NULL,
  question_id   VARCHAR(60) NOT NULL,
  group_key     VARCHAR(48) NOT NULL,
  part          TINYINT     NOT NULL,
  tags          TEXT        NOT NULL,
  streak        INT         NOT NULL,
  last_wrong_at DATETIME(6) NOT NULL,
  added_at      DATETIME(6) NOT NULL,
  PRIMARY KEY (user_id, question_id)
) ` + tableOptions,
		`CREATE TABLE IF NOT EXISTS toeic_goals (
  user_id    CHAR(24)    NOT NULL PRIMARY KEY,
  target     INT         NOT NULL,
  exam_date  VARCHAR(10) NOT NULL,
  updated_at DATETIME(6) NOT NULL
) ` + tableOptions,
	}
}

// --- toeic_groups ---

// ToeicGroups implements toeic.GroupRepository on "toeic_groups".
type ToeicGroups struct{ db *sql.DB }

// NewToeicGroups returns the ToeicGroups repository.
func NewToeicGroups(db *sql.DB) *ToeicGroups { return &ToeicGroups{db: db} }

var _ toeic.GroupRepository = (*ToeicGroups)(nil)

const toeicGroupCols = "id, group_key, part, level, status, source, version, body, created_at, updated_at"

func (r *ToeicGroups) query(ctx context.Context, where string, a ...any) ([]toeic.Group, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+toeicGroupCols+" FROM toeic_groups "+where+" ORDER BY part, group_key", a...)
	if err != nil {
		return nil, fmt.Errorf("mysql list toeic groups: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []toeic.Group
	for rows.Next() {
		var g toeic.Group
		var part int
		var status string
		var body []byte
		if err := rows.Scan(&g.ID, &g.Key, &part, &g.Level, &status, &g.Source, &g.Version, &body, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("mysql scan toeic group: %w", err)
		}
		var c toeic.Content
		if err := fromJSON(body, &c); err != nil {
			return nil, err
		}
		g.SetContent(c)
		g.Part, g.Status, g.CreatedAt, g.UpdatedAt = toeic.Part(part), toeic.Status(status), utc(g.CreatedAt), utc(g.UpdatedAt)
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list toeic groups: %w", err)
	}
	return out, nil
}

// List returns the groups matching f.
func (r *ToeicGroups) List(ctx context.Context, f toeic.GroupFilter) ([]toeic.Group, error) {
	var conds []string
	var a []any
	if f.Part != 0 {
		conds, a = append(conds, "part = ?"), append(a, int(f.Part))
	}
	if f.Status != "" {
		conds, a = append(conds, "status = ?"), append(a, string(f.Status))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	return r.query(ctx, where, a...)
}

// ByKeys returns the groups of the keys that exist.
func (r *ToeicGroups) ByKeys(ctx context.Context, keys []string) ([]toeic.Group, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	return r.query(ctx, "WHERE group_key IN ("+placeholders(len(keys))+")", args(keys)...)
}

// Published returns the published groups of a part (all parts when 0).
func (r *ToeicGroups) Published(ctx context.Context, part toeic.Part) ([]toeic.Group, error) {
	return r.List(ctx, toeic.GroupFilter{Part: part, Status: toeic.StatusPublished})
}

// Upsert creates the new keys and replaces the content of the known ones, keeping their status and
// creation time and raising their version.
func (r *ToeicGroups) Upsert(ctx context.Context, gs []toeic.Group) (created, replaced []string, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("mysql begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, g := range gs {
		body, err := toJSON(g.Content())
		if err != nil {
			return nil, nil, err
		}
		res, err := tx.ExecContext(ctx, `UPDATE toeic_groups SET part = ?, level = ?, source = ?, body = ?, version = version + 1,
updated_at = ? WHERE group_key = ?`, int(g.Part), g.Level, g.Source, body, utc(g.UpdatedAt), g.Key)
		if err != nil {
			return nil, nil, fmt.Errorf("mysql replace toeic group: %w", err)
		}
		if ok, err := changed(res); err != nil {
			return nil, nil, err
		} else if ok {
			replaced = append(replaced, g.Key)
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO toeic_groups ("+toeicGroupCols+") VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?)",
			newID(), g.Key, int(g.Part), g.Level, string(g.Status), g.Source, body, utc(g.CreatedAt), utc(g.UpdatedAt)); err != nil {
			return nil, nil, fmt.Errorf("mysql insert toeic group: %w", err)
		}
		created = append(created, g.Key)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("mysql commit: %w", err)
	}
	return created, replaced, nil
}

// SetStatus changes the status of a group.
func (r *ToeicGroups) SetStatus(ctx context.Context, key string, s toeic.Status) error {
	res, err := r.db.ExecContext(ctx, "UPDATE toeic_groups SET status = ? WHERE group_key = ?", string(s), key)
	if err != nil {
		return fmt.Errorf("mysql set toeic group status: %w", err)
	}
	if ok, err := changed(res); err != nil {
		return err
	} else if !ok {
		return toeic.ErrNotFound
	}
	return nil
}

// Delete removes the groups of the keys.
func (r *ToeicGroups) Delete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM toeic_groups WHERE group_key IN ("+placeholders(len(keys))+")", args(keys)...); err != nil {
		return fmt.Errorf("mysql delete toeic groups: %w", err)
	}
	return nil
}

// --- toeic_images ---

// ToeicImages implements toeic.ImageRepository on "toeic_images".
type ToeicImages struct{ db *sql.DB }

// NewToeicImages returns the ToeicImages repository.
func NewToeicImages(db *sql.DB) *ToeicImages { return &ToeicImages{db: db} }

var _ toeic.ImageRepository = (*ToeicImages)(nil)

// Get returns a picture.
func (r *ToeicImages) Get(ctx context.Context, key, slot string) (toeic.Image, error) {
	img := toeic.Image{GroupKey: key, Slot: slot}
	err := r.db.QueryRowContext(ctx, "SELECT mime, data, origin, created_at FROM toeic_images WHERE group_key = ? AND slot = ?", key, slot).
		Scan(&img.MIME, &img.Data, &img.Origin, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return toeic.Image{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Image{}, fmt.Errorf("mysql get toeic image: %w", err)
	}
	img.CreatedAt = utc(img.CreatedAt)
	return img, nil
}

// Save stores a picture, replacing the slot's previous one.
func (r *ToeicImages) Save(ctx context.Context, img toeic.Image) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO toeic_images (group_key, slot, mime, data, origin, created_at) VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE mime = VALUES(mime), data = VALUES(data), origin = VALUES(origin), created_at = VALUES(created_at)`,
		img.GroupKey, img.Slot, img.MIME, img.Data, img.Origin, utc(img.CreatedAt))
	if err != nil {
		return fmt.Errorf("mysql save toeic image: %w", err)
	}
	return nil
}

// Delete removes a picture.
func (r *ToeicImages) Delete(ctx context.Context, key, slot string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM toeic_images WHERE group_key = ? AND slot = ?", key, slot); err != nil {
		return fmt.Errorf("mysql delete toeic image: %w", err)
	}
	return nil
}

// Has returns the slots with a picture of each group.
func (r *ToeicImages) Has(ctx context.Context, keys []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, "SELECT group_key, slot FROM toeic_images WHERE group_key IN ("+placeholders(len(keys))+")", args(keys)...)
	if err != nil {
		return nil, fmt.Errorf("mysql list toeic images: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var k, slot string
		if err := rows.Scan(&k, &slot); err != nil {
			return nil, fmt.Errorf("mysql scan toeic image: %w", err)
		}
		out[k] = append(out[k], slot)
	}
	return out, rows.Err()
}

// --- toeic_tests ---

// ToeicTests implements toeic.TestRepository on "toeic_tests".
type ToeicTests struct{ db *sql.DB }

// NewToeicTests returns the ToeicTests repository.
func NewToeicTests(db *sql.DB) *ToeicTests { return &ToeicTests{db: db} }

var _ toeic.TestRepository = (*ToeicTests)(nil)

const toeicTestCols = "id, test_key, title, kind, ord, status, group_keys, created_at, updated_at"

func (r *ToeicTests) query(ctx context.Context, where string, a ...any) ([]toeic.Test, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+toeicTestCols+" FROM toeic_tests "+where+" ORDER BY ord, test_key", a...)
	if err != nil {
		return nil, fmt.Errorf("mysql list toeic tests: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []toeic.Test
	for rows.Next() {
		var t toeic.Test
		var kind, status string
		var groups []byte
		if err := rows.Scan(&t.ID, &t.Key, &t.Title, &kind, &t.Order, &status, &groups, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("mysql scan toeic test: %w", err)
		}
		if err := fromJSON(groups, &t.Groups); err != nil {
			return nil, err
		}
		t.Kind, t.Status, t.CreatedAt, t.UpdatedAt = toeic.TestKind(kind), toeic.Status(status), utc(t.CreatedAt), utc(t.UpdatedAt)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list toeic tests: %w", err)
	}
	return out, nil
}

// List returns the tests by order.
func (r *ToeicTests) List(ctx context.Context, onlyPublished bool) ([]toeic.Test, error) {
	if onlyPublished {
		return r.query(ctx, "WHERE status = ?", string(toeic.StatusPublished))
	}
	return r.query(ctx, "")
}

// Get returns a test.
func (r *ToeicTests) Get(ctx context.Context, key string) (toeic.Test, error) {
	list, err := r.query(ctx, "WHERE test_key = ?", key)
	if err != nil {
		return toeic.Test{}, err
	}
	if len(list) == 0 {
		return toeic.Test{}, toeic.ErrNotFound
	}
	return list[0], nil
}

// Save creates or replaces a test, keeping its creation time.
func (r *ToeicTests) Save(ctx context.Context, t toeic.Test) error {
	groups := t.Groups
	if groups == nil {
		groups = []string{}
	}
	body, err := toJSON(groups)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO toeic_tests ("+toeicTestCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE title = VALUES(title), kind = VALUES(kind), ord = VALUES(ord), status = VALUES(status),
group_keys = VALUES(group_keys), updated_at = VALUES(updated_at)`,
		newID(), t.Key, t.Title, string(t.Kind), t.Order, string(t.Status), body, utc(t.CreatedAt), utc(t.UpdatedAt))
	if err != nil {
		return fmt.Errorf("mysql save toeic test: %w", err)
	}
	return nil
}

// SetStatus changes the status of a test.
func (r *ToeicTests) SetStatus(ctx context.Context, key string, s toeic.Status) error {
	res, err := r.db.ExecContext(ctx, "UPDATE toeic_tests SET status = ? WHERE test_key = ?", string(s), key)
	if err != nil {
		return fmt.Errorf("mysql set toeic test status: %w", err)
	}
	if ok, err := changed(res); err != nil {
		return err
	} else if !ok {
		return toeic.ErrNotFound
	}
	return nil
}

// --- toeic_attempts ---

// toeicAttemptBody is the JSON body of an attempt row.
type toeicAttemptBody struct {
	Source      toeic.Source `json:"source"`
	GroupKeys   []string     `json:"groupKeys"`
	QuestionIDs []string     `json:"questionIds"`
	toeic.State
}

// ToeicAttempts implements toeic.AttemptRepository on "toeic_attempts".
type ToeicAttempts struct{ db *sql.DB }

// NewToeicAttempts returns the ToeicAttempts repository.
func NewToeicAttempts(db *sql.DB) *ToeicAttempts { return &ToeicAttempts{db: db} }

var _ toeic.AttemptRepository = (*ToeicAttempts)(nil)

const toeicAttemptCols = "id, user_id, status, mode, counted, body, result, started_at, saved_at, submitted_at"

func attemptRow(a toeic.Attempt) (body []byte, result any, err error) {
	body, err = toJSON(toeicAttemptBody{Source: a.Source, GroupKeys: a.GroupKeys, QuestionIDs: a.QuestionIDs, State: a.State()})
	if err != nil {
		return nil, nil, err
	}
	if a.Result != nil {
		b, err := toJSON(a.Result)
		if err != nil {
			return nil, nil, err
		}
		result = b
	}
	return body, result, nil
}

// Create stores a new attempt and its groups.
func (r *ToeicAttempts) Create(ctx context.Context, a toeic.Attempt) (toeic.Attempt, error) {
	a.ID = newID()
	body, result, err := attemptRow(a)
	if err != nil {
		return toeic.Attempt{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return toeic.Attempt{}, fmt.Errorf("mysql begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO toeic_attempts (id, user_id, status, test_key, mode, counted, body, result, started_at,
saved_at, submitted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, a.ID, a.UserID, string(a.Status), a.Source.TestKey, string(a.Mode),
		a.Counted, body, result, utc(a.StartedAt), utc(a.SavedAt), nullTime(a.SubmittedAt)); err != nil {
		return toeic.Attempt{}, fmt.Errorf("mysql insert toeic attempt: %w", err)
	}
	for _, k := range a.GroupKeys {
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO toeic_attempt_groups (attempt_id, group_key) VALUES (?, ?)", a.ID, k); err != nil {
			return toeic.Attempt{}, fmt.Errorf("mysql insert toeic attempt group: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return toeic.Attempt{}, fmt.Errorf("mysql commit: %w", err)
	}
	return a, nil
}

func (r *ToeicAttempts) query(ctx context.Context, tail string, a ...any) ([]toeic.Attempt, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+toeicAttemptCols+" FROM toeic_attempts "+tail, a...)
	if err != nil {
		return nil, fmt.Errorf("mysql list toeic attempts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []toeic.Attempt
	for rows.Next() {
		var at toeic.Attempt
		var status, mode string
		var body, result []byte
		var submitted sql.NullTime
		if err := rows.Scan(&at.ID, &at.UserID, &status, &mode, &at.Counted, &body, &result, &at.StartedAt, &at.SavedAt, &submitted); err != nil {
			return nil, fmt.Errorf("mysql scan toeic attempt: %w", err)
		}
		var b toeicAttemptBody
		if err := fromJSON(body, &b); err != nil {
			return nil, err
		}
		at.Source, at.GroupKeys, at.QuestionIDs = b.Source, b.GroupKeys, b.QuestionIDs
		at.SetState(b.State)
		if len(result) > 0 {
			var res toeic.Result
			if err := fromJSON(result, &res); err != nil {
				return nil, err
			}
			at.Result = &res
		}
		at.Status, at.Mode = toeic.AttemptStatus(status), toeic.Mode(mode)
		at.StartedAt, at.SavedAt, at.SubmittedAt = utc(at.StartedAt), utc(at.SavedAt), timeOf(submitted)
		out = append(out, at)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list toeic attempts: %w", err)
	}
	return out, nil
}

// Get returns the user's attempt.
func (r *ToeicAttempts) Get(ctx context.Context, userID, id string) (toeic.Attempt, error) {
	list, err := r.query(ctx, "WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return toeic.Attempt{}, err
	}
	if len(list) == 0 {
		return toeic.Attempt{}, toeic.ErrNotFound
	}
	return list[0], nil
}

// Save stores the progress of an attempt in progress.
func (r *ToeicAttempts) Save(ctx context.Context, a toeic.Attempt) error {
	body, _, err := attemptRow(a)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, "UPDATE toeic_attempts SET body = ?, saved_at = ? WHERE id = ? AND user_id = ? AND status = ?",
		body, utc(a.SavedAt), a.ID, a.UserID, string(toeic.AttemptInProgress))
	if err != nil {
		return fmt.Errorf("mysql save toeic attempt: %w", err)
	}
	if ok, err := changed(res); err != nil {
		return err
	} else if !ok {
		return toeic.ErrAttemptSubmitted
	}
	return nil
}

// Submit stores the result and the submitted status.
func (r *ToeicAttempts) Submit(ctx context.Context, a toeic.Attempt) error {
	body, result, err := attemptRow(a)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `UPDATE toeic_attempts SET body = ?, result = ?, status = ?, counted = ?, saved_at = ?, submitted_at = ?
WHERE id = ? AND user_id = ? AND status = ?`, body, result, string(a.Status), a.Counted, utc(a.SavedAt), nullTime(a.SubmittedAt),
		a.ID, a.UserID, string(toeic.AttemptInProgress))
	if err != nil {
		return fmt.Errorf("mysql submit toeic attempt: %w", err)
	}
	if ok, err := changed(res); err != nil {
		return err
	} else if !ok {
		return toeic.ErrAttemptSubmitted
	}
	return nil
}

// List returns the user's attempts matching f, newest first.
func (r *ToeicAttempts) List(ctx context.Context, userID string, f toeic.AttemptFilter) ([]toeic.Attempt, error) {
	conds, a := []string{"user_id = ?"}, []any{userID}
	if f.TestKey != "" {
		conds, a = append(conds, "test_key = ?"), append(a, f.TestKey)
	}
	if f.Status != "" {
		conds, a = append(conds, "status = ?"), append(a, string(f.Status))
	}
	if f.Counted {
		conds = append(conds, "counted = 1")
	}
	tail := "WHERE " + strings.Join(conds, " AND ") + " ORDER BY started_at DESC, id DESC"
	if f.Limit > 0 {
		tail += fmt.Sprintf(" LIMIT %d", f.Limit)
	}
	return r.query(ctx, tail, a...)
}

// InProgress returns the user's attempt in progress of a test.
func (r *ToeicAttempts) InProgress(ctx context.Context, userID, testKey string) (toeic.Attempt, error) {
	list, err := r.List(ctx, userID, toeic.AttemptFilter{TestKey: testKey, Status: toeic.AttemptInProgress, Limit: 1})
	if err != nil {
		return toeic.Attempt{}, err
	}
	if len(list) == 0 {
		return toeic.Attempt{}, toeic.ErrNotFound
	}
	return list[0], nil
}

// AnsweredQuestions returns the questions of the user's submitted attempts.
func (r *ToeicAttempts) AnsweredQuestions(ctx context.Context, userID string) (map[string]bool, error) {
	list, err := r.List(ctx, userID, toeic.AttemptFilter{Status: toeic.AttemptSubmitted})
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, a := range list {
		for _, id := range a.QuestionIDs {
			out[id] = true
		}
	}
	return out, nil
}

// WithGroup returns every submitted attempt having the group.
func (r *ToeicAttempts) WithGroup(ctx context.Context, groupKey string) ([]toeic.Attempt, error) {
	return r.query(ctx, `WHERE status = ? AND id IN (SELECT attempt_id FROM toeic_attempt_groups WHERE group_key = ?)
ORDER BY started_at DESC, id DESC`, string(toeic.AttemptSubmitted), groupKey)
}

// UpdateResult replaces the result of a submitted attempt.
func (r *ToeicAttempts) UpdateResult(ctx context.Context, id string, res toeic.Result) error {
	b, err := toJSON(res)
	if err != nil {
		return err
	}
	out, err := r.db.ExecContext(ctx, "UPDATE toeic_attempts SET result = ? WHERE id = ?", b, id)
	if err != nil {
		return fmt.Errorf("mysql update toeic result: %w", err)
	}
	if ok, err := changed(out); err != nil {
		return err
	} else if !ok {
		return toeic.ErrNotFound
	}
	return nil
}

// --- toeic_mistakes ---

// ToeicMistakes implements toeic.MistakeRepository on "toeic_mistakes".
type ToeicMistakes struct{ db *sql.DB }

// NewToeicMistakes returns the ToeicMistakes repository.
func NewToeicMistakes(db *sql.DB) *ToeicMistakes { return &ToeicMistakes{db: db} }

var _ toeic.MistakeRepository = (*ToeicMistakes)(nil)

func (r *ToeicMistakes) query(ctx context.Context, where string, a ...any) ([]toeic.Mistake, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, question_id, group_key, part, tags, streak, last_wrong_at, added_at
FROM toeic_mistakes `+where+" ORDER BY last_wrong_at, question_id", a...)
	if err != nil {
		return nil, fmt.Errorf("mysql list toeic mistakes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []toeic.Mistake
	for rows.Next() {
		var m toeic.Mistake
		var part int
		var tags []byte
		if err := rows.Scan(&m.UserID, &m.QuestionID, &m.GroupKey, &part, &tags, &m.Streak, &m.LastWrongAt, &m.AddedAt); err != nil {
			return nil, fmt.Errorf("mysql scan toeic mistake: %w", err)
		}
		if err := fromJSON(tags, &m.Tags); err != nil {
			return nil, err
		}
		m.Part, m.LastWrongAt, m.AddedAt = toeic.Part(part), utc(m.LastWrongAt), utc(m.AddedAt)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql list toeic mistakes: %w", err)
	}
	return out, nil
}

// List returns the user's entries, oldest wrong first.
func (r *ToeicMistakes) List(ctx context.Context, userID string, part toeic.Part) ([]toeic.Mistake, error) {
	if part != 0 {
		return r.query(ctx, "WHERE user_id = ? AND part = ?", userID, int(part))
	}
	return r.query(ctx, "WHERE user_id = ?", userID)
}

// Get returns an entry.
func (r *ToeicMistakes) Get(ctx context.Context, userID, qid string) (toeic.Mistake, error) {
	list, err := r.query(ctx, "WHERE user_id = ? AND question_id = ?", userID, qid)
	if err != nil {
		return toeic.Mistake{}, err
	}
	if len(list) == 0 {
		return toeic.Mistake{}, toeic.ErrNotFound
	}
	return list[0], nil
}

// Upsert stores an entry.
func (r *ToeicMistakes) Upsert(ctx context.Context, m toeic.Mistake) error {
	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}
	b, err := toJSON(tags)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO toeic_mistakes (user_id, question_id, group_key, part, tags, streak, last_wrong_at, added_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE group_key = VALUES(group_key), part = VALUES(part), tags = VALUES(tags),
streak = VALUES(streak), last_wrong_at = VALUES(last_wrong_at)`,
		m.UserID, m.QuestionID, m.GroupKey, int(m.Part), b, m.Streak, utc(m.LastWrongAt), utc(m.AddedAt))
	if err != nil {
		return fmt.Errorf("mysql save toeic mistake: %w", err)
	}
	return nil
}

// Delete removes an entry.
func (r *ToeicMistakes) Delete(ctx context.Context, userID, qid string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM toeic_mistakes WHERE user_id = ? AND question_id = ?", userID, qid); err != nil {
		return fmt.Errorf("mysql delete toeic mistake: %w", err)
	}
	return nil
}

// Count returns how many entries the user has.
func (r *ToeicMistakes) Count(ctx context.Context, userID string) (int, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM toeic_mistakes WHERE user_id = ?", userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("mysql count toeic mistakes: %w", err)
	}
	return n, nil
}

// --- toeic_goals ---

// ToeicGoals implements toeic.GoalRepository on "toeic_goals".
type ToeicGoals struct{ db *sql.DB }

// NewToeicGoals returns the ToeicGoals repository.
func NewToeicGoals(db *sql.DB) *ToeicGoals { return &ToeicGoals{db: db} }

var _ toeic.GoalRepository = (*ToeicGoals)(nil)

// Get returns the user's goal.
func (r *ToeicGoals) Get(ctx context.Context, userID string) (toeic.Goal, error) {
	g := toeic.Goal{UserID: userID}
	var at time.Time
	err := r.db.QueryRowContext(ctx, "SELECT target, exam_date, updated_at FROM toeic_goals WHERE user_id = ?", userID).
		Scan(&g.Target, &g.ExamDate, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return toeic.Goal{}, toeic.ErrNotFound
	}
	if err != nil {
		return toeic.Goal{}, fmt.Errorf("mysql get toeic goal: %w", err)
	}
	g.UpdatedAt = utc(at)
	return g, nil
}

// Save stores the user's goal.
func (r *ToeicGoals) Save(ctx context.Context, g toeic.Goal) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO toeic_goals (user_id, target, exam_date, updated_at) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE target = VALUES(target), exam_date = VALUES(exam_date), updated_at = VALUES(updated_at)`,
		g.UserID, g.Target, g.ExamDate, utc(g.UpdatedAt))
	if err != nil {
		return fmt.Errorf("mysql save toeic goal: %w", err)
	}
	return nil
}

// Delete removes the user's goal.
func (r *ToeicGoals) Delete(ctx context.Context, userID string) error {
	if _, err := r.db.ExecContext(ctx, "DELETE FROM toeic_goals WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("mysql delete toeic goal: %w", err)
	}
	return nil
}
