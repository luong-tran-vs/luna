package lesson

import (
	"slices"
	"testing"
)

// The same samples are used by frontend/src/app/shared/utils/tokenize.spec.ts.
func TestWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want []string
	}{
		{"We went to the park.", []string{"We", "went", "to", "the", "park"}},
		{"I don't know.", []string{"I", "don't", "know"}},
		{"A well-known café.", []string{"A", "well-known", "café"}},
		{"It’s rock'n'roll!", []string{"It’s", "rock'n'roll"}},
		{"U.S. troops at 9.30 a.m.", []string{"U", "S", "troops", "at", "a", "m"}},
		{`"Stop!" she said — twice.`, []string{"Stop", "she", "said", "twice"}},
		{"'quoted' -dash- end-", []string{"quoted", "dash", "end"}},
		{"123 456", nil},
	}
	for _, tt := range tests {
		if got := Words(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("Words(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
