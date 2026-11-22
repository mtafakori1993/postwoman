package discovery

import (
	"strings"

	"Postwoman/internal/model"
)

// enrichAuth applies file-level and controller-level authentication hints.
func (a *analyzer) enrichAuth() {
	for _, key := range a.order {
		ep := a.endpoints[key]
		if ep == nil || ep.AuthRequired {
			continue
		}
		file := a.byRel[ep.SourceFile]
		if file == nil {
			continue
		}
		for _, pattern := range globalAuthPatterns {
			if pattern.MatchString(file.Content) {
				ep.AuthRequired = true
				ep.AuthType = "middleware"
				ep.AuthNotes = "global or controller-level authentication middleware"
				break
			}
		}
	}
}

// resolveRelationships links endpoints that belong to the same resource so the
// UI can show related operations.
func (a *analyzer) resolveRelationships() {
	byResource := map[string][]*model.DiscoveredEndpoint{}
	for _, key := range a.order {
		ep := a.endpoints[key]
		if ep == nil {
			continue
		}
		resource := firstPathSegment(ep.Path)
		if resource == "" {
			continue
		}
		byResource[resource] = append(byResource[resource], ep)
	}
	for _, group := range byResource {
		if len(group) < 2 || len(group) > 12 {
			continue
		}
		for _, ep := range group {
			for _, other := range group {
				if ep.ID == other.ID {
					continue
				}
				ep.Related = appendUniqueStrings(ep.Related, []string{other.ID})
				if len(ep.Related) >= 6 {
					break
				}
			}
		}
	}
}

func firstPathSegment(path string) string {
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if segment == "" || strings.HasPrefix(segment, ":") || segment == "api" || segment == "v1" || segment == "v2" {
			continue
		}
		return segment
	}
	return ""
}
