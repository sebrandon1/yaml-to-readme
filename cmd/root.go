package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ollama "github.com/ollama/ollama/api"
	"github.com/spf13/cobra"
)

// Keeping a collection of models for future use.
const (
	ModelName           = "llama3.2:latest"
	DefaultCacheDirName = ".yaml_summary_cache"
	MarkdownFileName    = "yaml_details.md"
	MarkdownHeader      = `# YAML File Details

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

// findYAMLFiles recursively finds all YAML files under the given directory path.
func findYAMLFiles(dir string) ([]string, error) {
	var yamlFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			yamlFiles = append(yamlFiles, path)
		}
		return nil
	})
	return yamlFiles, err
}

// summarizeYAMLFile uses Ollama to generate a short summary for a YAML file.
func summarizeYAMLFile(ctx context.Context, client *ollama.Client, file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file, err)
	}
	// Use the stricter prompt from const
	req := &ollama.GenerateRequest{
		Model:  ModelName,
		Prompt: SummarizePrompt + string(content),
	}
	var summary string
	err = client.Generate(ctx, req, func(resp ollama.GenerateResponse) error {
		summary += resp.Response
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Ollama error for %s: %w", file, err)
	}
	// Post-process: Truncate to the first two sentences (ending with a period, exclamation, or question mark)
	trimmed := truncateToSentences(summary, 2)
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
	mdPath := filepath.Join(baseDir, MarkdownFileName)
	f, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(MarkdownHeader); err != nil {
		return err
	}

	for dir, files := range grouped {
		fmt.Fprintf(f, "\n## [%s/](../%s/)\n", dir, dir)
		for _, entry := range files {
			fmt.Fprintf(f, "- [%s](../%s/%s): %s\n", entry[0], dir, entry[0], entry[1])
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
	f, err := os.Open(mdPath)
	if err != nil {
		return existing
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentDir string
	for scanner.Scan() {
		line := scanner.Text()
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
	return existing
}

// writeIndividualSummary writes the summary for a single YAML file to a hidden cache directory in the repo root (where the binary is called from).
func writeIndividualSummary(baseDir, filePath, summary string) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(repoRoot, DefaultCacheDirName)
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
	defer f.Close()
	_, err = f.WriteString(summary)
	return err
}

// processYAMLFiles processes YAML files, generating summaries if needed, and returns the summaries map and counters.
func processYAMLFiles(yamlFiles []string, dir string, existingSummaries map[string]string, client *ollama.Client, forceRegenerate bool) (map[string]string, int, int) {
	summaries := make(map[string]string)
	skipped := 0
	processed := 0
	total := len(yamlFiles)
	for i, file := range yamlFiles {
		rel, _ := filepath.Rel(dir, file)
		rel = filepath.ToSlash(rel)
		if !forceRegenerate {
			if summary, ok := existingSummaries[rel]; ok && summary != "" {
				summaries[file] = summary
				skipped++
				progressBar(i+1, total)
				continue
			}
		}
		progressBar(i+1, total)
		summary, err := summarizeYAMLFile(context.Background(), client, file)
		if err != nil {
			fmt.Println(err)
			continue
		}
		summaries[file] = summary
		if localCache {
			_ = writeIndividualSummary(dir, file, summary) // Write to cache if flag is set
		}
		processed++
	}
	return summaries, processed, skipped
}

// runSummarizeYaml is the main logic for the summarize-yaml command.
func runSummarizeYaml(dir string) error {
	yamlFiles, err := findYAMLFiles(dir)
	if err != nil {
		return err
	}
	mdPath := filepath.Join(dir, MarkdownFileName)
	existingSummaries := parseExistingSummaries(mdPath)
	client, err := ollama.ClientFromEnvironment()
	if err != nil {
		return fmt.Errorf("failed to create Ollama client: %w", err)
	}

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

func init() {
	rootCmd.Flags().BoolVar(&regenerate, "regenerate", false, "Regenerate all summaries, even if they already exist in yaml_details.md")
	rootCmd.Flags().BoolVar(&localCache, "localcache", false, "Write individual summaries to .yaml_summary_cache in the repo root. Mostly used for debugging or local development.")
}

// Execute runs the root Cobra command.
func Execute() error {
	return rootCmd.Execute()
}
