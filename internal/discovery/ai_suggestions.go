package discovery

import (
	"encoding/json"
	"strings"
)

func (a *analyzer) applySuggestions(suggestions []aiSuggestion) {
	for _, suggestion := range suggestions {
		ep := a.endpoints[suggestion.ID]
		if ep == nil {
			continue
		}
		if strings.TrimSpace(suggestion.Name) != "" && ep.Name == "" {
			ep.Name = suggestion.Name
		}
		if strings.TrimSpace(suggestion.Description) != "" && ep.Description == "" {
			ep.Description = suggestion.Description
		}
		if suggestion.AuthRequired != nil && *suggestion.AuthRequired && !ep.AuthRequired {
			ep.AuthRequired = true
			ep.AuthType = suggestion.AuthType
			ep.AuthNotes = "detected by AI reasoning"
		}
		if suggestion.RequestBody != nil && ep.RequestBody == "" {
			encoded, err := json.MarshalIndent(suggestion.RequestBody, "", "  ")
			if err == nil && string(encoded) != "null" {
				ep.RequestBody = string(encoded)
				if ep.ContentType == "" {
					ep.ContentType = "application/json"
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
