package discovery

import (
	"go/ast"
	"go/token"
	"strings"
)

func goMethodToken(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return strings.ToUpper(strings.Trim(e.Value, "\"`"))
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok && ident.Name == "http" {
			return strings.ToUpper(strings.TrimPrefix(e.Sel.Name, "Method"))
		}
	}
	return "GET"
}

func stringLiteral(expr ast.Expr, content string, fset *token.FileSet) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		if call, ok := expr.(*ast.CallExpr); ok {
			text := exprText(call, content, fset)
			if strings.Contains(text, "\"") {
				return firstQuoted(text)
			}
		}
		return ""
	}
	return strings.Trim(lit.Value, "\"`")
}

func exprText(expr ast.Expr, content string, fset *token.FileSet) string {
	if expr == nil {
		return ""
	}
	start := fset.Position(expr.Pos()).Offset
	end := fset.Position(expr.End()).Offset
	if start < 0 || end > len(content) || start >= end {
		return ""
	}
	return strings.TrimSpace(content[start:end])
}

func receiverName(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if ident, ok := sel.X.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func isHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE", "CONNECT":
		return true
	}
	return false
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
