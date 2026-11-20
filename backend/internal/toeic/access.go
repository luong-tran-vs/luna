package toeic

import (
	"context"
	"slices"
)

// guestParts are the parts a guest may practise on their own (FR-045).
var guestParts = []Part{1, 2, 5}

// guestTestKey returns the test a guest may take: the published Mini Test with the lowest order, ""
// when there is none (research R10).
func (s *Service) guestTestKey(ctx context.Context) (string, error) {
	tests, err := s.d.Tests.List(ctx, true)
	if err != nil {
		return "", err
	}
	return guestTestOf(tests), nil
}

// guestTestOf returns the key of the guest's Mini Test among published tests (sorted by order).
func guestTestOf(tests []Test) string {
	best := -1
	for i, t := range tests {
		if t.Kind.Mini() && t.Status == StatusPublished && (best < 0 || t.Order < tests[best].Order) {
			best = i
		}
	}
	if best < 0 {
		return ""
	}
	return tests[best].Key
}

// checkStart returns ErrMembersOnly when a guest asks for an attempt only members get.
func (s *Service) checkStart(ctx context.Context, isGuest bool, src Source) error {
	if !isGuest {
		return nil
	}
	switch src.Kind {
	case SourceTest:
		key, err := s.guestTestKey(ctx)
		if err != nil {
			return err
		}
		if key == "" || src.TestKey != key {
			return ErrMembersOnly
		}
		return nil
	case SourcePart:
		if slices.Contains(guestParts, src.Part) {
			return nil
		}
	}
	return ErrMembersOnly
}

// locked reports whether a guest sees test as members only.
func locked(test Test, isGuest bool, guestKey string) bool {
	return isGuest && test.Key != guestKey
}
