package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"Postwoman/internal/model"
)

// isSecretFile reports whether a file is a credential/secret store that the
// agent must never read or send to an AI provider. Template files such as
// ".env.example" are safe because they contain no real values.
func isSecretFile(name string) bool {
	lower := strings.ToLower(name)
	if isSafeEnvTemplate(lower) {
		return false
	}
	if lower == ".env" || strings.HasPrefix(lower, ".env.") || strings.HasSuffix(lower, ".env") {
		return true
	}
	switch lower {
	case ".npmrc", ".netrc", ".htpasswd", "id_rsa", "id_dsa", "id_ecdsa",
		"id_ed25519", "credentials", "credentials.json":
		return true
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx", ".jks", ".keystore", ".secret", ".secrets", ".kdbx", ".ppk"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	if strings.HasPrefix(lower, "id_rsa") || strings.HasPrefix(lower, "id_ed25519") || strings.HasPrefix(lower, "id_ecdsa") {
		return true
	}
	if strings.Contains(lower, "service-account") && strings.HasSuffix(lower, ".json") {
		return true
	}
	if strings.Contains(lower, "secret") && (strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") ||
		strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".txt")) {
		return true
	}
	return false
}

// isBinary detects NUL bytes in the first block of a file.
func isBinary(content []byte) bool {
	limit := len(content)
	if limit > 8000 {
		limit = 8000
	}
	for i := 0; i < limit; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// isMinified skips bundler output that would only add noise.
func isMinified(content []byte, name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".min.js") || strings.HasSuffix(lower, ".min.css") {
		return true
	}
	if len(content) < 20000 {
		return false
	}
	newlines := strings.Count(string(content), "\n")
	return newlines < len(content)/3000
}

// ListDirectory returns one level of the project browser. Known dependency,
// build and secret locations are flagged so the UI can explain why they are
// excluded without ever reading them.
func ListDirectory(path string, opts model.DiscoveryOptions) (*model.DiscoveryDirListing, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	listing := &model.DiscoveryDirListing{Path: abs, Entries: []model.DiscoveryDirEntry{}}
	if parent := filepath.Dir(abs); parent != abs {
		listing.Parent = parent
	}
	excludes := compileExcludes(opts.ExcludePatterns)
	for _, entry := range entries {
		if len(listing.Entries) >= 500 {
			break
		}
		full := filepath.Join(abs, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		row := model.DiscoveryDirEntry{
			Name: entry.Name(), Path: full, IsDir: entry.IsDir(), Size: info.Size(),
		}
		switch {
		case entry.IsDir() && hardSkipDirs[entry.Name()]:
			row.Excluded, row.Reason = true, "dependency/build directory"
		case !entry.IsDir() && isSecretFile(entry.Name()):
			row.Excluded, row.Reason = true, "sensitive file"
		case matchesExcludes(excludes, entry.Name()):
			row.Excluded, row.Reason = true, "excluded pattern"
		case !opts.IncludeHidden && strings.HasPrefix(entry.Name(), "."):
			row.Excluded, row.Reason = true, "hidden"
		}
		listing.Entries = append(listing.Entries, row)
	}
	sort.SliceStable(listing.Entries, func(i, j int) bool {
		if listing.Entries[i].IsDir != listing.Entries[j].IsDir {
			return listing.Entries[i].IsDir
		}
		return strings.ToLower(listing.Entries[i].Name) < strings.ToLower(listing.Entries[j].Name)
	})
	return listing, nil
}
