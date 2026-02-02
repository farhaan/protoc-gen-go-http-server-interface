package httpinterface

import (
	"regexp"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

var pathParamRegex = regexp.MustCompile(`\{([^/{}]+)\}`)

// HTTPRule represents an HTTP binding from annotations.
type HTTPRule struct {
	Method     string
	Pattern    string
	Body       string
	PathParams []string
}

// HTTPRuleExtractor is the function type for extracting HTTP rules
type HTTPRuleExtractor func(method *descriptor.MethodDescriptorProto) []HTTPRule

// PathParamExtractor is the function type for extracting path parameters
type PathParamExtractor func(pattern string) []string

// PathPatternConverter is the function type for converting path patterns
type PathPatternConverter func(pattern string) string

// parseMethodHTTPRules extracts HTTP rules from a method descriptor using legacy approach
func parseMethodHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
	rules := []HTTPRule{}

	if method.Options != nil {
		v := proto.GetExtension(method.Options, options.E_Http)
		httpRule, ok := v.(*options.HttpRule)
		if ok && httpRule != nil {
			// Add the main rule
			rule := parseHTTPRule(httpRule)
			if rule.Method != "" {
				rule.PathParams = parsePathParams(rule.Pattern)
				rules = append(rules, rule)
			}

			// Add additional bindings
			for _, binding := range httpRule.AdditionalBindings {
				rule := parseHTTPRule(binding)
				if rule.Method != "" {
					rule.PathParams = parsePathParams(rule.Pattern)
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules
}

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
		if pattern.Custom != nil {
			rule.Method = pattern.Custom.Kind
			rule.Pattern = pattern.Custom.Path
		}
	}

	return rule
}

// parsePathParams extracts path parameters from a URL pattern using regex
func parsePathParams(pattern string) []string {
	params := []string{}
	matches := pathParamRegex.FindAllStringSubmatch(pattern, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			params = append(params, match[1])
		}
	}

	return params
}

// convertPathPattern converts a path pattern to Go format
func convertPathPattern(pattern string) string {
	return pattern
}

// CreateHTTPRuleExtractorForFile creates an HTTP rule extractor for a specific file
// This allows for editions support while maintaining dependency injection
func CreateHTTPRuleExtractorForFile(file *descriptor.FileDescriptorProto) HTTPRuleExtractor {
	return func(method *descriptor.MethodDescriptorProto) []HTTPRule {
		// Create a parser for the current file
		p := parser.CreateParser(file)

		// Get rules from the parser
		parsingRules := p.ParseHTTPRules(method)

		// Convert to our HTTPRule type
		rules := make([]HTTPRule, len(parsingRules))
		for i, rule := range parsingRules {
			rules[i] = HTTPRule{
				Method:     rule.Method,
				Pattern:    rule.Pattern,
				Body:       rule.Body,
				PathParams: rule.PathParams,
			}
		}

		return rules
	}
}

// CreatePathParamExtractorForFile creates a path parameter extractor for a specific file
func CreatePathParamExtractorForFile(file *descriptor.FileDescriptorProto) PathParamExtractor {
	return func(pattern string) []string {
		// Create a parser for the current file
		p := parser.CreateParser(file)

		// Get path params from the parser
		return p.ParsePathParams(pattern)
	}
}

// CreatePathPatternConverterForFile creates a path pattern converter for a specific file
func CreatePathPatternConverterForFile(file *descriptor.FileDescriptorProto) PathPatternConverter {
	return func(pattern string) string {
		// Create a parser for the current file
		p := parser.CreateParser(file)

		// Convert pattern using the parser
		return p.ConvertPathPattern(pattern)
	}
}
