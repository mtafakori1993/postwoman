package discovery

import (
	"path"
	"sort"
	"strings"
)

// projectProfile is the detected shape of a project.
type projectProfile struct {
	Languages  []string
	Frameworks []string
	BaseURL    string
	// routeHints maps a file path to the framework that governs it, so the
	// analyser can choose the right extraction strategy.
	routeHints map[string]string
	// modulePrefix is the Go module path from go.mod (used for handler lookups).
	modulePrefix string
}

// detectProfile inspects manifests and source files to determine languages and
// frameworks. It is deterministic and local; no AI call is involved.
func detectProfile(files []*fileEntry) *projectProfile {
	profile := &projectProfile{routeHints: map[string]string{}}
	langSet := map[string]bool{}
	frameworkSet := map[string]bool{}
	contentByPath := map[string]*fileEntry{}
	for _, f := range files {
		if f.Language != "" {
			langSet[f.Language] = true
		}
		contentByPath[strings.ToLower(f.Rel)] = f
	}

	for lower, f := range contentByPath {
		base := path.Base(lower)
		content := strings.ToLower(f.Content)
		switch base {
		case "package.json":
			for _, dep := range []struct{ key, framework string }{
				{"\"express\"", "Express"}, {"\"@nestjs/core\"", "NestJS"}, {"\"fastify\"", "Fastify"},
				{"\"koa\"", "Koa"}, {"\"@hapi/hapi\"", "Hapi"}, {"\"next\"", "Next.js"},
				{"\"nuxt\"", "Nuxt"}, {"\"@sveltejs/kit\"", "SvelteKit"}, {"\"hono\"", "Hono"},
				{"\"@trpc/server\"", "tRPC"}, {"\"apollo-server\"", "Apollo"}, {"\"@apollo/server\"", "Apollo"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "go.mod":
			for _, dep := range []struct{ key, framework string }{
				{"github.com/gin-gonic/gin", "Gin"}, {"github.com/labstack/echo", "Echo"},
				{"github.com/go-chi/chi", "Chi"}, {"github.com/gofiber/fiber", "Fiber"},
				{"github.com/gorilla/mux", "Gorilla Mux"}, {"github.com/julienschmidt/httprouter", "httprouter"},
				{"github.com/beego/beego", "Beego"}, {"github.com/emicklei/go-restful", "go-restful"},
				{"github.com/99designs/gqlgen", "gqlgen"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
			if mod := goModulePath(f.Content); mod != "" {
				profile.modulePrefix = mod
			}
		case "requirements.txt", "pyproject.toml", "pipfile", "setup.py":
			for _, dep := range []struct{ key, framework string }{
				{"flask", "Flask"}, {"fastapi", "FastAPI"}, {"django", "Django"},
				{"djangorestframework", "Django REST"}, {"starlette", "Starlette"},
				{"tornado", "Tornado"}, {"aiohttp", "aiohttp"}, {"sanic", "Sanic"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "composer.json":
			for _, dep := range []struct{ key, framework string }{
				{"laravel/framework", "Laravel"}, {"symfony/", "Symfony"},
				{"slim/slim", "Slim"}, {"cakephp/cakephp", "CakePHP"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "gemfile":
			for _, dep := range []struct{ key, framework string }{
				{"'rails'", "Rails"}, {"\"rails\"", "Rails"}, {"sinatra", "Sinatra"}, {"grape", "Grape"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "pom.xml", "build.gradle", "build.gradle.kts":
			for _, dep := range []struct{ key, framework string }{
				{"spring-boot", "Spring Boot"}, {"spring-web", "Spring"}, {"quarkus", "Quarkus"},
				{"micronaut", "Micronaut"}, {"ktor", "Ktor"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "cargo.toml":
			for _, dep := range []struct{ key, framework string }{
				{"actix-web", "Actix Web"}, {"rocket", "Rocket"}, {"axum", "Axum"}, {"warp", "Warp"},
			} {
				if strings.Contains(content, dep.key) {
					frameworkSet[dep.framework] = true
				}
			}
		case "mix.exs":
			if strings.Contains(content, "phoenix") {
				frameworkSet["Phoenix"] = true
			}
		}
		if strings.HasSuffix(base, ".csproj") {
			if strings.Contains(content, "microsoft.aspnetcore") || strings.Contains(content, "microsoft.net.sdk.web") {
				frameworkSet["ASP.NET Core"] = true
			}
		}
	}

	for _, f := range files {
		detectFromSource(f, frameworkSet)
		profile.routeHints[f.Rel] = hintForFile(f, frameworkSet)
	}

	profile.Languages = sortedKeys(langSet)
	profile.Frameworks = sortedKeys(frameworkSet)
	profile.BaseURL = detectBaseURL(contentByPath)
	return profile
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		if key != "" {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func goModulePath(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
