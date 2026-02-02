package parser

import (
	"strings"
	"testing"
)

// FuzzParsePathParams tests ParsePathParams with random inputs
func FuzzParsePathParams(f *testing.F) {
	// Seed corpus with known patterns
	seeds := []string{
		"",
		"/",
		"/v1/users",
		"/v1/users/{id}",
		"/v1/users/{user_id}/posts/{post_id}",
		"/v1/{org}/users/{user.id}",
		"/{a}/{b}/{c}/{d}/{e}",
		"/v1/users/{id}/comments/{comment_id}/replies/{reply_id}",
		"{{nested}}",
		"{unclosed",
		"unopened}",
		"/path/with spaces/{param}",
		"/path/with/special/chars!@#$%/{id}",
		"/unicode/路径/{参数}",
		strings.Repeat("/{x}", 100), // long pattern
		"/v1/users/{" + strings.Repeat("a", 1000) + "}", // long param name
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		for _, p := range parsers {
			// Property 1: ParsePathParams should never panic
			params := p.parser.ParsePathParams(pattern)

			// Property 2: Result should never be nil
			if params == nil {
				t.Errorf("%s: ParsePathParams(%q) returned nil, expected empty slice", p.name, pattern)
			}

			// Property 3: Number of params should match number of {param} patterns
			expectedCount := strings.Count(pattern, "{") - strings.Count(pattern, "{{")
			// This is approximate - malformed patterns may differ
			if len(params) > expectedCount {
				t.Errorf("%s: ParsePathParams(%q) returned %d params, but pattern has at most %d opening braces",
					p.name, pattern, len(params), expectedCount)
			}

			// Property 4: All returned params should be non-empty
			// Note: We don't check UTF-8 validity since proto files are always UTF-8
			// and invalid bytes would be rejected by protoc before reaching our plugin
			for i, param := range params {
				if param == "" {
					t.Errorf("%s: ParsePathParams(%q) returned empty param at index %d", p.name, pattern, i)
				}
			}

			// Property 5: Params should not contain braces
			for i, param := range params {
				if strings.ContainsAny(param, "{}") {
					t.Errorf("%s: ParsePathParams(%q) returned param with braces at index %d: %q", p.name, pattern, i, param)
				}
			}
		}
	})
}

// FuzzConvertPathPattern tests ConvertPathPattern with random inputs
func FuzzConvertPathPattern(f *testing.F) {
	seeds := []string{
		"",
		"/",
		"/v1/users/{id}",
		"/v1/users/{user_id}/posts/{post_id}",
		"/path/with spaces",
		"/unicode/路径/{参数}",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	parsers := []struct {
		name   string
		parser interface{ ConvertPathPattern(string) string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	f.Fuzz(func(t *testing.T, pattern string) {
		for _, p := range parsers {
			// Property 1: ConvertPathPattern should never panic
			result := p.parser.ConvertPathPattern(pattern)

			// Property 2: Result should not be longer than input (no explosion)
			// This is a sanity check - current implementation returns as-is
			if len(result) > len(pattern)*2 {
				t.Errorf("%s: ConvertPathPattern(%q) result length %d is unexpectedly large",
					p.name, pattern, len(result))
			}
		}
	})
}
