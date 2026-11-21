package runner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"Postwoman/internal/database"
	"Postwoman/internal/httpclient"
	"Postwoman/internal/model"
	"Postwoman/internal/variables"
)

func TestRunCollectionIterationsAndData(t *testing.T) {
	ctx := context.Background()
	db, err := database.New(filepath.Join(t.TempDir(), "runner.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ws, _ := repo.EnsureDefaultWorkspace(ctx)
	collection, err := repo.CreateCollection(ctx, ws.ID, model.CollectionInfo{Name: "Run", Schema: "s"}, nil, model.Auth{Type: "noauth"}, nil)
	if err != nil {
		t.Fatalf("collection: %v", err)
	}
	for _, name := range []string{"one", "two"} {
		if _, err := repo.CreateRequest(ctx, &model.Request{
			CollectionID: collection.ID, Name: name, Method: "GET",
			URL: "https://example.com/" + name, Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
		}); err != nil {
			t.Fatalf("request: %v", err)
		}
	}

	calls := 0
	exec := func(_ context.Context, req *model.Request, _ map[string]map[string]string) (*model.Response, *model.Request, error) {
		calls++
		return &model.Response{StatusCode: 200, StatusText: "OK", Headers: []model.KeyValue{}, Cookies: []model.Cookie{}, Tests: []model.TestResult{}}, req, nil
	}

	var progress int
	result, err := Run(ctx, repo, exec, model.RunOptions{
		CollectionID: collection.ID,
		Iterations:   3,
	}, func(model.RequestResult) { progress++ })
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 6 {
		t.Fatalf("expected 6 executions (2 requests x 3 iterations), got %d", calls)
	}
	if result.TotalRequests != 6 {
		t.Fatalf("expected TotalRequests=6, got %d", result.TotalRequests)
	}
	if len(result.Iterations) != 3 {
		t.Fatalf("expected 3 iterations, got %d", len(result.Iterations))
	}
	if progress != 6 {
		t.Fatalf("expected 6 progress callbacks, got %d", progress)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
}

func TestRunAgainstRealServer(t *testing.T) {
	ctx := context.Background()
	db, err := database.New(filepath.Join(t.TempDir(), "runner2.db"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	ws, _ := repo.EnsureDefaultWorkspace(ctx)
	collection, _ := repo.CreateCollection(ctx, ws.ID, model.CollectionInfo{Name: "Run2", Schema: "s"}, nil, model.Auth{Type: "noauth"}, nil)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	}))
	defer srv.Close()

	created, err := repo.CreateRequest(ctx, &model.Request{
		CollectionID: collection.ID, Name: "ping", Method: "GET",
		URL: srv.URL + "/ping", Body: model.Body{Mode: "none"}, Auth: model.Auth{Type: "noauth"},
	})
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	client, err := httpclient.NewClient()
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	exec := func(ctx context.Context, req *model.Request, vars map[string]map[string]string) (*model.Response, *model.Request, error) {
		resp, err := client.Do(ctx, req, variables.New(vars["local"], vars["environment"], vars["collection"], vars["globals"]))
		return resp, req, err
	}

	result, err := Run(ctx, repo, exec, model.RunOptions{CollectionID: collection.ID, RequestIDs: []string{created.ID}}, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.TotalRequests != 1 {
		t.Fatalf("expected 1 request, got %d", result.TotalRequests)
	}
	got := result.Iterations[0].Requests[0]
	if got.Response == nil {
		t.Fatal("expected a response to be recorded")
	}
	if got.Response.StatusCode != 200 || !strings.Contains(got.Response.Body, "pong") {
		t.Fatalf("unexpected response: status=%d body=%q", got.Response.StatusCode, got.Response.Body)
	}
}
