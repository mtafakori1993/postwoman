package discovery

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"time"

	"Postwoman/internal/model"

	"github.com/rs/zerolog/log"
)

// run executes one discovery session end to end.
func (m *Manager) run(ctx context.Context, s *session, settings model.AppSettings, emit func(model.DiscoveryProgress)) {
	defer func() {
		if s.cancel != nil {
			s.cancel()
		}
	}()
	progress := func(stage string, percent int, message, current string) {
		s.mu.Lock()
		s.state.Progress = model.DiscoveryProgress{
			SessionID: s.state.ID, Stage: stage, Message: message,
			Percent: percent, Current: current, FilesSeen: s.state.Stats.FilesSeen,
		}
		s.mu.Unlock()
		if emit != nil {
			emit(s.state.Progress)
		}
	}
	fail := func(err error) {
		s.mu.Lock()
		s.state.Status = "failed"
		s.state.Error = err.Error()
		s.state.FinishedAt = time.Now().UnixMilli()
		s.state.Progress = model.DiscoveryProgress{
			SessionID: s.state.ID, Stage: "failed", Message: err.Error(),
			Done: true, Error: err.Error(),
		}
		s.mu.Unlock()
		if emit != nil {
			emit(s.state.Progress)
		}
	}

	// A malformed project must never take down the desktop app: convert any
	// panic in the analysis pipeline into a failed session with a stack trace.
	defer func() {
		if recovered := recover(); recovered != nil {
			stack := string(debug.Stack())
			log.Error().Interface("panic", recovered).Msg("discovery agent panicked")
			fail(fmt.Errorf("discovery agent crashed: %v\n%s", recovered, stack))
		}
	}()

	progress("scanning", 2, "Scanning project files", "")
	result, err := scanProject(ctx, s.state.Options, func(percent int, message, current string) {
		progress("scanning", percent, message, current)
	})
	if err != nil {
		if ctx.Err() != nil {
			s.mu.Lock()
			s.state.Status = "cancelled"
			s.state.Progress = model.DiscoveryProgress{SessionID: s.state.ID, Stage: "cancelled", Message: "Analysis cancelled", Done: true, Percent: 100}
			s.mu.Unlock()
			return
		}
		fail(err)
		return
	}
	s.files = map[string]*fileEntry{}
	for _, file := range result.Files {
		s.files[file.Rel] = file
	}
	s.mu.Lock()
	s.state.ProjectName = result.ProjectName
	s.state.Stats.FilesSeen = result.Stats.FilesSeen
	s.mu.Unlock()

	progress("framework", 10, "Detecting framework and language", "")
	profile := detectProfile(result.Files)
	s.profile = profile
	s.mu.Lock()
	s.state.Languages = profile.Languages
	s.state.Frameworks = profile.Frameworks
	s.mu.Unlock()

	analyzer := newAnalyzer(profile, result.Files)
	endpoints, warnings := analyzer.analyze(ctx, progress)
	if ctx.Err() != nil {
		s.mu.Lock()
		s.state.Status = "cancelled"
		s.state.Progress = model.DiscoveryProgress{SessionID: s.state.ID, Stage: "cancelled", Message: "Analysis cancelled", Done: true, Percent: 100}
		s.mu.Unlock()
		return
	}

	if s.state.Options.UseAI {
		aiWarnings := analyzer.enrichWithAI(ctx, settings, m.ai, s.state.Options, progress)
		warnings = append(warnings, aiWarnings...)
		endpoints = analyzer.sortedEndpoints()
	}

	// Release the full source of files that produced no routes; the review
	// screen can re-read them from disk on demand. Large projects (for example
	// WordPress trees) otherwise keep the whole repository in memory.
	for _, file := range s.files {
		info := analyzer.fileInfos[file.Rel]
		if info == nil || !info.HasRoutes {
			file.Content = ""
			file.Lines = nil
		}
	}

	progress("generating", 88, "Generating Postwoman collection", "")
	stats := result.Stats
	stats.FilesAnalyzed = len(analyzer.fileInfos)
	stats.FilesSkipped = result.Stats.FilesSkipped
	stats.Endpoints = len(endpoints)
	for _, file := range result.Files {
		if file.Category == CategoryRoute {
			stats.RouteFiles++
		}
	}
	for _, endpoint := range endpoints {
		if endpoint.Confidence >= 80 {
			stats.HighConfidence++
		} else {
			stats.LowConfidence++
		}
		if endpoint.Origin == "openapi" {
			stats.OpenAPISources++
		}
		if endpoint.Origin == "merged" {
			stats.AIFilesSent++
		}
	}
	folders := []string{}
	seenFolders := map[string]bool{}
	for _, endpoint := range endpoints {
		if endpoint.Folder != "" && !seenFolders[endpoint.Folder] {
			seenFolders[endpoint.Folder] = true
			folders = append(folders, endpoint.Folder)
		}
	}
	sort.Strings(folders)

	infos := make([]model.DiscoveryFileInfo, 0, len(result.Infos))
	for _, info := range result.Infos {
		if updated, ok := analyzer.fileInfos[info.Path]; ok {
			info = *updated
		}
		infos = append(infos, info)
	}

	s.mu.Lock()
	s.state.Stats = stats
	s.state.Endpoints = endpoints
	s.state.Folders = folders
	s.state.Files = infos
	s.state.Warnings = appendUniqueStrings(s.state.Warnings, warnings)
	s.state.Options.BaseURL = firstNonEmpty(s.state.Options.BaseURL, profile.BaseURL)
	s.mu.Unlock()

	collection, genErr := BuildCollection(&s.state, model.DiscoveryGenerateRequest{
		SessionID: s.state.ID, CollectionName: s.state.Options.CollectionName, BaseURL: s.state.Options.BaseURL,
	})
	if genErr != nil {
		s.mu.Lock()
		s.state.Warnings = appendUniqueStrings(s.state.Warnings, []string{"collection generation failed: " + genErr.Error()})
		s.mu.Unlock()
	} else {
		validationWarnings, validateErr := ValidateCollection(collection)
		s.mu.Lock()
		if validateErr != nil {
			s.state.Warnings = appendUniqueStrings(s.state.Warnings, []string{"generated collection failed validation: " + validateErr.Error()})
		} else {
			s.state.Collection = collection
			s.state.Warnings = appendUniqueStrings(s.state.Warnings, validationWarnings)
		}
		s.mu.Unlock()
	}

	progress("done", 100, "Analysis complete", "")
	s.mu.Lock()
	s.state.Status = "completed"
	s.state.FinishedAt = time.Now().UnixMilli()
	s.state.Progress = model.DiscoveryProgress{
		SessionID: s.state.ID, Stage: "done", Message: "Analysis complete",
		Percent: 100, Done: true, Endpoints: len(endpoints), FilesSeen: stats.FilesSeen,
	}
	s.mu.Unlock()
	if emit != nil {
		emit(s.state.Progress)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
