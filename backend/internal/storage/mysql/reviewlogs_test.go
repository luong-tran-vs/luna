package mysql

import (
	"testing"
	"time"

	"github.com/luongtran/luna/backend/internal/vocab"
)

func TestReviewLogs(t *testing.T) {
	t.Parallel()
	db := testDB(t)
	r := NewReviewLogs(db)
	ctx := t.Context()
	user, other, card1, card2 := newID(), newID(), newID(), newID()
	at := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	add := func(u, c string, ctxt vocab.Context, when time.Time) {
		t.Helper()
		err := r.Add(ctx, vocab.ReviewLog{
			UserID: u, CardID: c, Rating: vocab.Good, Mode: vocab.ModeFlip, Context: ctxt, ReviewedAt: when,
			Before: vocab.Schedule{Due: at, Reps: 2, Stability: 1.5, State: vocab.StateReview, LastReview: at},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	add(user, card1, vocab.ContextDaily, at)
	add(user, card1, vocab.ContextDaily, at.Add(-time.Second))
	add(user, card2, vocab.ContextDaily, at.Add(time.Hour))
	add(user, card2, vocab.ContextFree, at.Add(time.Hour))
	add(other, card1, vocab.ContextDaily, at)
	// A zero Before schedule (a legacy card) is stored too.
	if err := r.Add(ctx, vocab.ReviewLog{UserID: user, CardID: card2, Rating: vocab.Again, Mode: vocab.ModeListen, Context: vocab.ContextFree, ReviewedAt: at}); err != nil {
		t.Fatal(err)
	}

	if n, err := r.CountSince(ctx, user, vocab.ContextDaily, at); err != nil || n != 2 {
		t.Fatalf("CountSince daily = %d, %v", n, err)
	}
	if n, _ := r.CountSince(ctx, user, vocab.ContextDaily, at.Add(-time.Hour)); n != 3 {
		t.Fatalf("CountSince wider = %d", n)
	}
	if n, _ := r.CountSince(ctx, user, vocab.ContextFree, at); n != 2 {
		t.Fatalf("CountSince free = %d", n)
	}
	if n, _ := r.CountSince(ctx, user, vocab.ContextDaily, at.Add(2*time.Hour)); n != 0 {
		t.Fatalf("CountSince future = %d", n)
	}
	if n, _ := r.CountSince(ctx, newID(), vocab.ContextDaily, at.Add(-time.Hour)); n != 0 {
		t.Fatalf("CountSince stranger = %d", n)
	}

	var state string
	if err := db.QueryRowContext(ctx, "SELECT CAST(state_before AS CHAR) FROM review_logs WHERE card_id = ? AND mode = 'listen'", card2).Scan(&state); err != nil || state == "" {
		t.Fatalf("state_before = %q, %v", state, err)
	}

	if n, err := r.DeleteByCard(ctx, user, card1); err != nil || n != 2 {
		t.Fatalf("DeleteByCard = %d, %v", n, err)
	}
	if n, _ := r.DeleteByCard(ctx, user, card1); n != 0 {
		t.Fatalf("DeleteByCard again = %d", n)
	}
	if n, err := r.DeleteByCard(ctx, user, "bad"); n != 0 || err != nil {
		t.Fatalf("DeleteByCard malformed = %d, %v", n, err)
	}
	if n, _ := r.CountSince(ctx, other, vocab.ContextDaily, at); n != 1 {
		t.Fatalf("other user's log was deleted: %d", n)
	}
}
