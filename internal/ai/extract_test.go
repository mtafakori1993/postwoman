package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractJSONToleratesProseAndFences(t *testing.T) {
	reply := "Here is the result you asked for:\n```json\n{\"endpoints\":[]}\n```\nLet me know if you need more."
	got, err := ExtractJSON(reply)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if got != `{"endpoints":[]}` {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSONRepairsTruncatedEnvelope(t *testing.T) {
	// This mirrors a reasoning model that ran out of completion tokens halfway
	// through the last endpoint.
	reply := `{"endpoints":[{"id":"post-login","name":"User login","statusCodes":[200,`
	got, err := ExtractJSON(reply)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	want := `{"endpoints":[{"id":"post-login","name":"User login","statusCodes":[200]}]}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if !json.Valid([]byte(got)) {
		t.Fatalf("repaired JSON is invalid: %s", got)
	}
}

func TestExtractJSONDropsIncompleteTrailingValue(t *testing.T) {
	reply := `{"endpoints":[{"id":"post-login","name":"Use`
	got, err := ExtractJSON(reply)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	want := `{"endpoints":[{"id":"post-login"}]}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestExtractJSONRemovesTrailingCommas(t *testing.T) {
	got, err := ExtractJSON(`{"endpoints":[],}`)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if got != `{"endpoints":[]}` {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSONRejectsProseOnly(t *testing.T) {
	_, err := ExtractJSON("I could not determine any endpoints from these snippets.")
	if err == nil || !strings.Contains(err.Error(), "no complete JSON") {
		t.Fatalf("error = %v, want a no-complete-JSON message", err)
	}
}

func TestExtractJSONHandlesBOMAndCRLF(t *testing.T) {
	got, err := ExtractJSON("\ufeff{\r\n\"endpoints\":[]\r\n}")
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if got != "{\r\n\"endpoints\":[]\r\n}" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSONIgnoresBracesInStrings(t *testing.T) {
	reply := `{"description":"uses {curly} braces and \"quotes\"","endpoints":[]}`
	got, err := ExtractJSON(reply)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if got != reply {
		t.Fatalf("got %q", got)
	}
}
