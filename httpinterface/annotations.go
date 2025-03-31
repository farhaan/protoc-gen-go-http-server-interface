package httpinterface

import (
	"regexp"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
)

// HTTPRuleExtractor is the function type for extracting HTTP rules
type HTTPRuleExtractor func(method *descriptor.MethodDescriptorProto) []HTTPRule

// parseMethodHTTPRules extracts HTTP rules from a method descriptor
func parseMethodHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
	rules := []HTTPRule{}

	if method.Options != nil {
		v := proto.GetExtension(method.Options, options.E_Http)
		httpRule := v.(*options.HttpRule)
		if httpRule != nil {
			// Add the main rule
			rule := parseHTTPRule(httpRule)
			if rule.Method != "" {
				rules = append(rules, rule)
			}

			// Add additional bindings
			for _, binding := range httpRule.AdditionalBindings {
				rule := parseHTTPRule(binding)
				if rule.Method != "" {
					rules = append(rules, rule)
				}
			}

		}
	}

	return rules
}

// GetHTTPRules is the exported variable for HTTP rule extraction
// This allows the function to be replaced in tests
var GetHTTPRules HTTPRuleExtractor = parseMethodHTTPRules

// parseHTTPRule extracts method, pattern, and body from an HttpRule
func parseHTTPRule(httpRule *options.HttpRule) HTTPRule {
	rule := HTTPRule{
		Body: httpRule.Body,
	}

	switch pattern := httpRule.Pattern.(type) {
	case *options.HttpRule_Get:
		rule.Method = "GET"
		rule.Pattern = pattern.Get
	case *options.HttpRule_Post:
		rule.Method = "POST"
		rule.Pattern = pattern.Post
	case *options.HttpRule_Put:
		rule.Method = "PUT"
		rule.Pattern = pattern.Put
	case *options.HttpRule_Delete:
		rule.Method = "DELETE"
		rule.Pattern = pattern.Delete
	case *options.HttpRule_Patch:
		rule.Method = "PATCH"
		rule.Pattern = pattern.Patch
	case *options.HttpRule_Custom:
		rule.Method = pattern.Custom.Kind
		rule.Pattern = pattern.Custom.Path
	}

	return rule
}

// PathParamExtractor is the function type for extracting path parameters
type PathParamExtractor func(pattern string) []string

// parsePathParams extracts path parameters from a URL pattern
func parsePathParams(pattern string) []string {
	params := []string{}
	re := regexp.MustCompile(`\{([^/{}]+)\}`)
	matches := re.FindAllStringSubmatch(pattern, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			params = append(params, match[1])
		}
	}

	return params
}

// GetPathParams is the exported variable for path parameter extraction
// This allows the function to be replaced in tests
var GetPathParams PathParamExtractor = parsePathParams

// PathPatternConverter is the function type for converting path patterns
type PathPatternConverter func(pattern string) string

// convertPathPattern converts a path pattern to Go format
func convertPathPattern(pattern string) string {
	return pattern
}

// ConvertPathPattern is the exported variable for path pattern conversion
// This allows the function to be replaced in tests
var ConvertPathPattern PathPatternConverter = convertPathPattern
