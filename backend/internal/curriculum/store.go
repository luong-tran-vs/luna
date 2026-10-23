package curriculum

import (
	"context"
	"sync"
	"time"
)

// Course is what the database keeps of a level's course (added 2026-10-23): the approved plan
// that learners and generation use, and the draft the admin is editing. A draft written by the AI
// may name topics the catalogue does not have yet, so it is kept here and not on the topics.
type Course struct {
	Level    string
	Approved *Plan
	Draft    *Plan
	// ApprovedAt is zero while nothing was approved.
	ApprovedAt time.Time
	UpdatedAt  time.Time
}

// Repository keeps one Course per level.
type Repository interface {
	// Get returns the course of level, or ErrNoPlan.
	Get(ctx context.Context, level string) (Course, error)
	// Save writes the course of c.Level, replacing it.
	Save(ctx context.Context, c Course) error
}

// cacheTTL is how long an approved plan read from the database is kept for learners: the learning
// path asks for every topic of every level. Development and production share one database, so an
// approval made elsewhere shows here within this time.
const cacheTTL = 30 * time.Second

type cached struct {
	plan *Plan
	at   time.Time
}

// planCache keeps the approved plans read for learners.
type planCache struct {
	mu    sync.Mutex
	plans map[string]cached
}

func (c *planCache) get(level string, now time.Time) (*Plan, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.plans[level]
	if !ok || now.Sub(e.at) > cacheTTL {
		return nil, false
	}
	return e.plan, true
}

func (c *planCache) put(level string, p *Plan, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.plans == nil {
		c.plans = map[string]cached{}
	}
	c.plans[level] = cached{plan: p, at: now}
}

func (c *planCache) drop(level string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.plans, level)
}
