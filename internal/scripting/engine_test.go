package scripting

import (
	"testing"
	"time"

	"Postwoman/internal/model"
)

func newEngine() *Engine { return NewEngine(5 * time.Second) }

func TestPassingAndFailingTests(t *testing.T) {
	engine := newEngine()
	ctx := &Context{
		Request:     model.Request{URL: "https://example.com"},
		Response:    &model.Response{StatusCode: 200, StatusText: "OK", Body: `{"ok":true,"n":5}`},
		Environment: map[string]string{},
		Globals:     map[string]string{},
	}

	res, err := engine.Run(`
		pm.test("status is 200", () => { pm.response.to.have.status(200); });
		pm.test("body says ok", () => {
			const data = pm.response.json();
			pm.expect(data.ok).to.be.true;
			pm.expect(data.n).to.equal(5);
		});
		pm.test("this fails", () => { pm.expect(1).to.equal(2); });
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(res.Tests))
	}
	if !res.Tests[0].Passed || !res.Tests[1].Passed {
		t.Fatalf("expected first two tests to pass: %+v", res.Tests)
	}
	if res.Tests[2].Passed {
		t.Fatalf("expected third test to fail")
	}
	if res.Tests[2].Error == "" {
		t.Fatal("expected an error message on the failing test")
	}
}

func TestEnvironmentAndGlobalsMutation(t *testing.T) {
	engine := newEngine()
	ctx := &Context{
		Request:     model.Request{URL: "https://example.com"},
		Environment: map[string]string{"existing": "1"},
		Globals:     map[string]string{},
	}
	res, err := engine.Run(`
		pm.environment.set("token", "abc");
		pm.globals.set("g", "1");
		pm.collectionVariables.set("c", "2");
		console.log("hello", 42);
	`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Environment["token"] != "abc" {
		t.Fatalf("environment not updated: %+v", res.Environment)
	}
	if res.Globals["g"] != "1" {
		t.Fatalf("globals not updated: %+v", res.Globals)
	}
	if res.CollectionVariables["c"] != "2" {
		t.Fatalf("collection variables not updated: %+v", res.CollectionVariables)
	}
	if len(res.Console) != 1 || res.Console[0].Args[0] != "hello" {
		t.Fatalf("console not captured: %+v", res.Console)
	}
}

func TestInfiniteLoopIsInterrupted(t *testing.T) {
	engine := NewEngine(300 * time.Millisecond)
	ctx := &Context{Request: model.Request{}, Environment: map[string]string{}}
	start := time.Now()
	_, err := engine.Run(`while (true) {}`, ctx)
	if err == nil {
		t.Fatal("expected the script to be interrupted")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("interrupt took too long: %v", elapsed)
	}
}

func TestNoFilesystemAccess(t *testing.T) {
	engine := newEngine()
	ctx := &Context{Request: model.Request{}, Environment: map[string]string{}}
	res, err := engine.Run(`try { require('fs'); } catch (e) { pm.environment.set("blocked", "yes"); }`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Environment["blocked"] != "yes" {
		t.Fatal("expected require() to be unavailable in the sandbox")
	}
}
