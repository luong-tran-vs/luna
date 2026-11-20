package toeic

import (
	"math/rand/v2"
	"sync"
	"time"
)

// Deps are the collaborators of Service; every one is required except Drawer and Writer.
type Deps struct {
	Groups    GroupRepository
	Images    ImageRepository
	Tests     TestRepository
	Attempts  AttemptRepository
	Mistakes  MistakeRepository
	Goals     GoalRepository
	StudyDays StudyDays
	Cards     Cards
	// Drawer draws the pictures; nil: drawing answers that the image AI is not set up.
	Drawer Drawer
	// Writer drafts groups with AI; nil: "Nhờ AI soạn" answers that the AI is not set up.
	Writer    GroupWriter
	Timezones Timezones
	Now       func() time.Time
	// Seed seeds the random picks; 0 uses a random seed (tests set it).
	Seed uint64
}

// Service is the TOEIC practice: content for the admin, attempts for the learners.
type Service struct {
	d Deps
	// mu guards rnd, shared by the requests.
	mu  sync.Mutex
	rnd *rand.Rand
	// batch is the background batch of missing Part 1 photos.
	batch photoBatch
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

// newRand returns a generator for one request, drawn from the service's own.
func (s *Service) newRand() *rand.Rand {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rand.New(rand.NewPCG(s.rnd.Uint64(), s.rnd.Uint64()))
}

func (s *Service) now() time.Time { return s.d.Now().UTC() }
