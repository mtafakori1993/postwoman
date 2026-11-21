package database

import (
	"context"
	"path/filepath"
	"testing"

	"Postwoman/internal/model"
)

func openTestRepo(t *testing.T) (*DB, *Repository) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := New(path)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	return db, NewRepository(db)
}

func TestCollectionRequestFolderRoundTrip(t *testing.T) {
	ctx := context.Background()
	db, repo := openTestRepo(t)
	defer db.Close()

	ws, err := repo.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}

	collection, err := repo.CreateCollection(ctx, ws.ID, model.CollectionInfo{
		Name: "Demo", Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
	}, []model.Variable{{Key: "base", Value: "https://api.example.com", Enabled: true, Type: "default"}},
		model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: "t"}}, nil)
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}

	folder, err := repo.CreateFolder(ctx, collection.ID, nil, "Users", "user endpoints", 0, nil, nil)
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}

	created, err := repo.CreateRequest(ctx, &model.Request{
		CollectionID: collection.ID,
		FolderID:     &folder.ID,
		Name:         "List users",
		Method:       "GET",
		URL:          "{{base}}/users",
		Headers:      []model.KeyValue{{Key: "Accept", Value: "application/json", Enabled: true}},
		Body:         model.Body{Mode: "none"},
		Auth:         model.Auth{Type: "inherit"},
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected an id to be assigned")
	}

	got, err := repo.GetRequest(ctx, created.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got == nil || got.Name != "List users" || got.Method != "GET" {
		t.Fatalf("unexpected request: %+v", got)
	}
	if got.FolderID == nil || *got.FolderID != folder.ID {
		t.Fatalf("folder id not persisted: %+v", got.FolderID)
	}
	if len(got.Headers) != 1 || got.Headers[0].Key != "Accept" {
		t.Fatalf("headers not persisted: %+v", got.Headers)
	}

	all, err := repo.ListAllRequests(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list all requests: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "persist.db")

	db1, err := New(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	repo1 := NewRepository(db1)
	ws, err := repo1.EnsureDefaultWorkspace(ctx)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	collection, err := repo1.CreateCollection(ctx, ws.ID, model.CollectionInfo{Name: "Persisted", Schema: "s"}, nil, model.Auth{Type: "noauth"}, nil)
	if err != nil {
		t.Fatalf("collection: %v", err)
	}
	if _, err := repo1.CreateRequest(ctx, &model.Request{
		CollectionID: collection.ID, Name: "Ping", Method: "GET", URL: "https://example.com/ping",
		Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
	}); err != nil {
		t.Fatalf("request: %v", err)
	}
	if err := repo1.SaveTabs(ctx, []model.Tab{{ID: "t1", Title: "Ping", Protocol: "http"}}); err != nil {
		t.Fatalf("save tabs: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen the same file and confirm the data survived.
	db2, err := New(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	repo2 := NewRepository(db2)

	collections, err := repo2.ListCollections(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list collections: %v", err)
	}
	if len(collections) != 1 || collections[0].Info.Name != "Persisted" {
		t.Fatalf("collection did not persist: %+v", collections)
	}

	requests, err := repo2.ListAllRequests(ctx, ws.ID)
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(requests) != 1 || requests[0].Name != "Ping" {
		t.Fatalf("request did not persist: %+v", requests)
	}

	tabs, err := repo2.ListTabs(ctx)
	if err != nil {
		t.Fatalf("list tabs: %v", err)
	}
	if len(tabs) != 1 || tabs[0].Title != "Ping" {
		t.Fatalf("tabs did not persist: %+v", tabs)
	}
}

func TestNestedFolderDeleteRemovesDescendants(t *testing.T) {
	ctx := context.Background()
	db, repo := openTestRepo(t)
	defer db.Close()

	ws, _ := repo.EnsureDefaultWorkspace(ctx)
	collection, _ := repo.CreateCollection(ctx, ws.ID, model.CollectionInfo{Name: "C", Schema: "s"}, nil, model.Auth{Type: "noauth"}, nil)
	parent, _ := repo.CreateFolder(ctx, collection.ID, nil, "Parent", "", 0, nil, nil)
	child, _ := repo.CreateFolder(ctx, collection.ID, &parent.ID, "Child", "", 0, nil, nil)
	if _, err := repo.CreateRequest(ctx, &model.Request{
		CollectionID: collection.ID, FolderID: &child.ID, Name: "R", Method: "GET",
		URL: "https://x", Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
	}); err != nil {
		t.Fatalf("request: %v", err)
	}

	if err := repo.DeleteFolder(ctx, parent.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}

	folders, _ := repo.ListFolders(ctx, collection.ID)
	if len(folders) != 0 {
		t.Fatalf("expected nested folders to be deleted, got %+v", folders)
	}
	requests, _ := repo.ListRequests(ctx, collection.ID)
	if len(requests) != 0 {
		t.Fatalf("expected requests in nested folders to be deleted, got %+v", requests)
	}
}
