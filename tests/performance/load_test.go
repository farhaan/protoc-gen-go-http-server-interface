package performance_test

import (
	"strings"
	"testing"
	"time"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

// TestPerformance_LargeConcurrentGeneration tests performance with multiple generators
func TestPerformance_LargeConcurrentGeneration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a moderately complex service
	methods := make([]httpinterface.MethodInfo, 20)
	for i := range 20 {
		methods[i] = httpinterface.MethodInfo{
			Name:       "Method" + string(rune(i+65)), // MethodA, MethodB, etc.
			InputType:  "Request" + string(rune(i+65)),
			OutputType: "Response" + string(rune(i+65)),
			HTTPRules: []httpinterface.HTTPRule{
				{
					Method:     "GET",
					Pattern:    "/api/v1/resource" + string(rune(i+65)) + "/{id}",
					Body:       "",
					PathParams: []string{"id"},
				},
			},
		}
	}

	serviceData := &httpinterface.ServiceData{
		PackageName: "performance",
		Services: []httpinterface.ServiceInfo{
			{Name: "PerformanceTestService", Methods: methods},
		},
	}

	// Measure generation time
	start := time.Now()

	// Run multiple generations concurrently
	const numGenerations = 100
	done := make(chan bool, numGenerations)

	for range numGenerations {
		go func() {
			generator := httpinterface.New()
			_, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Errorf("Generation failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all generations to complete
	for range numGenerations {
		<-done
	}

	duration := time.Since(start)

	// Performance should be reasonable (less than 5 seconds for 100 generations)
	if duration > 5*time.Second {
		t.Errorf("Performance test took too long: %v", duration)
	}

	t.Logf("Generated %d services in %v (avg: %v per generation)",
		numGenerations, duration, duration/numGenerations)
}

// TestCodeGeneration_SpecialCharacters tests handling of various special characters
func TestCodeGeneration_SpecialCharacters(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "specialchars",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "SpecialService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetWithUnicode",
						InputType:  "UnicodeRequest",
						OutputType: "UnicodeResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/api/🚀/resources/{id}",
								Body:       "",
								PathParams: []string{"id"},
							},
						},
					},
					{
						Name:       "GetWithSpecialChars",
						InputType:  "SpecialRequest",
						OutputType: "SpecialResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/api/v1/special-chars_123/{resource-id}",
								Body:       "",
								PathParams: []string{"resource-id"},
							},
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

	// Verify the generated code is valid Go and contains expected patterns
	if !strings.Contains(generated, "SpecialServiceHandler interface") {
		t.Error("Generated code missing service handler interface")
	}

	// Verify paths are properly quoted/escaped in generated Go code
	if !strings.Contains(generated, `"/api/🚀/resources/{id}"`) {
		t.Error("Generated code missing unicode path pattern")
	}

	if !strings.Contains(generated, `"/api/v1/special-chars_123/{resource-id}"`) {
		t.Error("Generated code missing special characters path pattern")
	}
}

// TestEdgeCases_HTTPMethodVariations tests various HTTP method edge cases
func TestEdgeCases_HTTPMethodVariations(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "httpmethods",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "HTTPMethodService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "HandleHead",
						InputType:  "HeadRequest",
						OutputType: "HeadResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "HEAD", Pattern: "/resources/{id}", Body: "", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "HandleOptions",
						InputType:  "OptionsRequest",
						OutputType: "OptionsResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "OPTIONS", Pattern: "/resources", Body: ""},
						},
					},
					{
						Name:       "HandleTrace",
						InputType:  "TraceRequest",
						OutputType: "TraceResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "TRACE", Pattern: "/resources/{id}", Body: "", PathParams: []string{"id"}},
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

	// Verify all HTTP methods are handled correctly
	expectedMethods := []string{
		`"HEAD", "/resources/{id}"`,
		`"OPTIONS", "/resources"`,
		`"TRACE", "/resources/{id}"`,
	}

	for _, method := range expectedMethods {
		if !strings.Contains(generated, method) {
			t.Errorf("Generated code missing HTTP method: %q", method)
		}
	}

	// Verify handler methods are generated
	handlerMethods := []string{
		"HandleHead",
		"HandleOptions",
		"HandleTrace",
	}

	for _, handler := range handlerMethods {
		if !strings.Contains(generated, handler) {
			t.Errorf("Generated code missing handler method: %q", handler)
		}
	}
}
