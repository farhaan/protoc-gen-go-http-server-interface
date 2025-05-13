package parser

import (
	"regexp"

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
		httpRule := v.(*options.HttpRule)
		if httpRule != nil {
			// Add the main rule
			rule := p.parseHTTPRule(httpRule)
			if rule.Method != "" {
				rule.PathParams = p.ParsePathParams(rule.Pattern)
				rules = append(rules, rule)
			}

			// Add additional bindings
			for _, binding := range httpRule.AdditionalBindings {
				rule := p.parseHTTPRule(binding)
				if rule.Method != "" {
					rule.PathParams = p.ParsePathParams(rule.Pattern)
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules
}

// parseHTTPRule extracts method, pattern, and body from an HttpRule
func (p *Proto3Parser) parseHTTPRule(httpRule *options.HttpRule) HTTPRule {
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

	rule.PathParams = p.ParsePathParams(rule.Pattern)
	return rule
}

// ParsePathParams extracts path parameters from a URL pattern
func (p *Proto3Parser) ParsePathParams(pattern string) []string {
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

// ConvertPathPattern converts a path pattern to Go format
func (p *Proto3Parser) ConvertPathPattern(pattern string) string {
	// For proto3, we just return the pattern as is
	return pattern
}
