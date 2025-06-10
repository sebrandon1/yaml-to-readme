package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ollama "github.com/ollama/ollama/api"
	"github.com/spf13/cobra"
)

const (
	ModelName = "llama3.2:latest"
)

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

func summarizeYAMLFile(ctx context.Context, client *ollama.Client, file string) (string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", file, err)
	}
	req := &ollama.GenerateRequest{
		Model:  ModelName,
		Prompt: "In one or two sentences, summarize what this YAML file does:\n" + string(content),
	}
	var summary string
	err = client.Generate(ctx, req, func(resp ollama.GenerateResponse) error {
		summary += resp.Response
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("Ollama error for %s: %w", file, err)
	}
	return summary, nil
}

func groupSummariesByDir(yamlFiles []string, summaries map[string]string, baseDir string) map[string][][2]string {
	grouped := make(map[string][][2]string)
	for _, file := range yamlFiles {
		relPath, _ := filepath.Rel(baseDir, file)
		dir := filepath.Dir(relPath)
		grouped[dir] = append(grouped[dir], [2]string{filepath.Base(file), summaries[file]})
	}
	return grouped
}

func writeMarkdownSummary(baseDir string, grouped map[string][][2]string) error {
	mdPath := filepath.Join(baseDir, "yaml_details.md")
	f, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer f.Close()

	head := `# YAML File Details

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
	if _, err := f.WriteString(head); err != nil {
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

var rootCmd = &cobra.Command{
	Use:   "summarize-yaml [directory]",
	Short: "Summarize YAML files in a directory using Ollama",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		yamlFiles, err := findYAMLFiles(dir)
		if err != nil {
			return err
		}
		client, err := ollama.ClientFromEnvironment()
		if err != nil {
			return fmt.Errorf("failed to create Ollama client: %w", err)
		}
		ctx := context.Background()
		summaries := make(map[string]string)
		for _, file := range yamlFiles {
			summary, err := summarizeYAMLFile(ctx, client, file)
			if err != nil {
				fmt.Println(err)
				continue
			}
			summaries[file] = summary
			fmt.Printf("Summary for %s:\n%s\n\n", file, summary)
		}
		grouped := groupSummariesByDir(yamlFiles, summaries, dir)
		if err := writeMarkdownSummary(dir, grouped); err != nil {
			return fmt.Errorf("failed to write markdown: %w", err)
		}
		fmt.Printf("\nMarkdown summary written to %s\n", filepath.Join(dir, "yaml_details.md"))
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
