package discovery

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"Postwoman/internal/ai"
	"Postwoman/internal/model"
)

func (a *analyzer) applySuggestions(suggestions []aiSuggestion) {
	for _, suggestion := range suggestions {
		ep := a.resolveSuggestionEndpoint(suggestion)
		if ep == nil {
			continue
		}
		if strings.TrimSpace(suggestion.Name) != "" && ep.Name == "" {
			ep.Name = suggestion.Name
		}
		if strings.TrimSpace(suggestion.Description) != "" && ep.Description == "" {
			ep.Description = suggestion.Description
		}
		if suggestion.AuthRequired != nil && *suggestion.AuthRequired {
			if !ep.AuthRequired {
				ep.AuthRequired = true
				ep.AuthNotes = "detected by AI reasoning"
			}
			if ep.AuthType == "" && suggestion.AuthType != "" {
				ep.AuthType = suggestion.AuthType
			}
		}
		if suggestion.RequestBody != nil && ep.RequestBody == "" {
			encoded, err := json.MarshalIndent(suggestion.RequestBody, "", "  ")
			if err == nil {
				text := string(encoded)
				if text != "null" && text != "{}" && text != "[]" {
					ep.RequestBody = text
					if ep.ContentType == "" {
						ep.ContentType = "application/json"
					}
				}
			}
		}
		ep.QueryParams = appendUniqueParams(ep.QueryParams, suggestion.QueryParams)
		ep.PathParams = appendUniqueParams(ep.PathParams, suggestion.PathParams)
		ep.Headers = appendUniqueParams(ep.Headers, suggestion.Headers)
		ep.StatusCodes = appendUniqueInts(ep.StatusCodes, suggestion.StatusCodes)
		ep.Notes = appendUniqueStrings(ep.Notes, suggestion.Notes)
		if suggestion.Confidence > ep.Confidence {
			ep.Confidence = suggestion.Confidence
		}
		if ep.Origin == "static" {
			ep.Origin = "merged"
		}
	}
}

// resolveSuggestionEndpoint matches a suggestion to a static endpoint. Models
// occasionally rewrite or omit the id, so method+path is used as a fallback.
func (a *analyzer) resolveSuggestionEndpoint(suggestion aiSuggestion) *model.DiscoveredEndpoint {
	if ep := a.endpoints[suggestion.ID]; ep != nil {
		return ep
	}
	if suggestion.ID != "" {
		for key, ep := range a.endpoints {
			if strings.EqualFold(key, suggestion.ID) || strings.EqualFold(ep.ID, suggestion.ID) {
				return ep
			}
		}
	}
	if suggestion.Method != "" && suggestion.Path != "" {
		if ep := a.endpoints[suggestion.Method+" "+normalizeRoutePath(suggestion.Path)]; ep != nil {
			return ep
		}
	}
	return nil
}

// decodeAISuggestions extracts suggestions from the several envelope shapes
// providers return: {"endpoints":[...]}, a bare array, or one of the common
// alternative keys. A single object is accepted too for a one-endpoint batch.
func decodeAISuggestions(reply string) ([]aiSuggestion, error) {
	content, err := ai.ExtractJSON(reply)
	if err != nil {
		return nil, err
	}
	var envelope aiSuggestionEnvelope
	if err := json.Unmarshal([]byte(content), &envelope); err == nil && envelope.Endpoints != nil {
		return envelope.Endpoints, nil
	}
	var list []aiSuggestion
	if err := json.Unmarshal([]byte(content), &list); err == nil {
		return list, nil
	}
	var alternatives struct {
		Suggestions []aiSuggestion `json:"suggestions"`
		Results     []aiSuggestion `json:"results"`
		Data        []aiSuggestion `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &alternatives); err == nil {
		for _, candidate := range [][]aiSuggestion{alternatives.Suggestions, alternatives.Results, alternatives.Data} {
			if candidate != nil {
				return candidate, nil
			}
		}
	}
	var single aiSuggestion
	if err := json.Unmarshal([]byte(content), &single); err == nil && single.ID != "" {
		return []aiSuggestion{single}, nil
	}
	return nil, fmt.Errorf("reply did not contain an endpoints array")
}

// UnmarshalJSON accepts the loose value shapes models produce. Strict struct
// decoding previously made one stringified number (for example "statusCodes":
// ["200"]) discard every suggestion in the batch.
func (s *aiSuggestion) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*s = aiSuggestion{}
		return nil
	}
	// A stray string/number in the endpoints array should not discard the whole
	// batch; treat it as an empty (and therefore ignored) suggestion.
	if trimmed[0] != '{' {
		*s = aiSuggestion{}
		return nil
	}
	type rawSuggestion struct {
		ID           aiText            `json:"id"`
		Method       aiText            `json:"method"`
		Path         aiText            `json:"path"`
		Name         aiText            `json:"name"`
		Description  aiText            `json:"description"`
		AuthRequired json.RawMessage   `json:"authRequired"`
		AuthType     aiText            `json:"authType"`
		RequestBody  json.RawMessage   `json:"requestBody"`
		QueryParams  aiParamList       `json:"queryParams"`
		PathParams   aiParamList       `json:"pathParams"`
		Headers      aiParamList       `json:"headers"`
		StatusCodes  []json.RawMessage `json:"statusCodes"`
		Confidence   json.RawMessage   `json:"confidence"`
		Notes        json.RawMessage   `json:"notes"`
	}
	var raw rawSuggestion
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ID = strings.TrimSpace(string(raw.ID))
	s.Method = strings.ToUpper(strings.TrimSpace(string(raw.Method)))
	s.Path = strings.TrimSpace(string(raw.Path))
	s.Name = strings.TrimSpace(string(raw.Name))
	s.Description = strings.TrimSpace(string(raw.Description))
	s.AuthRequired = flexibleBool(raw.AuthRequired)
	s.AuthType = normalizeAuthType(string(raw.AuthType))
	s.RequestBody = flexibleJSONValue(raw.RequestBody)
	s.QueryParams = normalizeParams(raw.QueryParams, "query")
	s.PathParams = normalizeParams(raw.PathParams, "path")
	s.Headers = normalizeParams(raw.Headers, "header")
	s.StatusCodes = flexibleStatusCodes(raw.StatusCodes)
	s.Confidence = flexibleInt(raw.Confidence, 0, 100)
	s.Notes = flexibleStrings(raw.Notes)
	return nil
}

// aiText accepts a string, number, boolean or null where the model was asked
// for a string.
type aiText string

func (t *aiText) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*t = ""
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*t = aiText(value)
		return nil
	}
	*t = aiText(strings.Trim(trimmed, `"`))
	return nil
}

func firstAIText(values ...aiText) aiText {
	for _, value := range values {
		if strings.TrimSpace(string(value)) != "" {
			return value
		}
	}
	return ""
}

// aiParamList accepts the several parameter shapes models emit: an array of
// objects, an array of plain names, a single object, or a single name.
type aiParamList []model.DiscoveryParam

func (p *aiParamList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*p = nil
		return nil
	}
	if trimmed[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			*p = nil
			return nil
		}
		out := make([]model.DiscoveryParam, 0, len(items))
		for _, item := range items {
			if param, ok := decodeAIParam(item); ok {
				out = append(out, param)
			}
		}
		if len(out) == 0 {
			*p = nil
		} else {
			*p = out
		}
		return nil
	}
	if param, ok := decodeAIParam(json.RawMessage(data)); ok {
		*p = []model.DiscoveryParam{param}
		return nil
	}
	*p = nil
	return nil
}

func decodeAIParam(data json.RawMessage) (model.DiscoveryParam, bool) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return model.DiscoveryParam{}, false
	}
	if trimmed[0] == '"' {
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return model.DiscoveryParam{}, false
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return model.DiscoveryParam{}, false
		}
		return model.DiscoveryParam{Name: name, Type: "string"}, true
	}
	if trimmed[0] != '{' {
		return model.DiscoveryParam{}, false
	}
	var obj struct {
		Name        aiText          `json:"name"`
		Key         aiText          `json:"key"`
		Param       aiText          `json:"param"`
		In          aiText          `json:"in"`
		Location    aiText          `json:"location"`
		Type        aiText          `json:"type"`
		Required    json.RawMessage `json:"required"`
		Description aiText          `json:"description"`
		Example     aiText          `json:"example"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return model.DiscoveryParam{}, false
	}
	name := strings.TrimSpace(string(firstAIText(obj.Name, obj.Key, obj.Param)))
	if name == "" {
		return model.DiscoveryParam{}, false
	}
	return model.DiscoveryParam{
		Name:        name,
		In:          strings.ToLower(strings.TrimSpace(string(firstAIText(obj.In, obj.Location)))),
		Type:        strings.TrimSpace(string(obj.Type)),
		Required:    flexibleBoolValue(obj.Required),
		Description: strings.TrimSpace(string(obj.Description)),
		Example:     strings.TrimSpace(string(obj.Example)),
	}, true
}

func normalizeParams(params aiParamList, defaultIn string) []model.DiscoveryParam {
	if len(params) == 0 {
		return nil
	}
	out := make([]model.DiscoveryParam, 0, len(params))
	for _, param := range params {
		param.Name = strings.TrimSpace(param.Name)
		if param.Name == "" {
			continue
		}
		param.In = normalizeParamLocation(param.In, defaultIn)
		param.Type = strings.TrimSpace(param.Type)
		if param.Type == "" {
			param.Type = "string"
		}
		param.Description = strings.TrimSpace(param.Description)
		param.Example = strings.TrimSpace(param.Example)
		out = append(out, param)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeParamLocation(location, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(location)) {
	case "", "unknown":
		return fallback
	case "header", "headers", "request", "response", "request-header", "response-header":
		return "header"
	case "query", "querystring", "query-string", "query-param", "queryparam":
		return "query"
	case "path", "route", "path-param", "pathparam":
		return "path"
	case "cookie", "cookies":
		return "cookie"
	case "body", "payload":
		return "body"
	default:
		return fallback
	}
}

// flexibleBool accepts a JSON boolean, "true"/"false" strings, or 0/1.
func flexibleBool(raw json.RawMessage) *bool {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return &value
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "yes", "required", "1":
			result := true
			return &result
		case "false", "no", "optional", "none", "0", "":
			result := false
			return &result
		}
		return nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		result := number != 0
		return &result
	}
	return nil
}

func flexibleBoolValue(raw json.RawMessage) bool {
	if value := flexibleBool(raw); value != nil {
		return *value
	}
	return false
}

// flexibleInt accepts a number or a numeric string, clamped to [minimum, maximum].
func flexibleInt(raw json.RawMessage, minimum, maximum int) int {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return minimum
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return clampInt(int(number), minimum, maximum)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		text = strings.TrimSpace(strings.TrimSuffix(text, "%"))
		if value, err := strconv.Atoi(text); err == nil {
			return clampInt(value, minimum, maximum)
		}
		if value, err := strconv.ParseFloat(text, 64); err == nil {
			return clampInt(int(value), minimum, maximum)
		}
	}
	return minimum
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// flexibleJSONValue normalises request bodies. Models sometimes answer with a
// stringified JSON object; decoding it here lets the UI show real fields instead
// of one escaped string. Placeholder values ("none", "null", …) become nil.
func flexibleJSONValue(raw json.RawMessage) any {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	text, isString := value.(string)
	if isString {
		trimmed := strings.TrimSpace(text)
		switch strings.ToLower(trimmed) {
		case "", "none", "null", "n/a", "na", "no body":
			return nil
		}
		if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed)) {
			var nested any
			if err := json.Unmarshal([]byte(trimmed), &nested); err == nil {
				return nested
			}
		}
		return text
	}
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return nil
		}
	case []any:
		if len(typed) == 0 {
			return nil
		}
	}
	return value
}

func normalizeAuthType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bearer", "bearer token", "jwt", "jwt bearer", "token":
		return "bearer"
	case "basic", "basic auth":
		return "basic"
	case "apikey", "api key", "api-key", "x-api-key", "key":
		return "apikey"
	case "oauth", "oauth2", "oauth 2":
		return "oauth2"
	case "session", "cookie", "cookie-based", "jwt cookie", "session cookie":
		return "session"
	default:
		return "unknown"
	}
}

// flexibleStrings accepts a string, an array of strings, or an array of objects
// carrying text/note/description.
func flexibleStrings(raw json.RawMessage) []string {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return splitNotes(text)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err == nil {
		out := []string{}
		for _, item := range items {
			var value string
			if err := json.Unmarshal(item, &value); err == nil {
				out = append(out, splitNotes(value)...)
				continue
			}
			var object struct {
				Text        aiText `json:"text"`
				Note        aiText `json:"note"`
				Description aiText `json:"description"`
			}
			if err := json.Unmarshal(item, &object); err == nil {
				value := strings.TrimSpace(string(firstAIText(object.Text, object.Note, object.Description)))
				out = append(out, splitNotes(value)...)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

func splitNotes(value string) []string {
	out := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimLeft(line, "-*•"))
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// flexibleStatusCodes accepts numbers, numeric strings ("200", "200 OK") and
// objects such as {"code": 200}. Duplicates and values outside the valid HTTP
// range are dropped.
func flexibleStatusCodes(items []json.RawMessage) []int {
	out := []int{}
	seen := map[int]bool{}
	for _, item := range items {
		for _, code := range parseStatusCodes(item) {
			if code < 100 || code > 599 || seen[code] {
				continue
			}
			seen[code] = true
			out = append(out, code)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseStatusCodes(raw json.RawMessage) []int {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return []int{int(number)}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return parseStatusCodeText(text)
	}
	var object struct {
		Code       json.RawMessage `json:"code"`
		Status     json.RawMessage `json:"status"`
		StatusCode json.RawMessage `json:"statusCode"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		for _, inner := range []json.RawMessage{object.Code, object.Status, object.StatusCode} {
			if codes := parseStatusCodes(inner); len(codes) > 0 {
				return codes
			}
		}
	}
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err == nil {
		out := []int{}
		for _, inner := range list {
			out = append(out, parseStatusCodes(inner)...)
		}
		return out
	}
	return nil
}

func parseStatusCodeText(value string) []int {
	value = strings.TrimSpace(value)
	for i := 0; i+2 < len(value); i++ {
		if value[i] < '1' || value[i] > '5' {
			continue
		}
		if value[i+1] < '0' || value[i+1] > '9' || value[i+2] < '0' || value[i+2] > '9' {
			continue
		}
		if code, err := strconv.Atoi(value[i : i+3]); err == nil {
			return []int{code}
		}
	}
	return nil
}
