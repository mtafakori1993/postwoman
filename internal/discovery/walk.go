package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"Postwoman/internal/model"
)

func scanProject(ctx context.Context, opts model.DiscoveryOptions, progress scanProgress) (*scanResult, error) {
	root, err := filepath.Abs(opts.Directory)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot open project directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = Defaults().MaxFiles
	}
	if opts.MaxFileSizeKB <= 0 {
		opts.MaxFileSizeKB = Defaults().MaxFileSizeKB
	}
	result := &scanResult{Root: root, ProjectName: filepath.Base(root), Ignore: &ignoreSet{}}
	excludes := compileExcludes(opts.ExcludePatterns)
	maxBytes := int64(opts.MaxFileSizeKB) * 1024

	// Root .gitignore is loaded before children are visited.
	if opts.RespectGitignore {
		if content, readErr := os.ReadFile(filepath.Join(root, ".gitignore")); readErr == nil {
			result.Ignore.addAll(string(content), "")
		}
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if hardSkipDirs[name] {
				return filepath.SkipDir
			}
			if !opts.IncludeHidden && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if opts.RespectGitignore && result.Ignore.ignored(rel, true) {
				return filepath.SkipDir
			}
			if matchesExcludes(excludes, rel) {
				return filepath.SkipDir
			}
			if opts.RespectGitignore {
				if content, readErr := os.ReadFile(filepath.Join(path, ".gitignore")); readErr == nil {
					result.Ignore.addAll(string(content), rel)
				}
			}
			return nil
		}
		return visitFile(result, opts, excludes, path, rel, name, d, maxBytes, progress)
	})
	if err != nil && err != context.Canceled {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	result.Infos = buildInfoList(result)
	return result, nil
}

func visitFile(result *scanResult, opts model.DiscoveryOptions, excludes []*regexp.Regexp, path, rel, name string, d fs.DirEntry, maxBytes int64, progress scanProgress) error {
	if len(result.Infos) >= opts.MaxFiles {
		return filepath.SkipAll
	}
	result.Stats.FilesSeen++
	if progress != nil && result.Stats.FilesSeen%200 == 0 {
		progress(5, fmt.Sprintf("Scanning project files (%d seen)", result.Stats.FilesSeen), rel)
	}
	if d.Type()&fs.ModeSymlink != 0 && !opts.FollowSymlinks {
		result.Stats.FilesSkipped++
		return nil
	}
	if opts.RespectGitignore && result.Ignore.ignored(rel, false) {
		result.Stats.FilesSkipped++
		return nil
	}
	if matchesExcludes(excludes, rel) {
		result.Stats.FilesSkipped++
		return nil
	}
	if !opts.IncludeHidden && strings.HasPrefix(name, ".") && !isSafeEnvTemplate(name) {
		result.Stats.FilesSkipped++
		return nil
	}
	if isSecretFile(name) {
		result.Stats.FilesSkipped++
		result.Infos = append(result.Infos, model.DiscoveryFileInfo{
			Path: rel, Category: CategoryConfig, Skipped: "sensitive file (never read)",
		})
		return nil
	}
	if !opts.IncludeTests && isTestPath(rel) {
		result.Stats.FilesSkipped++
		return nil
	}
	language, category, candidate, skipReason := classifyFile(rel, name)
	if !candidate {
		if skipReason != "" {
			result.Stats.FilesSkipped++
			if skipReason != "irrelevant file type" {
				result.Infos = append(result.Infos, model.DiscoveryFileInfo{
					Path: rel, Language: language, Category: category, Skipped: skipReason,
				})
			}
		}
		return nil
	}
	fileInfo, statErr := d.Info()
	if statErr != nil {
		return nil
	}
	if fileInfo.Size() > maxBytes {
		result.Stats.FilesSkipped++
		result.Infos = append(result.Infos, model.DiscoveryFileInfo{
			Path: rel, Language: language, Category: category, Bytes: fileInfo.Size(),
			Skipped: fmt.Sprintf("larger than %d KB", opts.MaxFileSizeKB),
		})
		return nil
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil {
		result.Stats.FilesSkipped++
		return nil
	}
	if isBinary(content) {
		result.Stats.FilesSkipped++
		result.Infos = append(result.Infos, model.DiscoveryFileInfo{
			Path: rel, Language: language, Category: category, Bytes: fileInfo.Size(),
			Skipped: "binary file",
		})
		return nil
	}
	if isMinified(content, name) {
		result.Stats.FilesSkipped++
		return nil
	}
	entry := &fileEntry{
		Rel: rel, Abs: path, Language: language, Category: category,
		Content: string(content), Lines: strings.Split(string(content), "\n"), Size: fileInfo.Size(),
	}
	entry.Info = model.DiscoveryFileInfo{
		Path: rel, Language: language, Category: category,
		Bytes: fileInfo.Size(), Lines: len(entry.Lines),
	}
	result.Files = append(result.Files, entry)
	result.Stats.BytesScanned += fileInfo.Size()
	return nil
}

func buildInfoList(result *scanResult) []model.DiscoveryFileInfo {
	filtered := make([]model.DiscoveryFileInfo, 0, len(result.Files)+result.Stats.FilesSkipped)
	for _, f := range result.Files {
		filtered = append(filtered, f.Info)
	}
	for _, skipped := range result.Infos {
		if skipped.Skipped != "" && len(filtered) < 800 {
			filtered = append(filtered, skipped)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Category != filtered[j].Category {
			return categoryRank(filtered[i].Category) < categoryRank(filtered[j].Category)
		}
		return filtered[i].Path < filtered[j].Path
	})
	return filtered
}
