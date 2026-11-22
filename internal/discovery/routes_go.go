package discovery

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"Postwoman/internal/model"
)

var goMethodByCall = map[string]string{
	"GET": "GET", "Get": "GET", "POST": "POST", "Post": "POST",
	"PUT": "PUT", "Put": "PUT", "PATCH": "PATCH", "Patch": "PATCH",
	"DELETE": "DELETE", "Delete": "DELETE", "HEAD": "HEAD", "Head": "HEAD",
	"OPTIONS": "OPTIONS", "Options": "OPTIONS", "Any": "ANY", "All": "ANY",
}

// extractGoRoutes parses Go source with go/parser and resolves router group
// prefixes, HTTP methods, handlers and middleware. It understands Gin, Echo,
// Chi, Fiber, Gorilla Mux, httprouter and net/http registration styles.
func extractGoRoutes(a *analyzer, f *fileEntry) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, f.Rel, f.Content, parser.ParseComments)
	if err != nil {
		return
	}
	prefixes := collectGoGroupPrefixes(file, fset, f.Content)
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		name := sel.Sel.Name
		prefix := resolveGoPrefix(sel.X, prefixes)
		line := fset.Position(call.Pos()).Line
		switch {
		case goMethodByCall[name] != "":
			handleGoMethodCall(a, f, call, name, prefix, line)
		case name == "Handle" || name == "HandleFunc":
			handleGoHandleCall(a, f, call, name, prefix, line)
		case name == "Method":
			handleGoMethodCall(a, f, call, "Method", prefix, line)
		}
		return true
	})
}

func collectGoGroupPrefixes(file *ast.File, fset *token.FileSet, content string) map[string]string {
	prefixes := map[string]string{}
	for pass := 0; pass < 2; pass++ {
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			ident, ok := assign.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "Group" && sel.Sel.Name != "Route" && sel.Sel.Name != "Prefix" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			path := stringLiteral(call.Args[0], content, fset)
			if path == "" {
				return true
			}
			prefixes[ident.Name] = joinRoutePaths(resolveGoPrefix(sel.X, prefixes), path)
			return true
		})
	}
	return prefixes
}

func resolveGoPrefix(expr ast.Expr, prefixes map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return prefixes[e.Name]
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok || len(e.Args) == 0 {
			return ""
		}
		if sel.Sel.Name == "Group" || sel.Sel.Name == "Route" || sel.Sel.Name == "Prefix" {
			if lit, ok := e.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return joinRoutePaths(resolveGoPrefix(sel.X, prefixes), strings.Trim(lit.Value, "\"`"))
			}
		}
	}
	return ""
}

func handleGoMethodCall(a *analyzer, f *fileEntry, call *ast.CallExpr, name, prefix string, line int) {
	args := call.Args
	if len(args) < 2 {
		return
	}
	method := goMethodByCall[name]
	var path, handler string
	if name == "Method" {
		method = goMethodToken(args[0])
		path = stringLiteral(args[1], f.Content, token.NewFileSet())
		if len(args) > 2 {
			handler = exprText(args[2], f.Content, token.NewFileSet())
		}
	} else {
		path = stringLiteral(args[0], f.Content, token.NewFileSet())
		handler = exprText(args[1], f.Content, token.NewFileSet())
	}
	if path == "" {
		return
	}
	handlerName, handlerFile, handlerLine := a.resolveHandler(handler, f.Rel)
	ep := model.DiscoveredEndpoint{
		Method: method, Path: joinRoutePaths(prefix, path), SourceFile: f.Rel, Line: line,
		Handler: handlerName, HandlerFile: handlerFile, Controller: receiverName(call),
		Framework: "Go", Confidence: 80, Origin: "static",
	}
	if handlerLine > 0 {
		ep.Notes = append(ep.Notes, "handler declared at line "+itoa(handlerLine))
	}
	applyGoMiddleware(&ep, call, f.Content)
	a.addEndpoint(ep)
}

func handleGoHandleCall(a *analyzer, f *fileEntry, call *ast.CallExpr, name, prefix string, line int) {
	if len(call.Args) < 2 {
		return
	}
	path := ""
	method := "GET"
	first := call.Args[0]
	if text := exprText(first, f.Content, token.NewFileSet()); strings.HasPrefix(strings.TrimSpace(text), "http.Method") {
		method = strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(text), "http.Method"))
	}
	if lit := stringLiteral(first, f.Content, token.NewFileSet()); lit != "" {
		fields := strings.Fields(lit)
		if len(fields) >= 2 && isHTTPMethod(fields[0]) {
			method = strings.ToUpper(fields[0])
			path = fields[1]
		} else {
			path = lit
		}
	}
	if path == "" {
		return
	}
	handler := exprText(call.Args[1], f.Content, token.NewFileSet())
	handlerName, handlerFile, _ := a.resolveHandler(handler, f.Rel)
	ep := model.DiscoveredEndpoint{
		Method: method, Path: joinRoutePaths(prefix, path), SourceFile: f.Rel, Line: line,
		Handler: handlerName, HandlerFile: handlerFile, Controller: receiverName(call),
		Framework: "Go", Confidence: 65, Origin: "static",
	}
	if name == "HandleFunc" && method == "GET" && !strings.Contains(strings.ToLower(handler), "get") {
		ep.Notes = append(ep.Notes, "HTTP method not explicit in registration; defaulted to GET")
		ep.Warnings = append(ep.Warnings, "method inferred from registration style")
	}
	applyGoMiddleware(&ep, call, f.Content)
	a.addEndpoint(ep)
}

func applyGoMiddleware(ep *model.DiscoveredEndpoint, call *ast.CallExpr, content string) {
	for _, arg := range call.Args {
		text := exprText(arg, content, token.NewFileSet())
		if text == "" || strings.HasPrefix(text, "\"") {
			continue
		}
		if hasAuthKeyword(text) {
			ep.AuthRequired = true
			ep.AuthType = "middleware"
			ep.AuthNotes = "middleware: " + text
			ep.Confidence = minInt(ep.Confidence+5, 95)
			return
		}
	}
}
