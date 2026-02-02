package parser

import (
	options "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

// EditionsParser implements parsing for editions files
type EditionsParser struct{}

// NewEditionsParser creates a new parser for editions
func NewEditionsParser() *EditionsParser {
	return &EditionsParser{}
}

// ParseHTTPRules extracts HTTP rules from a method descriptor
func (p *EditionsParser) ParseHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
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
func (p *EditionsParser) parseHTTPRule(httpRule *options.HttpRule) HTTPRule {
	return ExtractHTTPRule(httpRule)
}

// ParsePathParams extracts path parameters from a URL pattern
func (p *EditionsParser) ParsePathParams(pattern string) []string {
	return PathParams(pattern)
}

// ConvertPathPattern converts a path pattern to Go format
func (p *EditionsParser) ConvertPathPattern(pattern string) string {
	// For editions, we just return the pattern as is
	return pattern
}
