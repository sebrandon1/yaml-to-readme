package cmd

import (
	"context"
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

func TestWriteIndividualSummary(t *testing.T) {
	// Save and restore working directory since writeIndividualSummary uses os.Getwd()
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create a temp directory to act as the working directory (repo root)
	repoRoot, err := os.MkdirTemp("", "test_write_summary_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(repoRoot)
	}()

	assert.NoError(t, os.Chdir(repoRoot))

	// Create a baseDir with a YAML file
	baseDir := filepath.Join(repoRoot, "project")
	subDir := filepath.Join(baseDir, "subdir")
	assert.NoError(t, os.MkdirAll(subDir, 0755))

	testCases := []struct {
		name          string
		filePath      string
		summary       string
		expectedCache string
	}{
		{
			name:          "file in subdirectory",
			filePath:      filepath.Join(subDir, "config.yaml"),
			summary:       "This file configures the application.",
			expectedCache: "subdir_config.yaml.md",
		},
		{
			name:          "file in base directory",
			filePath:      filepath.Join(baseDir, "root.yaml"),
			summary:       "Root level configuration file.",
			expectedCache: "root.yaml.md",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := writeIndividualSummary(baseDir, tc.filePath, tc.summary)
			assert.NoError(t, err)

			// Verify cache directory was created
			cacheDir := filepath.Join(repoRoot, DefaultCacheDirName)
			_, err = os.Stat(cacheDir)
			assert.NoError(t, err, "cache directory should exist")

			// Verify cache file exists with correct name
			cacheFilePath := filepath.Join(cacheDir, tc.expectedCache)
			content, err := os.ReadFile(cacheFilePath)
			assert.NoError(t, err, "cache file should exist")
			assert.Equal(t, tc.summary, string(content), "cache file content should match summary")
		})
	}
}

func TestWriteIndividualSummaryFallback(t *testing.T) {
	// Test the fallback behavior when filePath is not under baseDir
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	repoRoot, err := os.MkdirTemp("", "test_write_summary_fallback_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(repoRoot)
	}()

	assert.NoError(t, os.Chdir(repoRoot))

	// Use a filePath that is not under baseDir
	baseDir := filepath.Join(repoRoot, "project")
	assert.NoError(t, os.MkdirAll(baseDir, 0755))

	otherDir := filepath.Join(repoRoot, "other")
	assert.NoError(t, os.MkdirAll(otherDir, 0755))

	err = writeIndividualSummary(baseDir, filepath.Join(otherDir, "external.yaml"), "External summary.")
	assert.NoError(t, err)

	// Should fall back to base name only
	cacheDir := filepath.Join(repoRoot, DefaultCacheDirName)
	cacheFilePath := filepath.Join(cacheDir, "external.yaml.md")
	content, err := os.ReadFile(cacheFilePath)
	assert.NoError(t, err, "cache file should exist with fallback name")
	assert.Equal(t, "External summary.", string(content))
}

func TestMockLLMProvider(t *testing.T) {
	mock := NewMockLLMProvider()

	// Test defaults
	assert.Equal(t, "mock", mock.Name())
	available, err := mock.Available(context.Background())
	assert.NoError(t, err)
	assert.True(t, available)

	// Test default response
	summary, err := mock.Summarize(context.Background(), "some content", "prompt: ")
	assert.NoError(t, err)
	assert.Equal(t, "This is a mock summary for testing purposes.", summary)

	// Test matched response
	mock.MockResponses["ConfigMap"] = "This is a ConfigMap."
	summary, err = mock.Summarize(context.Background(), "kind: ConfigMap", "prompt: ")
	assert.NoError(t, err)
	assert.Equal(t, "This is a ConfigMap.", summary)

	// Test unavailable
	mock.ModelAvailable = false
	available, err = mock.Available(context.Background())
	assert.NoError(t, err)
	assert.False(t, available)
}

func TestCreateProviderOpenAIMissingKey(t *testing.T) {
	origProvider := provider
	defer func() {
		provider = origProvider
	}()

	// createProvider with "openai" without OPENAI_API_KEY should return an error
	provider = "openai"
	t.Setenv("OPENAI_API_KEY", "")

	_, err := createProvider()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY")
}

func TestFindYAMLFilesHiddenDirectories(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test_yaml_finder_*")
	assert.NoError(t, err, "failed to create temp dir")
	defer func() {
		cerr := os.RemoveAll(tmpDir)
		if cerr != nil {
			assert.Fail(t, "failed to remove temp dir", cerr)
		}
	}()

	// Create test structure:
	// tmpDir/
	//   ├── regular.yaml
	//   ├── regular_dir/
	//   │   └── nested.yaml
	//   └── .hidden/
	//       ├── hidden.yaml
	//       └── subdir/
	//           └── deep.yaml

	regularDir := filepath.Join(tmpDir, "regular_dir")
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	hiddenSubdir := filepath.Join(hiddenDir, "subdir")

	// Create directories
	assert.NoError(t, os.MkdirAll(regularDir, 0755))
	assert.NoError(t, os.MkdirAll(hiddenSubdir, 0755))

	// Create YAML files
	files := map[string]string{
		filepath.Join(tmpDir, "regular.yaml"):    "test: regular",
		filepath.Join(regularDir, "nested.yaml"): "test: nested",
		filepath.Join(hiddenDir, "hidden.yaml"):  "test: hidden",
		filepath.Join(hiddenSubdir, "deep.yaml"): "test: deep",
	}

	for filePath, content := range files {
		assert.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	// Test with includeHidden=false (default behavior)
	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 2, "Should find 2 files when hidden directories are excluded")

	// Convert to relative paths for easier assertion
	var relPaths []string
	for _, file := range yamlFiles {
		rel, _ := filepath.Rel(tmpDir, file)
		relPaths = append(relPaths, rel)
	}
	assert.Contains(t, relPaths, "regular.yaml")
	assert.Contains(t, relPaths, filepath.Join("regular_dir", "nested.yaml"))

	// Test with includeHidden=true
	yamlFiles, err = findYAMLFiles(tmpDir, true)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 4, "Should find 4 files when hidden directories are included")

	// Convert to relative paths for easier assertion
	relPaths = []string{}
	for _, file := range yamlFiles {
		rel, _ := filepath.Rel(tmpDir, file)
		relPaths = append(relPaths, rel)
	}
	assert.Contains(t, relPaths, "regular.yaml")
	assert.Contains(t, relPaths, filepath.Join("regular_dir", "nested.yaml"))
	assert.Contains(t, relPaths, filepath.Join(".hidden", "hidden.yaml"))
	assert.Contains(t, relPaths, filepath.Join(".hidden", "subdir", "deep.yaml"))
}
