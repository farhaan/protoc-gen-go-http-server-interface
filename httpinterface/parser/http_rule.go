package parser

import (
	"net/http"
	"regexp"

	options "google.golang.org/genproto/googleapis/api/annotations"
)

// pathParamRegex matches {param} in URL patterns - unexported implementation detail
var pathParamRegex = regexp.MustCompile(`\{([^/{}]+)\}`)

// PathParams extracts path parameters from a URL pattern like "/users/{id}"
// Returns empty slice (not nil) when no params found - this is the API contract.
func PathParams(pattern string) []string {
	params := []string{}
	for _, match := range pathParamRegex.FindAllStringSubmatch(pattern, -1) {
		if len(match) >= 2 {
			params = append(params, match[1])
		}
	}
	return params
}

// ExtractHTTPRule converts a proto HttpRule to our HTTPRule type
func ExtractHTTPRule(httpRule *options.HttpRule) HTTPRule {
	if httpRule == nil {
		return HTTPRule{PathParams: []string{}}
	}

	rule := HTTPRule{Body: httpRule.Body}

	switch p := httpRule.Pattern.(type) {
	case *options.HttpRule_Get:
		rule.Method, rule.Pattern = http.MethodGet, p.Get
	case *options.HttpRule_Post:
		rule.Method, rule.Pattern = http.MethodPost, p.Post
	case *options.HttpRule_Put:
		rule.Method, rule.Pattern = http.MethodPut, p.Put
	case *options.HttpRule_Delete:
		rule.Method, rule.Pattern = http.MethodDelete, p.Delete
	case *options.HttpRule_Patch:
		rule.Method, rule.Pattern = http.MethodPatch, p.Patch
	case *options.HttpRule_Custom:
		if p.Custom != nil {
			rule.Method, rule.Pattern = p.Custom.Kind, p.Custom.Path
		}
	}

	rule.PathParams = PathParams(rule.Pattern)
	return rule
}
