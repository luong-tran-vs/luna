package games

import (
	"math/rand/v2"
	"sync"
	"time"
)

// Deps are the collaborators of Service; every one is required except AI (nil: no extra sentences).
type Deps struct {
	Bests     BestRepository
	Daily     DailyRepository
	Spots     SpotRepository
	Cards     Cards
	Pictures  Pictures
	Meanings  Meanings
	Lessons   LessonWords
	Study     Study
	Words     WordMisses
	Verbs     Verbs
	Grammar   Grammar
	AI        SpotAI
	StudyDays StudyDays
	Timezones Timezones
	Now       func() time.Time
	// Seed seeds the random rounds; 0 uses a random seed (tests set it).
	Seed uint64
}

// Service builds the rounds and takes their results.
type Service struct {
	d Deps
	// mu guards rnd, shared by the requests.
	mu  sync.Mutex
	rnd *rand.Rand
}

// NewService returns a Service on d.
func NewService(d Deps) *Service {
	seed := d.Seed
	if seed == 0 {
		seed = rand.Uint64()
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Service{d: d, rnd: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))}
}

// newRand returns a generator for one round, drawn from the service's own.
func (s *Service) newRand() *rand.Rand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rand.New(rand.NewPCG(s.rnd.Uint64(), s.rnd.Uint64()))
}
