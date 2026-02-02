package parser

import (
	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

// Proto3Parser implements parsing for proto3 files
type Proto3Parser struct{}

// NewProto3Parser creates a new parser for proto3
func NewProto3Parser() *Proto3Parser {
	return &Proto3Parser{}
}

// ParseHTTPRules extracts HTTP rules from a method descriptor
func (p *Proto3Parser) ParseHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
	rules := []HTTPRule{}

	if method.Options != nil {
		v := proto.GetExtension(method.Options, options.E_Http)
		httpRule, ok := v.(*options.HttpRule)
		if ok && httpRule != nil {
			// Add the main rule
			rule := p.parseHTTPRule(httpRule)
			if rule.Method != "" {
				rules = append(rules, rule)
			}

			// Add additional bindings
			for _, binding := range httpRule.AdditionalBindings {
				rule := p.parseHTTPRule(binding)
				if rule.Method != "" {
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules
}

// parseHTTPRule extracts method, pattern, and body from an HttpRule
func (p *Proto3Parser) parseHTTPRule(httpRule *options.HttpRule) HTTPRule {
	return ExtractHTTPRule(httpRule)
}

// ParsePathParams extracts path parameters from a URL pattern
func (p *Proto3Parser) ParsePathParams(pattern string) []string {
	return PathParams(pattern)
}

// ConvertPathPattern converts a path pattern to Go format
func (p *Proto3Parser) ConvertPathPattern(pattern string) string {
	// For proto3, we just return the pattern as is
	return pattern
}
