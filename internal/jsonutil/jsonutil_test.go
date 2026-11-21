package jsonutil

import (
	"strings"
	"testing"
)

const sample = `{"user":{"profile":{"email":"a@b.c","age":30}},"items":[{"id":1},{"id":2}]}`

func TestExtract(t *testing.T) {
	got, err := Extract(sample, "$.user.profile.email")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if got.Display != "a@b.c" || got.Type != "string" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if _, err := Extract(sample, "$.items[1].id"); err != nil {
		t.Fatalf("array index: %v", err)
	}
	if _, err := Extract(sample, "/items/0/id"); err != nil {
		t.Fatalf("json pointer: %v", err)
	}
	if _, err := Extract(sample, "$.user.missing"); err == nil {
		t.Fatal("expected missing property error")
	}
	if _, err := Extract(`{not json`, "$.a"); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestDiff(t *testing.T) {
	left := `{"a":1,"b":{"c":true},"items":[1,2]}`
	right := `{"a":2,"b":{"c":true},"items":[1],"d":"new"}`
	diff, err := Diff(left, right)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if diff.Equal {
		t.Fatal("expected documents to differ")
	}
	if diff.Changed != 1 || diff.Added != 1 || diff.Removed != 1 {
		t.Fatalf("unexpected counters: %+v", diff)
	}
	kinds := map[string]bool{}
	for _, e := range diff.Entries {
		kinds[e.Kind] = true
		if !strings.HasPrefix(e.Path, "$") {
			t.Fatalf("bad path %q", e.Path)
		}
	}
	for _, k := range []string{"added", "removed", "changed"} {
		if !kinds[k] {
			t.Fatalf("missing %s entry", k)
		}
	}
	same, err := Diff(left, left)
	if err != nil || !same.Equal {
		t.Fatalf("identical documents should be equal: %v %+v", err, same)
	}
}

func TestSchemaRoundTrip(t *testing.T) {
	value, err := Decode(`{"id":1,"tags":["a"],"meta":{"active":true}}`)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := MarshalSchema(InferSchema(value))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(schema, "\"required\"") || !strings.Contains(schema, "\"integer\"") {
		t.Fatalf("schema missing expected keywords: %s", schema)
	}
	errs, err := Validate(`{"id":2,"tags":["b"],"meta":{"active":false}}`, schema)
	if err != nil || len(errs) != 0 {
		t.Fatalf("valid document rejected: %v %v", err, errs)
	}
	errs, err = Validate(`{"id":"x","tags":[]}`, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) < 2 {
		t.Fatalf("expected type and missing-property errors, got %v", errs)
	}
}
