package importexport

import (
	"Postwoman/internal/model"
)

// Preview summarises a parsed document for the import dialog without writing
// anything to the database.
func Preview(parsed *Parsed) *model.ImportPreview {
	if parsed == nil {
		return &model.ImportPreview{Collections: []model.ImportItem{}, Environments: []model.ImportEnv{}, Warnings: []string{}, Errors: []string{}}
	}
	preview := &model.ImportPreview{
		Format:       parsed.Format,
		Name:         parsed.Name,
		Collections:  []model.ImportItem{},
		Environments: []model.ImportEnv{},
		Warnings:     append([]string{}, parsed.Warnings...),
		Errors:       []string{},
	}
	for _, c := range parsed.Collections {
		if c == nil {
			continue
		}
		preview.Collections = append(preview.Collections, model.ImportItem{
			Name:         c.Info.Name,
			FolderCount:  len(c.Folders),
			RequestCount: len(c.Requests),
		})
	}
	for _, e := range parsed.Environments {
		preview.Environments = append(preview.Environments, model.ImportEnv{Name: e.Name, Values: len(e.Values)})
	}
	if len(parsed.Globals) > 0 {
		preview.Environments = append(preview.Environments, model.ImportEnv{Name: "Global Variables", Values: len(parsed.Globals)})
	}
	if len(preview.Collections) == 0 && len(preview.Environments) == 0 {
		preview.Errors = append(preview.Errors, "The document did not contain any collections, requests or variables")
	}
	return preview
}
