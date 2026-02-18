package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIntegrationBasicFlow tests the complete flow from YAML files to markdown output.
func TestIntegrationBasicFlow(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "integration_test_*")
	assert.NoError(t, err, "failed to create temp dir")
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create test directory structure:
	// tmpDir/
	//   ├── config.yaml
	//   ├── app/
	//   │   ├── deployment.yaml
	//   │   └── service.yaml
	//   └── database/
	//       └── postgres.yaml

	appDir := filepath.Join(tmpDir, "app")
	dbDir := filepath.Join(tmpDir, "database")
	assert.NoError(t, os.MkdirAll(appDir, 0755))
	assert.NoError(t, os.MkdirAll(dbDir, 0755))

	// Create test YAML files with identifiable content
	testFiles := map[string]string{
		filepath.Join(tmpDir, "config.yaml"):     "apiVersion: v1\nkind: ConfigMap\nname: app-config",
		filepath.Join(appDir, "deployment.yaml"): "apiVersion: apps/v1\nkind: Deployment\nname: web-app",
		filepath.Join(appDir, "service.yaml"):    "apiVersion: v1\nkind: Service\nname: web-service",
		filepath.Join(dbDir, "postgres.yaml"):    "apiVersion: v1\nkind: StatefulSet\nname: postgres-db",
	}

	for filePath, content := range testFiles {
		assert.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	// Create mock client with specific responses for each file
	mockClient := NewMockOllamaClient()
	mockClient.MockResponses = map[string]string{
		"kind: ConfigMap":   "This YAML file defines a ConfigMap for application configuration. It stores key-value pairs for the application.",
		"kind: Deployment":  "This YAML file defines a Kubernetes Deployment for the web application. It manages the desired state of application pods.",
		"kind: Service":     "This YAML file defines a Kubernetes Service for network access. It exposes the application to other services.",
		"kind: StatefulSet": "This YAML file defines a StatefulSet for PostgreSQL database. It manages stateful database instances with persistent storage.",
	}

	// Find YAML files
	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 4, "Should find 4 YAML files")

	// Process files with mock client
	summaries, processed, skipped := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	assert.Equal(t, 4, processed, "Should process 4 files")
	assert.Equal(t, 0, skipped, "Should skip 0 files")
	assert.Len(t, summaries, 4, "Should have 4 summaries")

	// Group and write markdown
	grouped := groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	// Read and verify the generated markdown
	mdPath := filepath.Join(tmpDir, MarkdownFileName)
	content, err := os.ReadFile(mdPath)
	assert.NoError(t, err)
	mdContent := string(content)

	// Verify structure
	assert.Contains(t, mdContent, "# YAML File Details")
	assert.Contains(t, mdContent, "## [./](.././)") // Root directory
	assert.Contains(t, mdContent, "## [app/](../app/)")
	assert.Contains(t, mdContent, "## [database/](../database/)")

	// Verify file entries
	assert.Contains(t, mdContent, "[config.yaml](.././config.yaml)")
	assert.Contains(t, mdContent, "[deployment.yaml](../app/deployment.yaml)")
	assert.Contains(t, mdContent, "[service.yaml](../app/service.yaml)")
	assert.Contains(t, mdContent, "[postgres.yaml](../database/postgres.yaml)")

	// Verify summaries are present
	assert.Contains(t, mdContent, "ConfigMap for application configuration")
	assert.Contains(t, mdContent, "Deployment for the web application")
	assert.Contains(t, mdContent, "Service for network access")
	assert.Contains(t, mdContent, "StatefulSet for PostgreSQL database")
}

// TestIntegrationRegenerateFlag tests the --regenerate flag functionality.
func TestIntegrationRegenerateFlag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "integration_test_regenerate_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a single YAML file
	testFile := filepath.Join(tmpDir, "test.yaml")
	assert.NoError(t, os.WriteFile(testFile, []byte("test: data"), 0644))

	mockClient := NewMockOllamaClient()
	mockClient.DefaultResponse = "First summary."

	// First run: generate summaries
	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	summaries, processed, _ := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	assert.Equal(t, 1, processed)
	grouped := groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	// Parse existing summaries
	mdPath := filepath.Join(tmpDir, MarkdownFileName)
	existingSummaries := parseExistingSummaries(mdPath)
	assert.Len(t, existingSummaries, 1)
	assert.Contains(t, existingSummaries["test.yaml"], "First summary")

	// Second run: without regenerate flag (should skip)
	_, processed, skipped := processYAMLFiles(yamlFiles, tmpDir, existingSummaries, mockClient, false)
	assert.Equal(t, 0, processed, "Should process 0 files (all skipped)")
	assert.Equal(t, 1, skipped, "Should skip 1 file")

	// Third run: with regenerate flag (should reprocess)
	mockClient.DefaultResponse = "Second summary."
	summaries, processed, skipped = processYAMLFiles(yamlFiles, tmpDir, existingSummaries, mockClient, true)
	assert.Equal(t, 1, processed, "Should process 1 file (regenerate)")
	assert.Equal(t, 0, skipped, "Should skip 0 files")
	assert.Contains(t, summaries[testFile], "Second summary")
}

// TestIntegrationHiddenDirectories tests the --include-hidden-directories flag.
func TestIntegrationHiddenDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "integration_test_hidden_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create structure with hidden directory
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	assert.NoError(t, os.MkdirAll(hiddenDir, 0755))

	regularFile := filepath.Join(tmpDir, "regular.yaml")
	hiddenFile := filepath.Join(hiddenDir, "hidden.yaml")
	assert.NoError(t, os.WriteFile(regularFile, []byte("regular: data"), 0644))
	assert.NoError(t, os.WriteFile(hiddenFile, []byte("hidden: data"), 0644))

	mockClient := NewMockOllamaClient()
	mockClient.DefaultResponse = "Test summary."

	// Without hidden directories
	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 1, "Should find 1 file (hidden excluded)")

	summaries, _, _ := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	grouped := groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	mdPath := filepath.Join(tmpDir, MarkdownFileName)
	content, err := os.ReadFile(mdPath)
	assert.NoError(t, err)
	assert.NotContains(t, string(content), ".hidden", "Should not include hidden directory")

	// With hidden directories
	yamlFiles, err = findYAMLFiles(tmpDir, true)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 2, "Should find 2 files (hidden included)")

	summaries, _, _ = processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	grouped = groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	content, err = os.ReadFile(mdPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), ".hidden", "Should include hidden directory")
}

// TestIntegrationMarkdownFormat tests that the markdown output format is correct.
func TestIntegrationMarkdownFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "integration_test_format_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create multiple files in different directories
	subdir1 := filepath.Join(tmpDir, "alpha")
	subdir2 := filepath.Join(tmpDir, "beta")
	assert.NoError(t, os.MkdirAll(subdir1, 0755))
	assert.NoError(t, os.MkdirAll(subdir2, 0755))

	files := map[string]string{
		filepath.Join(subdir1, "a.yaml"): "test: a",
		filepath.Join(subdir1, "b.yaml"): "test: b",
		filepath.Join(subdir2, "c.yaml"): "test: c",
	}

	for path, content := range files {
		assert.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	mockClient := NewMockOllamaClient()
	mockClient.DefaultResponse = "Summary for testing."

	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)

	summaries, _, _ := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	grouped := groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	mdPath := filepath.Join(tmpDir, MarkdownFileName)
	content, err := os.ReadFile(mdPath)
	assert.NoError(t, err)
	lines := strings.Split(string(content), "\n")

	// Verify header is present
	assert.Contains(t, lines[0], "# YAML File Details")

	// Verify directories are sorted alphabetically
	alphaIdx := -1
	betaIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "## [alpha/]") {
			alphaIdx = i
		}
		if strings.Contains(line, "## [beta/]") {
			betaIdx = i
		}
	}
	assert.Greater(t, alphaIdx, 0, "Should find alpha directory")
	assert.Greater(t, betaIdx, 0, "Should find beta directory")
	assert.Less(t, alphaIdx, betaIdx, "Directories should be sorted alphabetically")

	// Verify files within directory are sorted
	aFileIdx := -1
	bFileIdx := -1
	for i := alphaIdx; i < len(lines) && i < betaIdx; i++ {
		if strings.Contains(lines[i], "[a.yaml]") {
			aFileIdx = i
		}
		if strings.Contains(lines[i], "[b.yaml]") {
			bFileIdx = i
		}
	}
	assert.Greater(t, aFileIdx, alphaIdx, "Should find a.yaml")
	assert.Greater(t, bFileIdx, alphaIdx, "Should find b.yaml")
	assert.Less(t, aFileIdx, bFileIdx, "Files should be sorted alphabetically")
}

// TestIntegrationSummaryCleaning tests that summaries are properly cleaned and truncated.
func TestIntegrationSummaryCleaning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "integration_test_cleaning_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	testFile := filepath.Join(tmpDir, "test.yaml")
	assert.NoError(t, os.WriteFile(testFile, []byte("test: data"), 0644))

	mockClient := NewMockOllamaClient()
	// Simulate a response with markdown, lists, and extra sentences
	mockClient.DefaultResponse = `# Summary
- This is a bullet point
This is sentence one. This is sentence two. This is sentence three. This is sentence four.
* Another list item
**Bold text here**`

	summary, err := summarizeYAMLFile(context.Background(), mockClient, testFile)
	assert.NoError(t, err)

	// Verify cleaning and truncation
	assert.NotContains(t, summary, "#", "Should not contain markdown headers")
	assert.NotContains(t, summary, "**", "Should not contain bold markdown")
	assert.NotContains(t, summary, "-", "Should not contain list markers")
	assert.NotContains(t, summary, "*", "Should not contain asterisk list markers")

	// Count sentences (should be truncated to 2)
	sentenceCount := strings.Count(summary, ".")
	assert.LessOrEqual(t, sentenceCount, 2, "Should have at most 2 sentences")
}

// TestIntegrationEmptyDirectory tests handling of directories with no YAML files.
func TestIntegrationEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "integration_test_empty_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	mockClient := NewMockOllamaClient()

	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 0, "Should find 0 YAML files in empty directory")

	_, processed, skipped := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	assert.Equal(t, 0, processed)
	assert.Equal(t, 0, skipped)
}

// TestIntegrationLocalCache tests the --localcache flag end-to-end.
func TestIntegrationLocalCache(t *testing.T) {
	// Save and restore working directory and localCache flag
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	origLocalCache := localCache
	defer func() {
		_ = os.Chdir(origDir)
		localCache = origLocalCache
	}()

	// Create a temp directory to act as both working directory and project root
	tmpDir, err := os.MkdirTemp("", "integration_test_localcache_*")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	assert.NoError(t, os.Chdir(tmpDir))

	// Create test directory structure with YAML files
	subDir := filepath.Join(tmpDir, "manifests")
	assert.NoError(t, os.MkdirAll(subDir, 0755))

	testFiles := map[string]string{
		filepath.Join(tmpDir, "root.yaml"):        "apiVersion: v1\nkind: ConfigMap",
		filepath.Join(subDir, "deployment.yaml"):   "apiVersion: apps/v1\nkind: Deployment",
	}

	for filePath, content := range testFiles {
		assert.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	mockClient := NewMockOllamaClient()
	mockClient.MockResponses = map[string]string{
		"kind: ConfigMap":  "This is a ConfigMap for app config.",
		"kind: Deployment": "This is a Deployment for the web app.",
	}

	// Enable localcache flag
	localCache = true

	// Find and process YAML files
	yamlFiles, err := findYAMLFiles(tmpDir, false)
	assert.NoError(t, err)
	assert.Len(t, yamlFiles, 2)

	summaries, processed, skipped := processYAMLFiles(yamlFiles, tmpDir, make(map[string]string), mockClient, false)
	assert.Equal(t, 2, processed)
	assert.Equal(t, 0, skipped)
	assert.Len(t, summaries, 2)

	// Verify cache directory was created
	cacheDir := filepath.Join(tmpDir, DefaultCacheDirName)
	info, err := os.Stat(cacheDir)
	assert.NoError(t, err, "cache directory should exist")
	assert.True(t, info.IsDir(), "cache path should be a directory")

	// Verify individual cache files exist
	entries, err := os.ReadDir(cacheDir)
	assert.NoError(t, err)
	assert.Len(t, entries, 2, "should have 2 cache files")

	// Verify cache file names use underscore separator for subdirectory paths
	cacheFileNames := make([]string, len(entries))
	for i, e := range entries {
		cacheFileNames[i] = e.Name()
	}
	assert.Contains(t, cacheFileNames, "root.yaml.md")
	assert.Contains(t, cacheFileNames, "manifests_deployment.yaml.md")

	// Verify cache file contents
	rootCache, err := os.ReadFile(filepath.Join(cacheDir, "root.yaml.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(rootCache), "ConfigMap for app config")

	deployCache, err := os.ReadFile(filepath.Join(cacheDir, "manifests_deployment.yaml.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(deployCache), "Deployment for the web app")

	// Write markdown and verify it also works alongside cache
	grouped := groupSummariesByDir(yamlFiles, summaries, tmpDir)
	assert.NoError(t, writeMarkdownSummary(tmpDir, grouped))

	mdPath := filepath.Join(tmpDir, MarkdownFileName)
	mdContent, err := os.ReadFile(mdPath)
	assert.NoError(t, err)
	assert.Contains(t, string(mdContent), "# YAML File Details")
}

// TestIntegrationModelAvailability tests the model availability check.
func TestIntegrationModelAvailability(t *testing.T) {
	mockClient := NewMockOllamaClient()
	mockClient.AvailableModels = []string{DefaultModelName}

	// Test with available model
	response, err := mockClient.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, response.Models, 1)
	assert.Equal(t, DefaultModelName, response.Models[0].Name)

	// Test with unavailable model
	mockClient.AvailableModels = []string{"other-model"}
	response, err = mockClient.List(context.Background())
	assert.NoError(t, err)
	modelAvailable := false
	for _, model := range response.Models {
		if model.Name == DefaultModelName {
			modelAvailable = true
			break
		}
	}
	assert.False(t, modelAvailable, "Default model should not be available")
}
