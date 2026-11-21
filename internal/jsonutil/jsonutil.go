// Package jsonutil provides the response-inspection tooling used by the
// desktop app: JSONPath extraction, structural diffing and lightweight JSON
// Schema inference/validation. It deliberately depends only on the standard
// library so it works offline.
package jsonutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"Postwoman/internal/model"
)

// Decode parses JSON preserving numeric precision.
func Decode(raw string) (any, error) {
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

type pathToken struct {
	key     string
	index   int
	isIndex bool
}

// parsePath accepts $.a.b[0], a.b[0], ['a.b'] and JSON Pointer (/a/b/0).
func parsePath(path string) ([]pathToken, error) {
	p := strings.TrimSpace(path)
	if p == "" || p == "$" {
		return nil, nil
	}
	if strings.HasPrefix(p, "/") {
		parts := strings.Split(p, "/")[1:]
		out := make([]pathToken, 0, len(parts))
		for _, part := range parts {
			part = strings.ReplaceAll(part, "~1", "/")
			part = strings.ReplaceAll(part, "~0", "~")
			if idx, err := strconv.Atoi(part); err == nil {
				out = append(out, pathToken{index: idx, isIndex: true})
			} else {
				out = append(out, pathToken{key: part})
			}
		}
		return out, nil
	}
	if strings.HasPrefix(p, "$") {
		p = p[1:]
	}
	var tokens []pathToken
	i := 0
	for i < len(p) {
		switch p[i] {
		case '.':
			i++
		case '[':
			end := strings.IndexByte(p[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unclosed '[' in JSONPath")
			}
			inner := strings.TrimSpace(p[i+1 : i+end])
			i += end + 1
			if len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') {
				quote := inner[0]
				if inner[len(inner)-1] != quote {
					return nil, fmt.Errorf("unclosed quote in JSONPath")
				}
				tokens = append(tokens, pathToken{key: inner[1 : len(inner)-1]})
			} else if idx, err := strconv.Atoi(inner); err == nil {
				tokens = append(tokens, pathToken{index: idx, isIndex: true})
			} else {
				tokens = append(tokens, pathToken{key: inner})
			}
		default:
			start := i
			for i < len(p) && p[i] != '.' && p[i] != '[' {
				i++
			}
			tokens = append(tokens, pathToken{key: p[start:i]})
		}
	}
	return tokens, nil
}

// Extract resolves a JSONPath and returns the value plus a display form.
func Extract(raw, path string) (*model.ExtractionResult, error) {
	root, err := Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	tokens, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	current := root
	walked := "$"
	for _, tok := range tokens {
		if tok.isIndex {
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("%s is not an array", walked)
			}
			if tok.index < 0 || tok.index >= len(arr) {
				return nil, fmt.Errorf("index %d is out of range at %s", tok.index, walked)
			}
			current = arr[tok.index]
			walked += fmt.Sprintf("[%d]", tok.index)
			continue
		}
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot read property %q from a non-object at %s", tok.key, walked)
		}
		next, ok := obj[tok.key]
		if !ok {
			return nil, fmt.Errorf("property %q was not found at %s", tok.key, walked)
		}
		current = next
		walked += "." + tok.key
	}
	encoded, _ := json.Marshal(current)
	display := string(encoded)
	if s, ok := current.(string); ok {
		display = s
	}
	return &model.ExtractionResult{
		Path:    path,
		Type:    TypeName(current),
		Value:   string(encoded),
		Display: display,
	}, nil
}

// TypeName returns the JSON type name for a decoded value.
func TypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case json.Number, float64, float32, int, int64, int32:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

// Diff compares two JSON documents structurally.
func Diff(leftRaw, rightRaw string) (*model.JSONDiff, error) {
	left, err := Decode(leftRaw)
	if err != nil {
		return nil, fmt.Errorf("first document is not valid JSON: %w", err)
	}
	right, err := Decode(rightRaw)
	if err != nil {
		return nil, fmt.Errorf("second document is not valid JSON: %w", err)
	}
	out := &model.JSONDiff{Entries: []model.JSONDiffEntry{}}
	walkDiff(left, right, "$", out)
	out.Equal = len(out.Entries) == 0
	return out, nil
}

const diffLimit = 2000

func walkDiff(left, right any, path string, out *model.JSONDiff) {
	if len(out.Entries) >= diffLimit {
		out.Truncated = true
		return
	}
	lt, rt := TypeName(left), TypeName(right)
	lobj, lok := left.(map[string]any)
	robj, rok := right.(map[string]any)
	if lok && rok {
		keys := map[string]bool{}
		for k := range lobj {
			keys[k] = true
		}
		for k := range robj {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			lv, hasL := lobj[k]
			rv, hasR := robj[k]
			child := path + "." + k
			switch {
			case !hasL:
				out.Added++
				out.Entries = append(out.Entries, model.JSONDiffEntry{Path: child, Kind: "added", Right: preview(rv), RightType: TypeName(rv)})
			case !hasR:
				out.Removed++
				out.Entries = append(out.Entries, model.JSONDiffEntry{Path: child, Kind: "removed", Left: preview(lv), LeftType: TypeName(lv)})
			default:
				walkDiff(lv, rv, child, out)
			}
			if len(out.Entries) >= diffLimit {
				out.Truncated = true
				return
			}
		}
		return
	}
	larr, laok := left.([]any)
	rarr, raok := right.([]any)
	if laok && raok {
		max := len(larr)
		if len(rarr) > max {
			max = len(rarr)
		}
		for i := 0; i < max; i++ {
			child := fmt.Sprintf("%s[%d]", path, i)
			switch {
			case i >= len(larr):
				out.Added++
				out.Entries = append(out.Entries, model.JSONDiffEntry{Path: child, Kind: "added", Right: preview(rarr[i]), RightType: TypeName(rarr[i])})
			case i >= len(rarr):
				out.Removed++
				out.Entries = append(out.Entries, model.JSONDiffEntry{Path: child, Kind: "removed", Left: preview(larr[i]), LeftType: TypeName(larr[i])})
			default:
				walkDiff(larr[i], rarr[i], child, out)
			}
			if len(out.Entries) >= diffLimit {
				out.Truncated = true
				return
			}
		}
		return
	}
	if lt != rt || !equalScalar(left, right) {
		out.Changed++
		out.Entries = append(out.Entries, model.JSONDiffEntry{
			Path: path, Kind: "changed", Left: preview(left), Right: preview(right), LeftType: lt, RightType: rt,
		})
	}
}

func equalScalar(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func preview(v any) string {
	if s, ok := v.(string); ok {
		if len(s) > 400 {
			return s[:400] + "…"
		}
		return s
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	if len(raw) > 400 {
		return string(raw[:400]) + "…"
	}
	return string(raw)
}
