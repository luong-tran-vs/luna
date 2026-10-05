package storage_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/luongtran/luna/backend/internal/storage"
)

// fakeStore is only a value to tell the openers apart; Registry never calls its methods.
type fakeStore struct {
	storage.Store
	name string
}

func opener(name string) storage.Opener {
	return func() (storage.Store, error) { return &fakeStore{name: name}, nil }
}

func TestRegistryOpensTheNamedDriver(t *testing.T) {
	t.Parallel()
	reg := storage.NewRegistry()
	reg.Register("mongo", opener("mongo"))
	reg.Register("mysql", opener("mysql"))

	for name, want := range map[string]string{"mongo": "mongo", "mysql": "mysql", " MySQL ": "mysql", "MONGO": "mongo"} {
		s, err := reg.Open(name)
		if err != nil {
			t.Fatalf("Open(%q): %v", name, err)
		}
		if got := s.(*fakeStore).name; got != want {
			t.Errorf("Open(%q) = %s, want %s", name, got, want)
		}
	}
}

func TestRegistryUnknownDriverListsTheKnownOnes(t *testing.T) {
	t.Parallel()
	reg := storage.NewRegistry()
	reg.Register("mysql", opener("mysql"))
	reg.Register("mongo", opener("mongo"))

	_, err := reg.Open("postgres")
	if !errors.Is(err, storage.ErrUnknownDriver) {
		t.Fatalf("err = %v, want ErrUnknownDriver", err)
	}
	if !strings.Contains(err.Error(), `"postgres"`) || !strings.Contains(err.Error(), "known: mongo, mysql") {
		t.Errorf("message = %q", err)
	}
	if got := strings.Join(reg.Drivers(), ","); got != "mongo,mysql" {
		t.Errorf("Drivers() = %s", got)
	}
}

func TestRegistryOpenErrorNamesTheDriverAndKeepsTheCause(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	reg := storage.NewRegistry()
	reg.Register("mongo", func() (storage.Store, error) { return nil, boom })

	s, err := reg.Open("Mongo")
	if s != nil || !errors.Is(err, boom) || !strings.Contains(err.Error(), "open mongo") {
		t.Fatalf("Open = %v, %v", s, err)
	}
}

func TestRegisterRejectsMistakes(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*storage.Registry){
		"empty name": func(r *storage.Registry) { r.Register("  ", opener("x")) },
		"nil opener": func(r *storage.Registry) { r.Register("x", nil) },
		"twice":      func(r *storage.Registry) { r.Register("x", opener("x")); r.Register("X", opener("x")) },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if recover() == nil {
					t.Error("Register did not panic")
				}
			}()
			fn(storage.NewRegistry())
		})
	}
}
