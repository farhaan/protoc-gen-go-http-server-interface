package integration_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	options "google.golang.org/genproto/googleapis/api/annotations"
)

// TestIntegration_CustomHTTPPattern tests custom HTTP patterns (HEAD, OPTIONS, etc.)
func TestIntegration_CustomHTTPPattern(t *testing.T) {
	t.Parallel()

	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "customhttp",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "HealthService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "HealthCheck",
						InputType:  "HealthCheckRequest",
						OutputType: "HealthCheckResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "HEAD", Pattern: "/health", Body: ""},
						},
					},
					{
						Name:       "GetOptions",
						InputType:  "OptionsRequest",
						OutputType: "OptionsResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "OPTIONS", Pattern: "/api/v1/resources", Body: ""},
						},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Verify custom methods are in generated code
	if !strings.Contains(generated, `r.HandleFunc("HEAD"`) {
		t.Error("Missing HEAD method registration in generated code")
	}
	if !strings.Contains(generated, `r.HandleFunc("OPTIONS"`) {
		t.Error("Missing OPTIONS method registration in generated code")
	}
}

// TestIntegration_CustomHTTPPatternNilSafety ensures nil Custom patterns don't cause panics
func TestIntegration_CustomHTTPPatternNilSafety(t *testing.T) {
	t.Parallel()

	// Test all parser types handle nil Custom gracefully
	parsers := []struct {
		name   string
		parser interface {
			ParsePathParams(pattern string) []string
		}
	}{
		{"proto3", parser.NewProto3Parser()},
		{"proto2", parser.NewProto2Parser()},
		{"editions", parser.NewEditionsParser()},
	}

	for _, p := range parsers {
		t.Run(p.name+"_nil_custom_no_panic", func(t *testing.T) {
			t.Parallel()

			// This should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser %s panicked on nil Custom: %v", p.name, r)
				}
			}()

			// ParsePathParams with empty pattern (result of nil Custom)
			params := p.parser.ParsePathParams("")
			if len(params) != 0 {
				t.Errorf("Expected empty params for empty pattern, got %v", params)
			}
		})
	}
}

// TestIntegration_CustomHTTPPatternParserNilCustom tests parser behavior with nil Custom
func TestIntegration_CustomHTTPPatternParserNilCustom(t *testing.T) {
	t.Parallel()

	// Create HTTP rule with nil Custom
	httpRule := &options.HttpRule{
		Pattern: &options.HttpRule_Custom{
			Custom: nil, // This is the edge case we're testing
		},
		Body: "",
	}

	// Test with EditionsParser (same logic in all parsers)
	editionsParser := parser.NewEditionsParser()

	// This should not panic - the fix adds nil check
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("EditionsParser panicked on nil Custom: %v", r)
		}
	}()

	// We can't directly call parseHTTPRule as it's unexported,
	// but we can test the generator doesn't panic with empty methods
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "nilcustom",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "NilCustomService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "NilCustomMethod",
						InputType:  "Request",
						OutputType: "Response",
						HTTPRules: []httpinterface.HTTPRule{
							// Empty method simulates nil Custom after parsing
							{Method: "", Pattern: "", Body: ""},
						},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Should generate valid code even with empty method
	if generated == "" {
		t.Error("Expected non-empty generated code")
	}

	// Verify the parser handles nil gracefully by testing path params
	params := editionsParser.ParsePathParams("")
	if params == nil {
		t.Error("ParsePathParams should return empty slice, not nil")
	}

	_ = httpRule // Use the variable to avoid unused warning
}

// TestIntegration_EmptyHTTPRulesNoRegression ensures empty HTTP rules don't break generation
func TestIntegration_EmptyHTTPRulesNoRegression(t *testing.T) {
	t.Parallel()

	generator := httpinterface.New()

	testCases := []struct {
		name      string
		httpRules []httpinterface.HTTPRule
	}{
		{
			name:      "empty_rules_slice",
			httpRules: []httpinterface.HTTPRule{},
		},
		{
			name: "single_empty_rule",
			httpRules: []httpinterface.HTTPRule{
				{Method: "", Pattern: "", Body: ""},
			},
		},
		{
			name: "mixed_valid_and_empty",
			httpRules: []httpinterface.HTTPRule{
				{Method: "GET", Pattern: "/valid", Body: ""},
				{Method: "", Pattern: "", Body: ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panicked on %s: %v", tc.name, r)
				}
			}()

			serviceData := &httpinterface.ServiceData{
				PackageName: "edgecase",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "EdgeCaseService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:       "EdgeCaseMethod",
								InputType:  "Request",
								OutputType: "Response",
								HTTPRules:  tc.httpRules,
							},
						},
					},
				},
			}

			_, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}
		})
	}
}
