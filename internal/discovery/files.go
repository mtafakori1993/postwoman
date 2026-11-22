package discovery

import (
	"path/filepath"
	"strings"
)

var languageByExt = map[string]string{
	".go": "go", ".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".ts": "typescript", ".tsx": "typescript", ".mts": "typescript", ".cts": "typescript",
	".py": "python", ".rb": "ruby", ".php": "php", ".java": "java", ".kt": "kotlin", ".kts": "kotlin",
	".cs": "csharp", ".fs": "fsharp", ".scala": "scala", ".rs": "rust", ".swift": "swift", ".dart": "dart",
	".ex": "elixir", ".exs": "elixir", ".clj": "clojure", ".vue": "vue", ".svelte": "svelte",
	".astro": "astro", ".graphql": "graphql", ".gql": "graphql", ".proto": "proto", ".prisma": "prisma",
	".json": "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".ini": "ini", ".conf": "config",
	".sql": "sql", ".md": "markdown", ".http": "http", ".rest": "http", ".raml": "yaml", ".env": "env",
	".gradle": "groovy", ".csproj": "xml", ".xml": "xml",
}

var manifestNames = map[string]bool{
	"package.json": true, "tsconfig.json": true, "go.mod": true, "requirements.txt": true,
	"pyproject.toml": true, "pipfile": true, "setup.py": true, "composer.json": true,
	"gemfile": true, "pom.xml": true, "build.gradle": true, "build.gradle.kts": true,
	"settings.gradle": true, "dockerfile": true, "docker-compose.yml": true,
	"docker-compose.yaml": true, "makefile": true, "cargo.toml": true, "mix.exs": true,
	"pubspec.yaml": true, "nuxt.config.ts": true, "next.config.js": true,
	"vite.config.ts": true, "wails.json": true,
}

var lockfileNames = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "composer.lock": true,
	"gemfile.lock": true, "poetry.lock": true, "pipfile.lock": true, "go.sum": true,
	"cargo.lock": true, "packages.lock.json": true, "mix.lock": true, "pubspec.lock": true,
}

// classifyFile decides whether a file can inform API discovery and which
// category it belongs to.
func classifyFile(rel, name string) (language, category string, candidate bool, skipReason string) {
	lower := strings.ToLower(name)
	relLower := strings.ToLower(rel)
	ext := strings.ToLower(filepath.Ext(name))
	language = languageByExt[ext]
	if strings.HasSuffix(relLower, ".csproj") || strings.HasSuffix(relLower, ".fsproj") {
		language = "csharp"
	}
	if lockfileNames[lower] {
		return language, CategoryOther, false, "lockfile"
	}
	if strings.Contains(relLower, ".designer.") || strings.HasSuffix(lower, ".d.ts") || strings.HasSuffix(lower, ".generated.ts") {
		return language, CategoryOther, false, "generated file"
	}
	if manifestNames[lower] || strings.HasPrefix(lower, "docker-compose") || strings.HasPrefix(lower, "appsettings") {
		return languageOr(language, "config"), CategoryConfig, true, ""
	}
	if ext == ".json" {
		if isSchemaName(lower) || isConfigName(lower) || strings.Contains(relLower, "openapi") || strings.Contains(relLower, "swagger") {
			return language, categoryFor(lower, relLower), true, ""
		}
		return language, CategoryOther, false, "irrelevant file type"
	}
	if ext == ".md" {
		if strings.HasPrefix(lower, "readme") || strings.Contains(relLower, "docs/") || strings.Contains(lower, "api") {
			return language, CategoryDoc, true, ""
		}
		return language, CategoryDoc, false, "irrelevant file type"
	}
	if ext == ".env" || strings.Contains(lower, ".env.") {
		if isSafeEnvTemplate(lower) {
			return "env", CategoryConfig, true, ""
		}
		return "env", CategoryConfig, false, "sensitive file (never read)"
	}
	if ext == "" {
		if manifestNames[lower] {
			return "config", CategoryConfig, true, ""
		}
		return "", CategoryOther, false, "irrelevant file type"
	}
	if language == "" {
		if ext == ".txt" && (strings.Contains(lower, "requirements") || strings.Contains(lower, "routes")) {
			return "text", CategoryConfig, true, ""
		}
		return "", CategoryOther, false, "irrelevant file type"
	}
	return language, categoryFor(lower, relLower), true, ""
}

func languageOr(language, fallback string) string {
	if language != "" {
		return language
	}
	return fallback
}

func isSchemaName(lower string) bool {
	return strings.Contains(lower, "openapi") || strings.Contains(lower, "swagger") ||
		strings.Contains(lower, "asyncapi") || strings.Contains(lower, "api-spec") ||
		strings.Contains(lower, "api_spec") || lower == "api.json"
}

func isConfigName(lower string) bool {
	for _, keyword := range []string{"config", "settings", "manifest", "package"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
