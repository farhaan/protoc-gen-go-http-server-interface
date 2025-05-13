package httpinterface

import (
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

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

// GetHTTPRules is the exported variable for HTTP rule extraction
var GetHTTPRules HTTPRuleExtractor = getHTTPRules

// GetPathParams is the exported variable for path parameter extraction
var GetPathParams PathParamExtractor = getPathParams

// ConvertPathPattern is the exported variable for path pattern conversion
var ConvertPathPattern PathPatternConverter = convertPathPattern

// Current file descriptor being processed
var currentFileDescriptor *descriptor.FileDescriptorProto

// SetFileDescriptor sets the current file descriptor for processing
func SetFileDescriptor(file *descriptor.FileDescriptorProto) {
	currentFileDescriptor = file
}

func ResetParserState() {
	currentFileDescriptor = nil
}

// Implementation for getHTTPRules that uses the parser system
func getHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
	// If we don't have a current file descriptor, return empty rules
	if currentFileDescriptor == nil {
		return []HTTPRule{}
	}

	// Create a parser for the current file
	p := parser.CreateParser(currentFileDescriptor)

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

// Implementation for getPathParams that uses the parser system
func getPathParams(pattern string) []string {
	// If we don't have a current file descriptor, return empty params
	if currentFileDescriptor == nil {
		return []string{}
	}

	// Create a parser for the current file
	p := parser.CreateParser(currentFileDescriptor)

	// Get path params from the parser
	return p.ParsePathParams(pattern)
}

// Implementation for convertPathPattern that uses the parser system
func convertPathPattern(pattern string) string {
	// If we don't have a current file descriptor, return pattern unchanged
	if currentFileDescriptor == nil {
		return pattern
	}

	// Create a parser for the current file
	p := parser.CreateParser(currentFileDescriptor)

	// Convert pattern using the parser
	return p.ConvertPathPattern(pattern)
}
