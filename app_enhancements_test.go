package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Postwoman/internal/model"
)

func TestAuthInheritanceAndQueryDeduplication(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		fmt.Fprintf(w, "auth=%s x=%d y=%d", r.Header.Get("Authorization"), len(r.Form["x"]), len(r.Form["y"]))
	}))
	defer srv.Close()

	collection, err := app.CreateCollection(app.workspaceID, "Inherit", "")
	if err != nil {
		t.Fatal(err)
	}
	collection.Auth = model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: "collection-token"}}
	if err := app.UpdateCollection(*collection); err != nil {
		t.Fatal(err)
	}

	req, err := app.CreateRequest(model.Request{
		CollectionID: collection.ID, Name: "inherit", Method: "GET",
		URL: srv.URL + "/items?x=1", Auth: model.Auth{Type: "inherit"},
		QueryParams: []model.KeyValue{
			{Key: "x", Value: "1", Enabled: true}, // already in the URL
			{Key: "y", Value: "2", Enabled: true},
		},
		Body: model.Body{Mode: "none"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := app.SendRequest(*req, nil, nil, nil, true, "")
	if err != nil || resp.Error != "" {
		t.Fatalf("send: %v %v", err, resp.Error)
	}
	if resp.Body != "auth=Bearer collection-token x=1 y=1" {
		t.Fatalf("unexpected response: %q", resp.Body)
	}
}

func TestFolderAuthOverridesCollection(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		fmt.Fprintf(w, "ok=%v user=%s pass=%s", ok, user, pass)
	}))
	defer srv.Close()

	collection, _ := app.CreateCollection(app.workspaceID, "Folders", "")
	collection.Auth = model.Auth{Type: "bearer", Bearer: &model.BearerAuth{Token: "collection"}}
	_ = app.UpdateCollection(*collection)

	folder, err := app.CreateFolder(collection.ID, "", "Internal", "")
	if err != nil {
		t.Fatal(err)
	}
	folder.Auth = &model.Auth{Type: "basic", Basic: &model.BasicAuth{Username: "alice", Password: "secret"}}
	if err := app.UpdateFolder(*folder); err != nil {
		t.Fatal(err)
	}

	req, _ := app.CreateRequest(model.Request{
		CollectionID: collection.ID, FolderID: &folder.ID, Name: "folder-auth", Method: "GET",
		URL: srv.URL, Auth: model.Auth{Type: "inherit"}, Body: model.Body{Mode: "none"},
	})
	resp, err := app.SendRequest(*req, nil, nil, nil, false, "")
	if err != nil || resp.Error != "" {
		t.Fatalf("send: %v %v", err, resp.Error)
	}
	if resp.Body != "ok=true user=alice pass=secret" {
		t.Fatalf("folder auth not applied: %q", resp.Body)
	}
}

func TestCookieManager(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	collection, _ := app.CreateCollection(app.workspaceID, "Cookies", "")
	req, _ := app.CreateRequest(model.Request{CollectionID: collection.ID, Name: "cookies", Method: "GET", URL: srv.URL})
	if _, err := app.SendRequest(*req, nil, nil, nil, false, ""); err != nil {
		t.Fatal(err)
	}
	cookies, err := app.ListCookies()
	if err != nil || len(cookies) == 0 {
		t.Fatalf("expected a captured cookie: %v %v", err, cookies)
	}

	if err := app.SaveCookie(model.StoredCookie{URL: srv.URL, Cookie: model.Cookie{Name: "manual", Value: "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteCookie(srv.URL, "manual"); err != nil {
		t.Fatal(err)
	}
	if err := app.ClearCookies(); err != nil {
		t.Fatal(err)
	}
	cookies, _ = app.ListCookies()
	if len(cookies) != 0 {
		t.Fatalf("cookies were not cleared: %+v", cookies)
	}
}

func TestCancelRequest(t *testing.T) {
	app := newTestApp(t)
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()

	collection, _ := app.CreateCollection(app.workspaceID, "Cancel", "")
	req, _ := app.CreateRequest(model.Request{CollectionID: collection.ID, Name: "slow", Method: "GET", URL: srv.URL})

	done := make(chan *model.Response, 1)
	go func() {
		resp, _ := app.SendRequestWithID("run-1", *req, "")
		done <- resp
	}()
	<-started
	app.CancelRequest("run-1")
	select {
	case resp := <-done:
		if resp == nil || !strings.Contains(strings.ToLower(resp.Error), "cancel") {
			t.Fatalf("expected cancellation response, got %+v", resp)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("request was not cancelled in time")
	}
}

func TestJSONTools(t *testing.T) {
	app := newTestApp(t)
	body := `{"user":{"id":7,"name":"Ada"},"tags":["x"]}`
	ext, err := app.ExtractJSONPath(body, "$.user.id")
	if err != nil || ext.Display != "7" {
		t.Fatalf("extract: %v %+v", err, ext)
	}
	diff, err := app.CompareJSON(body, `{"user":{"id":8,"name":"Ada"},"tags":["x"],"extra":true}`)
	if err != nil || diff.Equal || diff.Changed != 1 || diff.Added != 1 {
		t.Fatalf("diff: %v %+v", err, diff)
	}
	schema, err := app.GenerateJSONSchema(body)
	if err != nil || !strings.Contains(schema, "properties") {
		t.Fatalf("schema: %v %s", err, schema)
	}
	errs, err := app.ValidateJSONSchema(`{"user":{"id":"seven"}}`, schema)
	if err != nil || len(errs) == 0 {
		t.Fatalf("validation should fail: %v %v", err, errs)
	}
}

func TestAIAssistantDisabled(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.AIAssist(model.AIRequest{Prompt: "hello"}); err == nil {
		t.Fatal("expected AI assistant to be disabled by default")
	}
}
