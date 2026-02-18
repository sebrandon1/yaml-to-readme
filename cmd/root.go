package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ollama "github.com/ollama/ollama/api"
	"github.com/spf13/cobra"
)

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

// findYAMLFiles recursively finds all YAML files under the given directory path.
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

		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
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

// summarizeYAMLFile uses Ollama to generate a short summary for a YAML file.
func summarizeYAMLFile(ctx context.Context, client OllamaClient, file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file, err)
	}

	falseVar := false
	chatReq := &ollama.ChatRequest{
		Model: ModelName,
		Messages: []ollama.Message{
			{
				Role:    "user",
				Content: SummarizePrompt + string(content),
			},
		},
		Options: map[string]interface{}{
			"seed": 42, // Seed for reproducibility
		},
		Stream: &falseVar, // Disable streaming for simplicity
	}

	var summary string
	err = client.Chat(ctx, chatReq, func(resp ollama.ChatResponse) error {
		summary += resp.Message.Content
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("ollama chat error for %s: %w", file, err)
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

	dirs := make([]string, 0, len(grouped))
	for dir := range grouped {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		files := grouped[dir]
		// Sort files alphabetically by filename
		sortedFiles := make([][2]string, len(files))
		copy(sortedFiles, files)
		sort.Slice(sortedFiles, func(i, j int) bool {
			return sortedFiles[i][0] < sortedFiles[j][0]
		})
		if _, err := fmt.Fprintf(f, "\n## [%s/](../%s/)\n", dir, dir); err != nil {
			return err
		}
		for _, entry := range sortedFiles {
			if _, err := fmt.Fprintf(f, "- [%s](../%s/%s): %s\n", entry[0], dir, entry[0], entry[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

// progressBar displays a simple progress bar in the terminal.
func progressBar(current, total int) {
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

// writeIndividualSummary writes the summary for a single YAML file to a hidden cache directory in the repo root (where the binary is called from).
func writeIndividualSummary(baseDir, filePath, summary string) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(repoRoot, cacheDirName)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		// If not under baseDir, fallback to base name only
		relPath = filepath.Base(filePath)
	}
	cacheFile := strings.ReplaceAll(relPath, string(os.PathSeparator), "_")
	cacheFilePath := filepath.Join(cacheDir, cacheFile+".md")
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
func processYAMLFiles(yamlFiles []string, dir string, existingSummaries map[string]string, client OllamaClient, forceRegenerate bool) (map[string]string, int, int) {
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
				summaries[file] = summary
				skipped++
				continue
			}
		}
		toProcess = append(toProcess, file)
	}

	// Determine effective concurrency
	workers := concurrency
	if workers < 1 {
		workers = 1
	}
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

			summary, err := summarizeYAMLFile(context.Background(), client, f)
			if err != nil {
				fmt.Println(err)
				completed.Add(1)
				progressBar(int(completed.Load()), total)
				return
			}

			mu.Lock()
			summaries[f] = summary
			mu.Unlock()

			if localCache {
				_ = writeIndividualSummary(dir, f, summary)
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

// runSummarizeYaml is the main logic for the summarize-yaml command.
func runSummarizeYaml(dir string) error {
	if dryRun {
		return runDryRun(dir)
	}

	realClient, err := NewRealOllamaClient()
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}
	return runSummarizeYamlWithClient(dir, OllamaClient(realClient))
}

// runSummarizeYamlWithClient contains the core summarization logic, accepting an OllamaClient for testability.
func runSummarizeYamlWithClient(dir string, client OllamaClient) error {
	yamlFiles, err := findYAMLFiles(dir, includeHidden)
	if err != nil {
		return err
	}
	mdPath := filepath.Join(dir, markdownFileName)
	existingSummaries := parseExistingSummaries(mdPath)

	// Check if the model is available
	response, err := client.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	modelAvailable := false
	for _, model := range response.Models {
		if model.Name == ModelName {
			modelAvailable = true
			break
		}
	}
	if !modelAvailable {
		return fmt.Errorf("model %s is not available. Please ensure it is downloaded and available in Ollama", ModelName)
	}

	start := time.Now()
	summaries, processed, skipped := processYAMLFiles(yamlFiles, dir, existingSummaries, client, regenerate)
	elapsed := time.Since(start)
	grouped := groupSummariesByDir(yamlFiles, summaries, dir)
	if err := writeMarkdownSummary(dir, grouped); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}
	fmt.Printf("\nMarkdown summary written to %s\n", mdPath)
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

func init() {
	rootCmd.Flags().BoolVar(&regenerate, "regenerate", false, "Regenerate all summaries, even if they already exist in yaml_details.md")
	rootCmd.Flags().BoolVar(&localCache, "localcache", false, "Write individual summaries to .yaml_summary_cache in the repo root. Mostly used for debugging or local development.")
	rootCmd.Flags().BoolVar(&includeHidden, "include-hidden-directories", false, "Include hidden directories (starting with '.') when searching for YAML files")
	rootCmd.Flags().StringVar(&ModelName, "model", DefaultModelName, "Ollama model to use (default: "+DefaultModelName+")")
	rootCmd.Flags().StringVarP(&markdownFileName, "output", "o", DefaultMarkdownFileName, "Output markdown filename (default: "+DefaultMarkdownFileName+")")
	rootCmd.Flags().StringVar(&cacheDirName, "cache-dir", DefaultCacheDirName, "Cache directory name for --localcache (default: "+DefaultCacheDirName+")")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview which YAML files would be processed without calling the LLM")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "j", 1, "Number of concurrent workers for processing YAML files")
}

// Execute runs the root Cobra command.
func Execute() error {
	return rootCmd.Execute()
}
