package variables

import (
	"strings"
	"testing"
)

func TestResolveScopePrecedence(t *testing.T) {
	// Scopes are ordered highest priority first (local, environment,
	// collection, globals) — matching how the request path builds them.
	r := New(
		map[string]string{"shared": "local", "token": "local-token"},
		map[string]string{"shared": "environment", "token": "env-token"},
		map[string]string{"shared": "collection"},
		map[string]string{"shared": "global", "host": "global.example.com"},
	)

	got := r.Resolve("{{host}}/api?t={{token}}&s={{shared}}")
	if got != "global.example.com/api?t=local-token&s=local" {
		t.Fatalf("unexpected resolution: %q", got)
	}
}

func TestResolveFallsBackThroughScopes(t *testing.T) {
	r := New(
		map[string]string{},
		map[string]string{"token": "env-token"},
		map[string]string{"token": "col-token", "coll": "c"},
		map[string]string{"glob": "g"},
	)
	if got := r.Resolve("{{token}}/{{coll}}/{{glob}}"); got != "env-token/c/g" {
		t.Fatalf("unexpected fallback resolution: %q", got)
	}
}

func TestResolveRecursive(t *testing.T) {
	r := New(map[string]string{
		"base": "https://api.example.com",
		"path": "/v1/users",
		"url":  "{{base}}{{path}}",
	})
	if got := r.Resolve("{{url}}"); got != "https://api.example.com/v1/users" {
		t.Fatalf("expected recursive resolution, got %q", got)
	}
}

func TestResolveUnknownLeftIntact(t *testing.T) {
	r := New(map[string]string{"known": "value"})
	if got := r.Resolve("{{known}}/{{missing}}"); got != "value/{{missing}}" {
		t.Fatalf("expected unknown variable to remain, got %q", got)
	}
}

func TestDynamicVariables(t *testing.T) {
	r := New()
	uuid := r.Resolve("{{$guid}}")
	if len(uuid) != 36 || strings.Count(uuid, "-") != 4 {
		t.Fatalf("expected a UUID, got %q", uuid)
	}

	ts := r.Resolve("{{$timestamp}}")
	if len(ts) < 9 {
		t.Fatalf("expected a unix timestamp, got %q", ts)
	}

	if got := r.Resolve("{{$randomInt}}"); got == "" {
		t.Fatal("expected a random int")
	}
}

func TestSetUpdatesValue(t *testing.T) {
	r := New(map[string]string{"a": "1"})
	r.Set("a", "2")
	if got := r.Resolve("{{a}}"); got != "2" {
		t.Fatalf("expected 2, got %q", got)
	}
}
