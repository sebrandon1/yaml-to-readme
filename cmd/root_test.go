package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateToSentences(t *testing.T) {
	testCases := []struct {
		input    string
		n        int
		expected string
	}{
		{"This is one. This is two. This is three.", 2, "This is one. This is two."},
		{"One! Two? Three.", 2, "One! Two?"},
		{"No period here", 2, "No period here"},
		{"Sentence one. Sentence two.", 1, "Sentence one."},
	}
	for _, c := range testCases {
		assert.Equal(t, c.expected, truncateToSentences(c.input, c.n))
	}
}
