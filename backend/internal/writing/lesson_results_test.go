package writing

import "testing"

func TestLessonResults(t *testing.T) {
	t.Parallel()
	e := newEnv()
	e.addGraded("u1", 1, GradeDone, tense("t1"), article("a1")) // lesson lw1
	e.addGraded("u1", 2, GradeDone, tense("t2"))                // lesson lw2
	e.addGraded("u1", 3, GradeDone, tense("t3"))                // lesson lw3, another topic
	e.addGraded("u1", 4, GradePending)                          // lesson lw4, not graded yet
	scores := map[string][]int{"w1": {4, 4, 5, 5}, "w2": {3, 3, 3, 3}, "w3": {1, 1, 1, 1}}
	e.repo.mu.Lock()
	for id, ss := range scores {
		w := e.repo.byID[id]
		for _, s := range ss {
			w.Grade.Criteria = append(w.Grade.Criteria, Criterion{Name: CriterionTask, Score: s})
		}
		e.repo.byID[id] = w
	}
	e.repo.mu.Unlock()

	graded, avg, mistakes, err := e.svc.LessonResults(t.Context(), "u1", []string{"lw1", "lw2", "lw4"})
	if err != nil {
		t.Fatal(err)
	}
	// (4.5 + 3) / 2
	if graded != 2 || avg != 3.8 {
		t.Errorf("graded %d, average %v", graded, avg)
	}
	if len(mistakes) != 2 || mistakes[0].Type != MistakeTense || mistakes[0].Count != 2 || mistakes[1].Count != 1 {
		t.Errorf("mistakes = %+v", mistakes)
	}
	if graded, avg, mistakes, err := e.svc.LessonResults(t.Context(), "u2", []string{"lw1"}); err != nil || graded != 0 || avg != 0 || len(mistakes) != 0 {
		t.Errorf("u2: %d %v %v %v", graded, avg, mistakes, err)
	}
}
