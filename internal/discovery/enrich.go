package discovery

import "regexp"

var (
	queryParamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:c\.Query|QueryParam|FormValue|URL\.Query\(\)\.Get)\(\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)req\.query\.([A-Za-z_][\w]*)`),
		regexp.MustCompile(`(?i)request\.args\.get\(\s*['"]([^'"]+)['"]`),
		regexp.MustCompile(`(?i)@RequestParam\(\s*(?:value\s*=\s*)?"([^"]+)"`),
		regexp.MustCompile(`(?i)\[FromQuery\]\s*\w+\s+(\w+)`),
		regexp.MustCompile(`(?i)\$request->query\(\s*['"]([^'"]+)['"]`),
	}
	headerParamPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:c\.GetHeader|Header|GetHeader)\(\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)req\.headers\[['"]([^'"]+)['"]\]`),
		regexp.MustCompile(`(?i)request\.headers\.get\(\s*['"]([^'"]+)['"]`),
		regexp.MustCompile(`(?i)@RequestHeader\(\s*(?:value\s*=\s*)?"([^"]+)"`),
		regexp.MustCompile(`(?i)\[FromHeader\]\s*\w+\s+(\w+)`),
	}
	pathVarPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:c\.Param|PathValue|PathVariable|params)\s*[\(\[]?\s*"([^"]+)"`),
		regexp.MustCompile(`(?i)req\.params\.([A-Za-z_][\w]*)`),
		regexp.MustCompile(`(?i)@PathVariable\(\s*(?:value\s*=\s*)?"([^"]+)"`),
		regexp.MustCompile(`(?i)\[FromRoute\]\s*\w+\s+(\w+)`),
		regexp.MustCompile(`(?i)\$request->route\(\s*['"]([^'"]+)['"]`),
	}
	statusCodePattern  = regexp.MustCompile(`(?i)(?:StatusOK|StatusCreated|StatusAccepted|StatusNoContent|StatusBadRequest|StatusUnauthorized|StatusForbidden|StatusNotFound|StatusConflict|StatusUnprocessableEntity|StatusInternalServerError|status_code\s*=\s*|\.status\(\s*|WriteHeader\(\s*|c\.JSON\(\s*)(\d{3})`)
	statusNamePattern  = regexp.MustCompile(`Status(OK|Created|Accepted|NoContent|BadRequest|Unauthorized|Forbidden|NotFound|Conflict|UnprocessableEntity|InternalServerError)`)
	jsonBodyPattern    = regexp.MustCompile(`(?i)(ShouldBindJSON|BindJSON|json\.NewDecoder|BodyParser|ShouldBind|Bind\(|get_json\(\)|request\.json|req\.body|request\.body|@RequestBody|\[FromBody\]|->validate\(|->all\(\)|->input\()`) //nolint:lll
	goStructPattern    = regexp.MustCompile(`(?m)type\s+([A-Za-z_]\w*)\s+struct\s*\{`)
	tsInterfacePattern = regexp.MustCompile(`(?m)(?:interface|type)\s+([A-Za-z_]\w*)\s*(?:=\s*)?\{`)
	globalAuthPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:router|app|server|group|api)\.(?:Use|use)\(\s*[^)]*(?:auth|jwt|middleware|guard|passport)[^)]*\)`),
		regexp.MustCompile(`(?i)@PreAuthorize|@Secured|@RolesAllowed|\[Authorize\]`),
	}
)
