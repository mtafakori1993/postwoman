package discovery

import (
	"path/filepath"
	"strings"
)

func categoryFor(lower, relLower string) string {
	if isSchemaName(lower) || strings.Contains(relLower, "graphql") || strings.HasSuffix(lower, ".proto") || strings.HasSuffix(lower, ".raml") {
		return CategorySchema
	}
	if isTestPath(relLower) {
		return CategoryTest
	}
	if strings.HasSuffix(lower, "routes.rb") || strings.HasSuffix(lower, "urls.py") || strings.HasSuffix(lower, "web.php") ||
		strings.HasSuffix(lower, "api.php") || strings.Contains(lower, "route") {
		return CategoryRoute
	}
	if isConfigName(lower) || strings.Contains(relLower, "/config/") {
		return CategoryConfig
	}
	for _, hint := range []string{"/routes/", "/route/", "router.", "/routers/", "/controllers/", "/controller/",
		"/handlers/", "/handler/", "/endpoints/", "/endpoint/", "/api/", "/views/", "/resolvers/"} {
		if strings.Contains(relLower, hint) {
			return CategoryRoute
		}
	}
	for _, hint := range []string{"/models/", "/model/", "/schemas/", "/schema/", "/dtos/", "/dto/",
		"/entities/", "/entity/", "/serializers/", "/serializer/", "/validators/", "/validator/",
		"/types/", "/interfaces/", "/forms/", "/requests/", "/resources/"} {
		if strings.Contains(relLower, hint) {
			return CategoryModel
		}
	}
	return CategorySource
}

func isTestPath(relLower string) bool {
	for _, marker := range []string{"/tests/", "/test/", "/__tests__/", "/spec/", "/cypress/", "/e2e/", "/testdata/"} {
		if strings.Contains(relLower, marker) {
			return true
		}
	}
	base := filepath.Base(relLower)
	return strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.HasPrefix(base, "test_") || base == "conftest.py"
}

func isSafeEnvTemplate(lower string) bool {
	for _, suffix := range []string{".example", ".sample", ".template", ".dist", ".defaults"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}
