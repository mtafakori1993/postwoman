// Package discovery implements the API Discovery Agent: a local, privacy-aware
// project analyser that finds HTTP endpoints by understanding project structure,
// route registrations, handlers, schemas and API documents, optionally using the
// central AI service for reasoning. It never sends secrets to an AI provider.
package discovery

import (
	"regexp"
	"strings"

	"Postwoman/internal/model"
)

// File categories used by the UI filter.
const (
	CategorySource = "source"
	CategoryRoute  = "route"
	CategoryModel  = "model"
	CategoryConfig = "config"
	CategorySchema = "schema"
	CategoryDoc    = "doc"
	CategoryTest   = "test"
	CategoryOther  = "other"
)

// Defaults returns production defaults for a discovery run.
func Defaults() model.DiscoveryOptions {
	return model.DiscoveryOptions{
		CollectionName:   "Discovered API",
		RespectGitignore: true,
		IncludeTests:     false,
		UseAI:            true,
		MaxFiles:         4000,
		MaxFileSizeKB:    512,
		MaxAIFiles:       40,
		ExcludePatterns:  DefaultExcludes(),
	}
}

// DefaultExcludes lists the directories/files the agent never reads. These are
// dependency, build and secret locations rather than user code.
func DefaultExcludes() []string {
	return []string{
		"**/node_modules/**", "**/.git/**", "**/vendor/**", "**/dist/**",
		"**/build/**", "**/out/**", "**/target/**", "**/bin/**", "**/obj/**",
		"**/coverage/**", "**/htmlcov/**", "**/.nyc_output/**", "**/.next/**",
		"**/.nuxt/**", "**/.svelte-kit/**", "**/__pycache__/**", "**/.venv/**",
		"**/venv/**", "**/env/**", "**/.tox/**", "**/.mypy_cache/**",
		"**/.pytest_cache/**", "**/.gradle/**", "**/.idea/**", "**/.vscode/**",
		"**/.terraform/**", "**/tmp/**", "**/logs/**", "**/bower_components/**",
		"**/Pods/**", "**/DerivedData/**", "**/.angular/**", "**/.cache/**",
		"**/*.min.js", "**/*.min.css", "**/*.map", "**/*.lock",
	}
}

// hardSkipDirs are structural directories that are never part of API source.
var hardSkipDirs = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true, "dist": true,
	"build": true, "out": true, "target": true, "bin": true, "obj": true,
	"coverage": true, "htmlcov": true, ".nyc_output": true, ".next": true,
	".nuxt": true, ".svelte-kit": true, "__pycache__": true, ".venv": true,
	"venv": true, ".tox": true, ".mypy_cache": true, ".pytest_cache": true,
	".gradle": true, ".terraform": true, "bower_components": true,
	"Pods": true, "DerivedData": true, ".angular": true, ".cache": true,
	".parcel-cache": true, ".turbo": true, ".serverless": true,
}

type fileEntry struct {
	Rel      string
	Abs      string
	Language string
	Category string
	Content  string
	Lines    []string
	Size     int64
	Info     model.DiscoveryFileInfo
}

type scanResult struct {
	Root        string
	ProjectName string
	Files       []*fileEntry
	Infos       []model.DiscoveryFileInfo
	Ignore      *ignoreSet
	Stats       model.DiscoveryStats
	Warnings    []string
}

// scanProgress is a callback invoked while scanning.
type scanProgress func(percent int, message, current string)

func categoryRank(category string) int {
	switch category {
	case CategoryRoute:
		return 0
	case CategorySchema:
		return 1
	case CategorySource:
		return 2
	case CategoryModel:
		return 3
	case CategoryConfig:
		return 4
	case CategoryDoc:
		return 5
	default:
		return 6
	}
}

func compileExcludes(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	seen := map[string]bool{}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		dirOnly := strings.HasSuffix(pattern, "/")
		pattern = strings.TrimSuffix(pattern, "/")
		anchored := strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "/")
		pattern = strings.TrimPrefix(pattern, "/")
		expr := globToRegexp(pattern)
		prefix := `^(?:.*/)?`
		if anchored {
			prefix = `^`
		}
		suffix := `(?:$|/.*$)`
		if dirOnly {
			suffix = `(?:/.*)?$`
		}
		if re, err := regexp.Compile(prefix + expr + suffix); err == nil {
			out = append(out, re)
		}
	}
	return out
}

func matchesExcludes(patterns []*regexp.Regexp, rel string) bool {
	for _, re := range patterns {
		if re.MatchString(rel) {
			return true
		}
	}
	return false
}
