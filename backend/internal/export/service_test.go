package export

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

var hcm = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		panic(err)
	}
	return loc
}()

// seed stores data for u1 (hoc) and u2 (binh), plus sessions that must never be read.
func seed() *fakeRepo {
	r := newFakeRepo()
	created := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	r.add("users", Doc{"_id": "u1", "email": "hoc@example.com", "role": "learner", "createdAt": created, "passwordHash": "HASH-U1"})
	r.add("users", Doc{"_id": "u2", "email": "binh@example.com", "role": "admin", "createdAt": created, "passwordHash": "HASH-U2"})
	r.add("sessions", Doc{"_id": "s1", "userId": "u1", "tokenHash": "TOKEN-U1"})
	for _, u := range []string{"u1", "u2"} {
		r.add(CollCards, Doc{"_id": "card-" + u, "userId": u, "lemma": "go-" + u})
		r.add(CollReviewLogs, Doc{"_id": "log-" + u, "userId": u, "cardId": "card-" + u, "rating": 3})
		r.add(CollGoals, Doc{"_id": "goal-" + u, "userId": u, "topicId": "t1"})
		r.add(CollStudyDays, Doc{"_id": "day-" + u, "userId": u, "dayKey": "2026-09-30"})
		r.add(CollDictationResults, Doc{"_id": "dict-" + u, "userId": u, "lessonId": "l1", "sentenceIndex": 0})
	}
	r.add(CollLessonProgress, Doc{"_id": "p1", "userId": "u1", "lessonId": "l1"})
	r.add(CollLessonProgress, Doc{"_id": "p2", "userId": "u1", "lessonId": "gone"})
	r.add(CollLessonProgress, Doc{"_id": "p3", "userId": "u2", "lessonId": "l2"})
	r.lessons["l1"] = Doc{"_id": "l1", "title": "At the café"}
	r.lessons["l2"] = Doc{"_id": "l2", "title": "Binh's lesson"}
	return r
}

func newSvc(r *fakeRepo, now time.Time) *Service {
	return NewService(r, fakeSettings{loc: hcm}, func() time.Time { return now })
}

func TestBuildOnlyTheUsersData(t *testing.T) {
	t.Parallel()
	r := seed()
	now := time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC) // 2026-10-01 01:00 in Viet Nam
	e, name, err := newSvc(r, now).Build(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if name != "luna-export-20261001.json" {
		t.Fatalf("file name = %s", name)
	}
	if e.Version != 1 || !e.ExportedAt.Equal(now) || e.ExportedAt.Location().String() != "Asia/Ho_Chi_Minh" {
		t.Fatalf("header = %d %v", e.Version, e.ExportedAt)
	}
	if e.Account.ID != "u1" || e.Account.Email != "hoc@example.com" || e.Account.Role != "learner" {
		t.Fatalf("account = %+v", e.Account)
	}
	if e.Settings["timezone"] != "Asia/Ho_Chi_Minh" {
		t.Fatalf("settings = %v", e.Settings)
	}
	for name, docs := range map[string][]Doc{
		"cards": e.Cards, "reviewLogs": e.ReviewLogs, "goals": e.Goals, "studyDays": e.StudyDays,
		"dictationResults": e.DictationResults,
	} {
		if len(docs) != 1 || !strings.HasSuffix(docs[0]["_id"].(string), "-u1") {
			t.Errorf("%s = %v", name, docs)
		}
	}
	if len(e.LessonProgress) != 2 {
		t.Fatalf("lessonProgress = %v", e.LessonProgress)
	}
	for _, d := range append(slices.Clone(e.Cards), e.LessonProgress...) {
		if _, ok := d["userId"]; ok {
			t.Fatalf("userId kept in %v", d)
		}
	}
	// Lessons: the ones the learner started, a deleted one marked as such.
	if len(e.Lessons) != 2 || e.Lessons[0]["title"] != "At the café" ||
		e.Lessons[1]["id"] != "gone" || e.Lessons[1]["deleted"] != true {
		t.Fatalf("lessons = %v", e.Lessons)
	}

	out, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"passwordHash", "HASH-U1", "TOKEN-U1", "tokenHash", "binh@example.com", "-u2", "Binh's lesson"} {
		if strings.Contains(string(out), secret) {
			t.Errorf("export contains %q", secret)
		}
	}
	for _, c := range r.collectionsRead() {
		if !Allowed(c) {
			t.Errorf("read collection %q", c)
		}
	}
}

func TestBuildNewLearner(t *testing.T) {
	t.Parallel()
	r := newFakeRepo()
	r.add("users", Doc{"_id": "u3", "email": "moi@example.com", "role": "learner", "createdAt": time.Now()})
	e, _, err := newSvc(r, time.Now()).Build(t.Context(), "u3")
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"cards", "reviewLogs", "goals", "lessonProgress", "studyDays", "dictationResults", "lessons"} {
		if !strings.Contains(string(out), `"`+key+`":[]`) {
			t.Errorf("%s is not an empty list in %s", key, out)
		}
	}
}

func TestBuildErrors(t *testing.T) {
	t.Parallel()
	r := seed()
	if _, _, err := newSvc(r, time.Now()).Build(t.Context(), "nobody"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown user: %v", err)
	}
	r.fail = true
	if _, _, err := newSvc(r, time.Now()).Build(t.Context(), "u1"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("db error: %v", err)
	}
}
