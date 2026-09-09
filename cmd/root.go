package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// setupLogging configures the slog default logger based on the verbose flag.
func setupLogging() {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// Keeping a collection of constants for future use.
const (
	DefaultModelName        = "llama3.2:latest"
	DefaultCacheDirName     = ".yaml_summary_cache"
	DefaultMarkdownFileName = "yaml_details.md"
	MarkdownHeader          = `# YAML File Details

This document provides an overview of all YAML files in the repository, organized by directory, with a brief description of what each file does or configures. Use this as a reference for understanding the purpose of each manifest or configuration file.

---

## How to Use
- Click the file links to jump to the file in the repository.
- Each entry includes a short summary of the file's intent or function.

---

<!--
  To keep this file up to date, add new YAMLs as they are introduced and provide a short description for each.
-->

`
	SummarizePrompt = "Summarize the purpose of this YAML file in no more than two short, high-level sentences. Do not include any lists, breakdowns, explanations, advice, notes, or formatting. Do not use markdown. No newlines. No code sections. Only output a single, concise summary of the file's purpose, and nothing else. Stop after two sentences. If you cannot summarize in two sentences, summarize in one: \n"
)

// ModelName is configurable via the --model flag and defaults to DefaultModelName.
var ModelName string = DefaultModelName

// markdownFileName is configurable via the --output flag and defaults to DefaultMarkdownFileName.
var markdownFileName string = DefaultMarkdownFileName

// cacheDirName is configurable via the --cache-dir flag and defaults to DefaultCacheDirName.
var cacheDirName string = DefaultCacheDirName

// matchGlob reports whether relPath matches the glob pattern.
// It supports ** as a wildcard for any directory depth.
func matchGlob(pattern, relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	pattern = filepath.ToSlash(pattern)
	if idx := strings.Index(pattern, "**"); idx >= 0 {
		return strings.HasPrefix(relPath, pattern[:idx])
	}
	if matched, _ := filepath.Match(pattern, relPath); matched {
		return true
	}
	matched, _ := filepath.Match(pattern, filepath.Base(relPath))
	return matched
}

// findYAMLFiles recursively finds all YAML files under the given directory path.
// Files are filtered by includeGlobs (if non-empty) and excludeGlobs.
func findYAMLFiles(dir string, includeHidden bool) ([]string, error) {
	var yamlFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories unless includeHidden is true
		if info.IsDir() && !includeHidden && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if info.IsDir() || (!strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml")) {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)

		if len(includeGlobs) > 0 {
			included := false
			for _, pattern := range includeGlobs {
				if matchGlob(pattern, relPath) {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		for _, pattern := range excludeGlobs {
			if matchGlob(pattern, relPath) {
				slog.Debug("excluding file", "file", relPath, "pattern", pattern)
				return nil
			}
		}

		yamlFiles = append(yamlFiles, path)
		return nil
	})
	slog.Debug("found YAML files", "count", len(yamlFiles), "dir", dir, "includeHidden", includeHidden)
	return yamlFiles, err
}

// cleanSummary removes markdown, lists, and breakdowns from the summary, keeping only a concise, plain-text summary.
func cleanSummary(summary string) string {
	lines := strings.Split(summary, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "**") ||
			strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "Here's a breakdown") ||
			strings.HasPrefix(trimmed, "The following") ||
			strings.HasPrefix(trimmed, "* ") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return strings.Join(cleaned, " ")
}

// summarizeYAMLFile uses an LLM provider to generate a short summary for a YAML file.
func summarizeYAMLFile(ctx context.Context, provider LLMProvider, file string) (string, error) {
	slog.Debug("summarizing file", "file", file, "model", ModelName, "provider", provider.Name())
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file, err)
	}

	var parsed interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		slog.Warn("skipping file with invalid YAML syntax", "file", file, "error", err)
		return "", fmt.Errorf("invalid YAML syntax in %s: %w", file, err)
	}

	summary, err := provider.Summarize(ctx, string(content), SummarizePrompt)
	if err != nil {
		return "", fmt.Errorf("%s error for %s: %w", provider.Name(), file, err)
	}

	// Clean and truncate summary
	cleaned := cleanSummary(summary)
	trimmed := truncateToSentences(cleaned, 2)
	return trimmed, nil
}

// truncateToSentences returns the first n sentences from the input string.
func truncateToSentences(text string, n int) string {
	count := 0
	end := 0
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
			end = i + 1
			if count == n {
				break
			}
		}
	}
	if end > 0 {
		return strings.TrimSpace(text[:end])
	}
	return strings.TrimSpace(text)
}

// groupSummariesByDir organizes file summaries by their relative directory.
func groupSummariesByDir(yamlFiles []string, summaries map[string]string, baseDir string) map[string][][2]string {
	grouped := make(map[string][][2]string)
	for _, file := range yamlFiles {
		relPath, _ := filepath.Rel(baseDir, file)
		dir := filepath.Dir(relPath)
		grouped[dir] = append(grouped[dir], [2]string{filepath.Base(file), summaries[file]})
	}
	return grouped
}

// sortedDirs returns the directory keys from grouped in sorted order,
// and sorts the file entries within each directory alphabetically by filename.
func sortedDirs(grouped map[string][][2]string) ([]string, map[string][][2]string) {
	dirs := make([]string, 0, len(grouped))
	for dir := range grouped {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	sorted := make(map[string][][2]string, len(grouped))
	for _, dir := range dirs {
		files := make([][2]string, len(grouped[dir]))
		copy(files, grouped[dir])
		sort.Slice(files, func(i, j int) bool {
			return files[i][0] < files[j][0]
		})
		sorted[dir] = files
	}
	return dirs, sorted
}

// writeMarkdownSummary writes the grouped summaries to a markdown file in the base directory.
func writeMarkdownSummary(baseDir string, grouped map[string][][2]string) error {
	mdPath := filepath.Join(baseDir, markdownFileName)
	f, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", cerr)
		}
	}()

	if _, err := f.WriteString(MarkdownHeader); err != nil {
		return err
	}

	dirs, sorted := sortedDirs(grouped)

	for _, dir := range dirs {
		if _, err := fmt.Fprintf(f, "\n## [%s/](../%s/)\n", dir, dir); err != nil {
			return err
		}
		for _, entry := range sorted[dir] {
			if _, err := fmt.Fprintf(f, "- [%s](../%s/%s): %s\n", entry[0], dir, entry[0], entry[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

// JSONOutput represents the structured JSON output format.
type JSONOutput struct {
	BaseDirectory string                     `json:"base_directory"`
	GeneratedAt   string                     `json:"generated_at"`
	Model         string                     `json:"model"`
	Directories   map[string][]JSONFileEntry `json:"directories"`
}

// JSONFileEntry represents a single file entry in the JSON output.
type JSONFileEntry struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

// writeJSONSummary writes the grouped summaries as a JSON file.
func writeJSONSummary(baseDir string, grouped map[string][][2]string) error {
	output := JSONOutput{
		BaseDirectory: baseDir,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Model:         ModelName,
		Directories:   make(map[string][]JSONFileEntry),
	}

	dirs, sorted := sortedDirs(grouped)

	for _, dir := range dirs {
		dirKey := dir + "/"
		for _, entry := range sorted[dir] {
			output.Directories[dirKey] = append(output.Directories[dirKey], JSONFileEntry{
				File:    entry[0],
				Path:    filepath.Join(dir, entry[0]),
				Summary: entry[1],
			})
		}
	}

	outPath := filepath.Join(baseDir, markdownFileName)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return os.WriteFile(outPath, data, 0o644)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>YAML File Details</title>
<style>
body { font-family: system-ui, sans-serif; max-width: 900px; margin: 2rem auto; padding: 0 1rem; color: #333; }
h1 { border-bottom: 2px solid #eee; padding-bottom: 0.5rem; }
h2 { color: #555; margin-top: 2rem; }
ul { list-style: none; padding-left: 0; }
li { padding: 0.4rem 0; border-bottom: 1px solid #f0f0f0; }
a { color: #0366d6; text-decoration: none; }
a:hover { text-decoration: underline; }
.summary { color: #666; }
.meta { color: #999; font-size: 0.85rem; margin-top: 2rem; }
</style>
</head>
<body>
<h1>YAML File Details</h1>
<p>Overview of all YAML files, organized by directory.</p>
{{range .Dirs}}<h2><a href="../{{.Name}}/">{{.Name}}/</a></h2>
<ul>
{{range .Files}}<li><a href="../{{.Path}}">{{.File}}</a>: <span class="summary">{{.Summary}}</span></li>
{{end}}</ul>
{{end}}<p class="meta">Generated at {{.GeneratedAt}} using model {{.Model}}</p>
</body>
</html>
`

type htmlData struct {
	Dirs        []htmlDir
	GeneratedAt string
	Model       string
}

type htmlDir struct {
	Name  string
	Files []JSONFileEntry
}

// writeHTMLSummary writes the grouped summaries as an HTML file.
func writeHTMLSummary(baseDir string, grouped map[string][][2]string) error {
	tmpl, err := template.New("html").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	dirs, sorted := sortedDirs(grouped)

	var data htmlData
	data.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	data.Model = ModelName

	for _, dir := range dirs {
		hd := htmlDir{Name: dir}
		for _, entry := range sorted[dir] {
			hd.Files = append(hd.Files, JSONFileEntry{
				File:    entry[0],
				Path:    filepath.Join(dir, entry[0]),
				Summary: entry[1],
			})
		}
		data.Dirs = append(data.Dirs, hd)
	}

	outPath := filepath.Join(baseDir, markdownFileName)
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", cerr)
		}
	}()

	return tmpl.Execute(f, data)
}

// writeGitHubSummary appends markdown to the $GITHUB_STEP_SUMMARY file.
func writeGitHubSummary(grouped map[string][][2]string) error {
	summaryPath := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryPath == "" {
		return fmt.Errorf("GITHUB_STEP_SUMMARY environment variable is not set; github-summary format requires a GitHub Actions environment")
	}

	f, err := os.OpenFile(summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_STEP_SUMMARY: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing GITHUB_STEP_SUMMARY: %v\n", cerr)
		}
	}()

	if _, err := fmt.Fprintf(f, "## YAML File Summary\n\n"); err != nil {
		return err
	}

	dirs, sorted := sortedDirs(grouped)
	for _, dir := range dirs {
		if _, err := fmt.Fprintf(f, "### %s/\n\n", dir); err != nil {
			return err
		}
		for _, entry := range sorted[dir] {
			if _, err := fmt.Fprintf(f, "- **%s**: %s\n", entry[0], entry[1]); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(f); err != nil {
			return err
		}
	}
	return nil
}

// writeSummary dispatches to the appropriate writer based on the outputFormat flag.
func writeSummary(baseDir string, grouped map[string][][2]string) error {
	switch outputFormat {
	case "json":
		return writeJSONSummary(baseDir, grouped)
	case "html":
		return writeHTMLSummary(baseDir, grouped)
	case "github-summary":
		return writeGitHubSummary(grouped)
	default:
		return writeMarkdownSummary(baseDir, grouped)
	}
}

var progressMu sync.Mutex

// progressBar displays a simple progress bar in the terminal.
// It is safe to call from multiple goroutines.
func progressBar(current, total int) {
	progressMu.Lock()
	defer progressMu.Unlock()
	percent := float64(current) / float64(total) * 100
	barLen := 40
	filledLen := int(float64(barLen) * float64(current) / float64(total))
	bar := strings.Repeat("=", filledLen) + strings.Repeat(" ", barLen-filledLen)
	fmt.Printf("\rProcessing YAML files: [%s] %3.0f%% (%d/%d)", bar, percent, current, total)
	if current == total {
		fmt.Println()
	}
}

// parseExistingSummaries parses an existing yaml_details.md and returns a map of file path to summary.
func parseExistingSummaries(mdPath string) map[string]string {
	existing := make(map[string]string)
	parseSummaryLines(readLinesFromFile(mdPath), existing)
	return existing
}

// readLinesFromFile opens a file and returns its lines as a slice of strings.
func readLinesFromFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// parseSummaryLines processes lines from the markdown and fills the map with file path to summary.
func parseSummaryLines(lines []string, existing map[string]string) {
	var currentDir string
	for _, line := range lines {
		if strings.HasPrefix(line, "## [") && strings.Contains(line, "](") {
			// Extract directory from section header
			start := strings.Index(line, "[") + 1
			end := strings.Index(line, "]")
			if start > 0 && end > start {
				currentDir = line[start:end]
				currentDir = strings.TrimSuffix(currentDir, "/")
			}
		} else if strings.HasPrefix(line, "- [") && strings.Contains(line, "](") {
			// Extract file and summary
			start := strings.Index(line, "[") + 1
			end := strings.Index(line, "]")
			if start > 0 && end > start {
				file := line[start:end]
				if currentDir != "" {
					key := filepath.Join(currentDir, file)
					colon := strings.Index(line, ": ")
					if colon > 0 {
						summary := strings.TrimSpace(line[colon+2:])
						existing[key] = summary
					}
				}
			}
		}
	}
}

// cacheKey returns the cache filename for a YAML file relative path.
func cacheKey(relPath string) string {
	return strings.ReplaceAll(relPath, string(os.PathSeparator), "_") + ".md"
}

// loadLocalCacheSummaries reads per-file summaries from the cache directory for any YAML files
// that have a matching cache entry. Used to resume interrupted runs when --localcache is set.
func loadLocalCacheSummaries(repoRoot, baseDir string, yamlFiles []string) map[string]string {
	cacheDir := filepath.Join(repoRoot, cacheDirName)
	cached := make(map[string]string)
	for _, file := range yamlFiles {
		relPath, err := filepath.Rel(baseDir, file)
		if err != nil || strings.HasPrefix(relPath, "..") {
			continue
		}
		cacheFilePath := filepath.Join(cacheDir, cacheKey(relPath))
		data, err := os.ReadFile(cacheFilePath)
		if err != nil || len(data) == 0 {
			continue
		}
		cached[filepath.ToSlash(relPath)] = string(data)
	}
	return cached
}

// writeIndividualSummary writes the summary for a single YAML file to a hidden cache directory in the given repo root.
func writeIndividualSummary(repoRoot, baseDir, filePath, summary string) error {
	cacheDir := filepath.Join(repoRoot, cacheDirName)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// If not under baseDir, fallback to base name only
		relPath = filepath.Base(filePath)
	}
	cacheFilePath := filepath.Join(cacheDir, cacheKey(relPath))
	f, err := os.Create(cacheFilePath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", cerr)
		}
	}()
	_, err = f.WriteString(summary)
	return err
}

// processYAMLFiles processes YAML files, generating summaries if needed, and returns the summaries map and counters.
func processYAMLFiles(yamlFiles []string, dir string, existingSummaries map[string]string, provider LLMProvider, forceRegenerate bool) (map[string]string, int, int) {
	summaries := make(map[string]string)
	total := len(yamlFiles)
	skipped := 0

	// First pass: identify which files need processing and collect existing summaries
	var toProcess []string
	for _, file := range yamlFiles {
		rel, _ := filepath.Rel(dir, file)
		rel = filepath.ToSlash(rel)
		if !forceRegenerate {
			if summary, ok := existingSummaries[rel]; ok && summary != "" {
				slog.Debug("skipping file with existing summary", "file", rel)
				summaries[file] = summary
				skipped++
				continue
			}
		}
		toProcess = append(toProcess, file)
	}

	// Cache repo root for writeIndividualSummary calls
	var repoRoot string
	if localCache {
		repoRoot, _ = os.Getwd()
	}

	// Determine effective concurrency
	workers := max(concurrency, 1)
	if workers > len(toProcess) && len(toProcess) > 0 {
		workers = len(toProcess)
	}

	// Second pass: process new files concurrently
	var mu sync.Mutex
	var completed atomic.Int64
	completed.Store(int64(skipped))
	var processed atomic.Int64

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, file := range toProcess {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore slot
		go func(f string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore slot

			ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
			defer cancel()
			summary, err := summarizeYAMLFile(ctx, provider, f)
			if err != nil {
				slog.Error("failed to summarize file", "file", f, "error", err)
				completed.Add(1)
				progressBar(int(completed.Load()), total)
				return
			}

			mu.Lock()
			summaries[f] = summary
			mu.Unlock()

			if localCache {
				_ = writeIndividualSummary(repoRoot, dir, f, summary)
			}

			processed.Add(1)
			completed.Add(1)
			progressBar(int(completed.Load()), total)
		}(file)
	}

	wg.Wait()

	return summaries, int(processed.Load()), skipped
}

// runDryRun prints which YAML files would be processed without calling the LLM.
func runDryRun(dir string) error {
	yamlFiles, err := findYAMLFiles(dir, includeHidden)
	if err != nil {
		return err
	}
	mdPath := filepath.Join(dir, markdownFileName)
	existingSummaries := parseExistingSummaries(mdPath)

	newFiles := 0
	existingFiles := 0
	var newList []string
	for _, file := range yamlFiles {
		rel, _ := filepath.Rel(dir, file)
		rel = filepath.ToSlash(rel)
		if summary, ok := existingSummaries[rel]; ok && summary != "" {
			existingFiles++
		} else {
			newFiles++
			newList = append(newList, rel)
		}
	}

	fmt.Printf("Dry run: %d YAML files found in %s\n", len(yamlFiles), dir)
	fmt.Printf("  New (would summarize): %d\n", newFiles)
	fmt.Printf("  Existing (would skip): %d\n", existingFiles)
	if len(newList) > 0 {
		fmt.Println("\nFiles to summarize:")
		for _, f := range newList {
			fmt.Printf("  %s\n", f)
		}
	}
	return nil
}

// createProvider creates an LLMProvider based on the --provider flag.
func createProvider() (LLMProvider, error) {
	switch provider {
	case "openai":
		return NewOpenAIProvider()
	default:
		return NewOllamaProvider()
	}
}

// runSummarizeYaml is the main logic for the summarize-yaml command.
func runSummarizeYaml(dir string) error {
	setupLogging()
	if dryRun {
		return runDryRun(dir)
	}

	llm, err := createProvider()
	if err != nil {
		return fmt.Errorf("failed to create %s provider: %w", provider, err)
	}
	return runSummarizeYamlWithProvider(dir, llm)
}

// runSummarizeYamlWithProvider contains the core summarization logic, accepting an LLMProvider for testability.
func runSummarizeYamlWithProvider(dir string, llm LLMProvider) error {
	slog.Debug("starting summarization", "dir", dir, "model", ModelName, "provider", llm.Name(), "concurrency", concurrency)
	yamlFiles, err := findYAMLFiles(dir, includeHidden)
	if err != nil {
		return err
	}
	mdPath := filepath.Join(dir, markdownFileName)
	existingSummaries := parseExistingSummaries(mdPath)

	if localCache {
		repoRoot, _ := os.Getwd()
		for rel, summary := range loadLocalCacheSummaries(repoRoot, dir, yamlFiles) {
			if _, ok := existingSummaries[rel]; !ok {
				existingSummaries[rel] = summary
			}
		}
	}

	pending := 0
	for _, file := range yamlFiles {
		rel, _ := filepath.Rel(dir, file)
		rel = filepath.ToSlash(rel)
		if existingSummaries[rel] == "" {
			pending++
		}
	}
	fmt.Printf("YAML files found: %d  |  already cached: %d  |  pending: %d\n",
		len(yamlFiles), len(yamlFiles)-pending, pending)

	// Check if the model is available
	modelAvailable, err := llm.Available(context.Background())
	if err != nil {
		return fmt.Errorf("failed to check model availability: %w", err)
	}
	if !modelAvailable {
		return fmt.Errorf("model %s is not available. Please ensure it is downloaded and available in your %s provider", ModelName, llm.Name())
	}

	start := time.Now()
	summaries, processed, skipped := processYAMLFiles(yamlFiles, dir, existingSummaries, llm, regenerate)
	elapsed := time.Since(start)
	grouped := groupSummariesByDir(yamlFiles, summaries, dir)
	if err := writeSummary(dir, grouped); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	if outputFormat == "github-summary" {
		fmt.Printf("\n%s summary written to $GITHUB_STEP_SUMMARY\n", outputFormat)
	} else {
		fmt.Printf("\n%s summary written to %s\n", outputFormat, mdPath)
	}
	fmt.Printf("Files processed (new summaries): %d\n", processed)
	fmt.Printf("Files skipped (already summarized): %d\n", skipped)
	fmt.Printf("Time elapsed: %s\n", elapsed.Round(time.Second))
	return nil
}

// rootCmd is the main Cobra command for the CLI application.
var rootCmd = &cobra.Command{
	Use:   "summarize-yaml [directory]",
	Short: "Summarize YAML files in a directory using Ollama",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSummarizeYaml(args[0])
	},
}

var regenerate bool
var localCache bool
var includeHidden bool
var dryRun bool
var concurrency int
var verbose bool
var outputFormat string
var provider string
var llmTimeout time.Duration
var includeGlobs []string
var excludeGlobs []string

func init() {
	rootCmd.Flags().BoolVar(&regenerate, "regenerate", false, "Regenerate all summaries, even if they already exist in yaml_details.md")
	rootCmd.Flags().BoolVar(&localCache, "localcache", false, "Write individual summaries to .yaml_summary_cache in the repo root. Mostly used for debugging or local development.")
	rootCmd.Flags().BoolVar(&includeHidden, "include-hidden-directories", false, "Include hidden directories (starting with '.') when searching for YAML files")
	rootCmd.Flags().StringVar(&ModelName, "model", DefaultModelName, "Ollama model to use (default: "+DefaultModelName+")")
	rootCmd.Flags().StringVarP(&markdownFileName, "output", "o", DefaultMarkdownFileName, "Output markdown filename (default: "+DefaultMarkdownFileName+")")
	rootCmd.Flags().StringVar(&cacheDirName, "cache-dir", DefaultCacheDirName, "Cache directory name for --localcache (default: "+DefaultCacheDirName+")")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview which YAML files would be processed without calling the LLM")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 1, "Number of concurrent workers for processing YAML files")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
	rootCmd.Flags().StringVar(&outputFormat, "format", "markdown", "Output format: markdown, json, html, or github-summary")
	rootCmd.Flags().StringVar(&provider, "provider", "ollama", "LLM provider: ollama (default) or openai")
	rootCmd.Flags().DurationVar(&llmTimeout, "timeout", 60*time.Second, "Timeout for each LLM request (e.g. 30s, 2m)")
	rootCmd.Flags().StringArrayVar(&includeGlobs, "include", nil, "Glob patterns to include (e.g. 'charts/**'); can be repeated")
	rootCmd.Flags().StringArrayVar(&excludeGlobs, "exclude", nil, "Glob patterns to exclude (e.g. 'testdata/**'); can be repeated")
}

// Execute runs the root Cobra command.
func Execute() error {
	return rootCmd.Execute()
}
