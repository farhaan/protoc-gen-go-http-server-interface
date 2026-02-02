package parser

import (
	"regexp"
	"strings"
	"testing"
	"testing/quick"
)

// TestProperty_ParsePathParamsNeverReturnsNil verifies ParsePathParams always returns non-nil slice
func TestProperty_ParsePathParamsNeverReturnsNil(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		p := p // capture
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			f := func(pattern string) bool {
				result := p.parser.ParsePathParams(pattern)
				return result != nil
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
				t.Errorf("Property violated: %v", err)
			}
		})
	}
}

// TestProperty_ParsePathParamsCountMatchesBraces verifies param count <= brace pairs
func TestProperty_ParsePathParamsCountMatchesBraces(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		p := p
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			f := func(pattern string) bool {
				params := p.parser.ParsePathParams(pattern)
				// Count matched brace pairs using regex
				re := regexp.MustCompile(`\{[^/{}]+\}`)
				matches := re.FindAllString(pattern, -1)
				return len(params) <= len(matches)
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
				t.Errorf("Property violated: %v", err)
			}
		})
	}
}

// TestProperty_ParsePathParamsIdempotent verifies calling twice gives same result
func TestProperty_ParsePathParamsIdempotent(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		p := p
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			f := func(pattern string) bool {
				result1 := p.parser.ParsePathParams(pattern)
				result2 := p.parser.ParsePathParams(pattern)

				if len(result1) != len(result2) {
					return false
				}
				for i := range result1 {
					if result1[i] != result2[i] {
						return false
					}
				}
				return true
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
				t.Errorf("Property violated: %v", err)
			}
		})
	}
}

// TestProperty_ParsePathParamsParamsAreSubstrings verifies extracted params exist in pattern
func TestProperty_ParsePathParamsParamsAreSubstrings(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		p := p
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			f := func(pattern string) bool {
				params := p.parser.ParsePathParams(pattern)
				for _, param := range params {
					// Each param should appear in pattern wrapped in braces
					if !strings.Contains(pattern, "{"+param+"}") {
						return false
					}
				}
				return true
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
				t.Errorf("Property violated: %v", err)
			}
		})
	}
}

// TestProperty_AllParsersProduceSameResult verifies all parsers are consistent
func TestProperty_AllParsersProduceSameResult(t *testing.T) {
	t.Parallel()

	proto3 := NewProto3Parser()
	proto2 := NewProto2Parser()
	editions := NewEditionsParser()

	f := func(pattern string) bool {
		result3 := proto3.ParsePathParams(pattern)
		result2 := proto2.ParsePathParams(pattern)
		resultE := editions.ParsePathParams(pattern)

		// All should have same length
		if len(result3) != len(result2) || len(result2) != len(resultE) {
			return false
		}

		// All should have same values
		for i := range result3 {
			if result3[i] != result2[i] || result2[i] != resultE[i] {
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("Parsers produced different results: %v", err)
	}
}

// TestProperty_ConvertPathPatternIdempotent verifies ConvertPathPattern is idempotent
func TestProperty_ConvertPathPatternIdempotent(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ConvertPathPattern(string) string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		p := p
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			f := func(pattern string) bool {
				result1 := p.parser.ConvertPathPattern(pattern)
				result2 := p.parser.ConvertPathPattern(result1)
				return result1 == result2
			}

			if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
				t.Errorf("Property violated: %v", err)
			}
		})
	}
}

// TestProperty_EmptyPatternEmptyParams verifies empty pattern gives empty params
func TestProperty_EmptyPatternEmptyParams(t *testing.T) {
	t.Parallel()

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, p := range parsers {
		t.Run(p.name, func(t *testing.T) {
			result := p.parser.ParsePathParams("")
			if len(result) != 0 {
				t.Errorf("Expected empty params for empty pattern, got %v", result)
			}
		})
	}
}

// TestProperty_KnownPatternsProduceExpectedResults verifies known inputs produce expected outputs
func TestProperty_KnownPatternsProduceExpectedResults(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		pattern  string
		expected []string
	}{
		{"", []string{}},
		{"/v1/users", []string{}},
		{"/v1/users/{id}", []string{"id"}},
		{"/v1/users/{user_id}/posts/{post_id}", []string{"user_id", "post_id"}},
		{"/v1/{a}/{b}/{c}", []string{"a", "b", "c"}},
		{"/users/{user.name}", []string{"user.name"}},
		{"/{org_id}/projects/{project_id}/tasks/{task_id}", []string{"org_id", "project_id", "task_id"}},
	}

	parsers := []struct {
		name   string
		parser interface{ ParsePathParams(string) []string }
	}{
		{"proto3", NewProto3Parser()},
		{"proto2", NewProto2Parser()},
		{"editions", NewEditionsParser()},
	}

	for _, tc := range testCases {
		for _, p := range parsers {
			t.Run(p.name+"_"+tc.pattern, func(t *testing.T) {
				result := p.parser.ParsePathParams(tc.pattern)

				if len(result) != len(tc.expected) {
					t.Errorf("Expected %d params, got %d: %v", len(tc.expected), len(result), result)
					return
				}

				for i, exp := range tc.expected {
					if result[i] != exp {
						t.Errorf("Expected param[%d] = %q, got %q", i, exp, result[i])
					}
				}
			})
		}
	}
}
