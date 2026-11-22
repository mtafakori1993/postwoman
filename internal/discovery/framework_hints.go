package discovery

import "strings"

// detectFromSource fills gaps when manifests are missing or incomplete.
func detectFromSource(f *fileEntry, frameworks map[string]bool) {
	lower := strings.ToLower(f.Rel)
	switch f.Language {
	case "go":
		switch {
		case strings.Contains(f.Content, "gin-gonic/gin") || strings.Contains(f.Content, "gin.Default()") || strings.Contains(f.Content, "gin.New()"):
			frameworks["Gin"] = true
		case strings.Contains(f.Content, "labstack/echo") || strings.Contains(f.Content, "echo.New()"):
			frameworks["Echo"] = true
		case strings.Contains(f.Content, "go-chi/chi"):
			frameworks["Chi"] = true
		case strings.Contains(f.Content, "gofiber/fiber"):
			frameworks["Fiber"] = true
		case strings.Contains(f.Content, "gorilla/mux"):
			frameworks["Gorilla Mux"] = true
		}
	case "javascript", "typescript":
		switch {
		case strings.Contains(f.Content, "express()") || strings.Contains(f.Content, "from 'express'") || strings.Contains(f.Content, `from "express"`):
			frameworks["Express"] = true
		case strings.Contains(f.Content, "@nestjs/common") || strings.Contains(f.Content, "@Controller("):
			frameworks["NestJS"] = true
		case strings.Contains(f.Content, "fastify(") || strings.Contains(f.Content, "Fastify("):
			frameworks["Fastify"] = true
		case strings.Contains(f.Content, "new Koa()") || strings.Contains(f.Content, "from 'koa'"):
			frameworks["Koa"] = true
		}
		if strings.Contains(lower, "/pages/api/") || strings.Contains(lower, "/app/api/") {
			frameworks["Next.js"] = true
		}
	case "python":
		switch {
		case strings.Contains(f.Content, "from flask") || strings.Contains(f.Content, "Flask(__name__)"):
			frameworks["Flask"] = true
		case strings.Contains(f.Content, "FastAPI(") || strings.Contains(f.Content, "from fastapi"):
			frameworks["FastAPI"] = true
		case strings.Contains(f.Content, "django") || strings.Contains(f.Content, "urlpatterns"):
			frameworks["Django"] = true
		}
	case "php":
		if strings.Contains(f.Content, "Route::") || strings.Contains(f.Content, "Illuminate\\") {
			frameworks["Laravel"] = true
		}
		if strings.Contains(f.Content, "Symfony\\") || strings.Contains(f.Content, "#[Route(") {
			frameworks["Symfony"] = true
		}
	case "ruby":
		if strings.Contains(f.Content, "Rails.application.routes") || strings.Contains(f.Content, "ActionController") {
			frameworks["Rails"] = true
		}
	case "java", "kotlin":
		if strings.Contains(f.Content, "@RestController") || strings.Contains(f.Content, "@RequestMapping") || strings.Contains(f.Content, "@GetMapping") {
			frameworks["Spring Boot"] = true
		}
	case "csharp":
		if strings.Contains(f.Content, "[ApiController]") || strings.Contains(f.Content, "[Route(") || strings.Contains(f.Content, "[HttpGet") {
			frameworks["ASP.NET Core"] = true
		}
	case "rust":
		if strings.Contains(f.Content, "actix_web") {
			frameworks["Actix Web"] = true
		}
		if strings.Contains(f.Content, "rocket::") {
			frameworks["Rocket"] = true
		}
		if strings.Contains(f.Content, "axum::") {
			frameworks["Axum"] = true
		}
	case "elixir":
		if strings.Contains(f.Content, "Phoenix.Router") || strings.Contains(f.Content, "use Phoenix.Router") {
			frameworks["Phoenix"] = true
		}
	}
}

// hintForFile selects the extraction strategy for a file.
func hintForFile(f *fileEntry, frameworks map[string]bool) string {
	switch f.Language {
	case "go":
		for _, candidate := range []struct{ name, hint string }{
			{"Gin", "gin"}, {"Echo", "echo"}, {"Chi", "chi"}, {"Fiber", "fiber"}, {"Gorilla Mux", "gorilla"},
		} {
			if frameworks[candidate.name] {
				return candidate.hint
			}
		}
		return "go"
	case "javascript", "typescript":
		for _, candidate := range []struct{ name, hint string }{
			{"NestJS", "nestjs"}, {"Next.js", "nextjs"}, {"Fastify", "fastify"}, {"Express", "express"}, {"Koa", "koa"},
		} {
			if frameworks[candidate.name] {
				return candidate.hint
			}
		}
		return "js"
	case "python":
		for _, candidate := range []struct{ name, hint string }{
			{"FastAPI", "fastapi"}, {"Flask", "flask"}, {"Django", "django"}, {"Django REST", "django"},
		} {
			if frameworks[candidate.name] {
				return candidate.hint
			}
		}
		return "python"
	case "php":
		if frameworks["Laravel"] {
			return "laravel"
		}
		if frameworks["Symfony"] {
			return "symfony"
		}
		return "php"
	case "ruby":
		return "rails"
	case "java", "kotlin":
		return "spring"
	case "csharp":
		return "aspnet"
	case "rust":
		return "rust"
	case "elixir":
		return "phoenix"
	}
	return ""
}

var baseURLPrefixes = []string{"http://localhost:", "https://localhost:", "http://127.0.0.1:"}

// detectBaseURL looks for a development server URL in configuration or env
// templates. It never reads real secret files.
func detectBaseURL(files map[string]*fileEntry) string {
	for lower, f := range files {
		if !strings.Contains(lower, ".env") && !strings.Contains(lower, "config") && !strings.Contains(lower, "docker-compose") {
			continue
		}
		for _, line := range strings.Split(f.Content, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			for _, prefix := range baseURLPrefixes {
				if idx := strings.Index(trimmed, prefix); idx >= 0 {
					value := strings.TrimSpace(trimmed[idx:])
					value = strings.Trim(value, `"'`)
					if space := strings.IndexAny(value, " \t"); space > 0 {
						value = value[:space]
					}
					return strings.TrimRight(value, "/")
				}
			}
		}
	}
	for lower, f := range files {
		if strings.Contains(lower, "package.json") || strings.Contains(lower, "main.go") || strings.Contains(lower, "app.py") {
			for _, prefix := range baseURLPrefixes {
				if idx := strings.Index(f.Content, prefix); idx >= 0 {
					end := idx
					for end < len(f.Content) && !strings.ContainsRune("\"'\\) \n\t", rune(f.Content[end])) {
						end++
					}
					return strings.TrimRight(f.Content[idx:end], "/")
				}
			}
		}
	}
	return ""
}
