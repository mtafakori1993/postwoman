// Package collections ties the database repository to the import/export
// package so the Wails layer stays thin.
package collections

import (
	"context"
	"fmt"

	"Postwoman/internal/database"
	"Postwoman/internal/importexport"
	"Postwoman/internal/model"
)

type Manager struct {
	repo *database.Repository
}

func NewManager(repo *database.Repository) *Manager {
	return &Manager{repo: repo}
}

// CreateCollection creates an empty Postman-compatible collection.
func (m *Manager) CreateCollection(ctx context.Context, workspaceID, name, description string) (*model.Collection, error) {
	return m.repo.CreateCollection(ctx, workspaceID, model.CollectionInfo{
		Name:        name,
		Description: description,
		Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}, []model.Variable{}, model.Auth{Type: "noauth"}, []model.CollectionEvent{})
}

// Import detects the document format and persists it.
func (m *Manager) Import(ctx context.Context, workspaceID string, data []byte, filename string) (*model.ImportSummary, error) {
	parsed, err := importexport.Parse(data, filename)
	if err != nil {
		return nil, err
	}
	collectionIDs, environmentIDs, err := importexport.Persist(ctx, m.repo, workspaceID, parsed)
	if err != nil {
		return nil, err
	}
	summary := &model.ImportSummary{
		Format:       parsed.Format,
		Collections:  []string{},
		Environments: []string{},
		Warnings:     parsed.Warnings,
	}
	for _, id := range collectionIDs {
		if c, err := m.repo.GetCollection(ctx, id); err == nil && c != nil {
			summary.Collections = append(summary.Collections, c.Info.Name)
		}
	}
	for _, id := range environmentIDs {
		if e, err := m.repo.GetEnvironment(ctx, id); err == nil && e != nil {
			summary.Environments = append(summary.Environments, e.Name)
		}
	}
	if len(summary.Collections) == 0 && len(summary.Environments) == 0 && len(parsed.Globals) == 0 {
		return nil, fmt.Errorf("the document did not contain anything to import")
	}
	summary.Message = fmt.Sprintf("Imported %d collection(s) and %d environment(s)", len(summary.Collections), len(summary.Environments))
	return summary, nil
}

func (m *Manager) ExportCollection(ctx context.Context, collectionID string) ([]byte, error) {
	return importexport.ExportCollection(ctx, m.repo, collectionID)
}

func (m *Manager) ExportEnvironment(ctx context.Context, environmentID string) ([]byte, error) {
	return importexport.ExportEnvironment(ctx, m.repo, environmentID)
}

// ExportRequestAsCurl renders a request as a cURL command.
func ExportRequestAsCurl(req model.Request) string {
	return importexport.ExportAsCurl(&req)
}
