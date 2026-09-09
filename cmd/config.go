package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// fileConfig mirrors the subset of CLI flags that can be set via .readmebuilder.yaml.
type fileConfig struct {
	Provider    string `yaml:"provider"`
	Model       string `yaml:"model"`
	Format      string `yaml:"format"`
	Output      string `yaml:"output"`
	CacheDir    string `yaml:"cache-dir"`
	Concurrency int    `yaml:"concurrency"`
	Regenerate  bool   `yaml:"regenerate"`
	LocalCache  bool   `yaml:"localcache"`
	Verbose     bool   `yaml:"verbose"`
	DryRun      bool   `yaml:"dry-run"`
}

// applyFileConfig reads the config file and applies values for any CLI flags that were not
// explicitly set by the user. Explicit CLI flags always take precedence.
func applyFileConfig(cmd *cobra.Command, cfgPath string) error {
	paths := []string{cfgPath}
	if cfgPath == "" {
		paths = []string{".readmebuilder.yaml", ".readmebuilder.yml"}
	}

	var data []byte
	var chosenPath string
	for _, p := range paths {
		var err error
		data, err = os.ReadFile(p)
		if err == nil {
			chosenPath = p
			break
		}
	}
	if chosenPath == "" {
		return nil // no config file present is fine
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("error parsing config file %s: %w", chosenPath, err)
	}

	flags := cmd.Flags()

	if !flags.Changed("provider") && cfg.Provider != "" {
		provider = cfg.Provider
	}
	if !flags.Changed("model") && cfg.Model != "" {
		ModelName = cfg.Model
	}
	if !flags.Changed("format") && cfg.Format != "" {
		outputFormat = cfg.Format
	}
	if !flags.Changed("output") && cfg.Output != "" {
		markdownFileName = cfg.Output
	}
	if !flags.Changed("cache-dir") && cfg.CacheDir != "" {
		cacheDirName = cfg.CacheDir
	}
	if !flags.Changed("concurrency") && cfg.Concurrency > 0 {
		concurrency = cfg.Concurrency
	}
	if !flags.Changed("regenerate") && cfg.Regenerate {
		regenerate = cfg.Regenerate
	}
	if !flags.Changed("localcache") && cfg.LocalCache {
		localCache = cfg.LocalCache
	}
	if !flags.Changed("verbose") && cfg.Verbose {
		verbose = cfg.Verbose
	}
	if !flags.Changed("dry-run") && cfg.DryRun {
		dryRun = cfg.DryRun
	}
	return nil
}
