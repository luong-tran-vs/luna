package export

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

// fakeRepo keeps raw documents per collection, with userId like the database; "users" holds
// password hashes and "sessions" tokens so tests can check they never leak.
type fakeRepo struct {
	mu      sync.Mutex
	colls   map[string][]Doc
	lessons map[string]Doc
	read    []string // collections read through UserDocs
	fail    bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{colls: map[string][]Doc{}, lessons: map[string]Doc{}}
}

func (f *fakeRepo) add(collection string, d Doc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.colls[collection] = append(f.colls[collection], d)
}

func (f *fakeRepo) Account(_ context.Context, userID string) (Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return Account{}, errors.New("db down")
	}
	for _, u := range f.colls["users"] {
		if u["_id"] == userID {
			return Account{ID: userID, Email: u["email"].(string), Role: u["role"].(string), CreatedAt: u["createdAt"].(time.Time)}, nil
		}
	}
	return Account{}, ErrNotFound
}

func (f *fakeRepo) UserDocs(_ context.Context, collection, userID string) ([]Doc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.read = append(f.read, collection)
	if !Allowed(collection) {
		return nil, ErrCollection
	}
	var out []Doc
	for _, d := range f.colls[collection] {
		if d["userId"] == userID {
			out = append(out, maps.Clone(d))
		}
	}
	return out, nil
}

func (f *fakeRepo) Lessons(_ context.Context, ids []string) ([]Doc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Doc
	for _, id := range ids {
		if l, ok := f.lessons[id]; ok {
			out = append(out, maps.Clone(l))
		}
	}
	return out, nil
}

func (f *fakeRepo) collectionsRead() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.read)
}

type fakeSettings struct {
	loc *time.Location
}

func (f fakeSettings) Values(context.Context, string) (map[string]any, error) {
	return map[string]any{"theme": "system", "dailyReviewLimit": 30, "timezone": f.loc.String()}, nil
}

func (f fakeSettings) Location(context.Context, string) (*time.Location, error) { return f.loc, nil }
