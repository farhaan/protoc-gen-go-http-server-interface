package httpinterface

import (
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

// HTTPRuleExtractor is the function type for extracting HTTP rules
type HTTPRuleExtractor func(method *descriptor.MethodDescriptorProto) []parser.HTTPRule

// PathParamExtractor is the function type for extracting path parameters
type PathParamExtractor func(pattern string) []string

// PathPatternConverter is the function type for converting path patterns
type PathPatternConverter func(pattern string) string

// extractHTTPRules extracts HTTP rules from a method descriptor.
func extractHTTPRules(method *descriptor.MethodDescriptorProto) []parser.HTTPRule {
	rules := []parser.HTTPRule{}

	if method.Options != nil {
		v := proto.GetExtension(method.Options, options.E_Http)
		httpRule, ok := v.(*options.HttpRule)
		if ok && httpRule != nil {
			// Add the main rule
			if rule := parser.ExtractHTTPRule(httpRule); rule.Method != "" {
				rules = append(rules, rule)
			}

			// Add additional bindings
			for _, binding := range httpRule.AdditionalBindings {
				if rule := parser.ExtractHTTPRule(binding); rule.Method != "" {
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules
}

// extractPathParams extracts path parameters from a URL pattern.
func extractPathParams(pattern string) []string {
	return parser.PathParams(pattern)
}

// convertPathPattern converts a path pattern to Go format
func convertPathPattern(pattern string) string {
	return pattern
}

// CreateHTTPRuleExtractorForFile creates an HTTP rule extractor for a specific file
// This allows for editions support while maintaining dependency injection
func CreateHTTPRuleExtractorForFile(file *descriptor.FileDescriptorProto) HTTPRuleExtractor {
	return func(method *descriptor.MethodDescriptorProto) []parser.HTTPRule {
		p := parser.CreateParser(file)
		return p.ParseHTTPRules(method)
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
