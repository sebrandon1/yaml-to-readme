package cmd

import (
	"os"
	"path/filepath"
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

func TestParseExistingSummaries(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "yaml_details_*.md")
	assert.NoError(t, err, "failed to create temp file")
	defer func() {
		cerr := os.Remove(tmpFile.Name())
		if cerr != nil {
			assert.Fail(t, "failed to remove temp file", cerr)
		}
	}()

	content := `# YAML File Details

---

## [foo/](../foo/)
- [bar.yaml](../foo/bar.yaml): This is bar summary.
- [baz.yaml](../foo/baz.yaml): This is baz summary.

## [root/](../root/)
- [top.yaml](../root/top.yaml): Top level summary.`
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err, "failed to write to temp file")
	cerr := tmpFile.Close()
	assert.NoError(t, cerr, "failed to close temp file")

	summaries := parseExistingSummaries(tmpFile.Name())

	expect := map[string]string{
		filepath.Join("foo", "bar.yaml"):  "This is bar summary.",
		filepath.Join("foo", "baz.yaml"):  "This is baz summary.",
		filepath.Join("root", "top.yaml"): "Top level summary.",
	}

	assert.Equal(t, len(expect), len(summaries), "expected %d summaries, got %d", len(expect), len(summaries))
	for k, v := range expect {
		assert.Equal(t, v, summaries[k], "expected %q for %q, got %q", v, k, summaries[k])
	}
}

func TestParseSummaryLines(t *testing.T) {
	lines := []string{
		"# YAML File Details",
		"",
		"---",
		"",
		"## [foo/](../foo/)",
		"- [bar.yaml](../foo/bar.yaml): This is bar summary.",
		"- [baz.yaml](../foo/baz.yaml): This is baz summary.",
		"",
		"## [root/](../root/)",
		"- [top.yaml](../root/top.yaml): Top level summary.",
	}

	existing := make(map[string]string)
	parseSummaryLines(lines, existing)

	expect := map[string]string{
		filepath.Join("foo", "bar.yaml"):  "This is bar summary.",
		filepath.Join("foo", "baz.yaml"):  "This is baz summary.",
		filepath.Join("root", "top.yaml"): "Top level summary.",
	}

	assert.Equal(t, len(expect), len(existing), "expected %d summaries, got %d", len(expect), len(existing))
	for k, v := range expect {
		assert.Equal(t, v, existing[k], "expected %q for %q, got %q", v, k, existing[k])
	}
}
