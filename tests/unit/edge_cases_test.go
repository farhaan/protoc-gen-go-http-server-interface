package unit_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestAdvancedEdgeCases_HTTPBodyVariations tests different HTTP body patterns
func TestAdvancedEdgeCases_HTTPBodyVariations(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "bodytest",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "BodyTestService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "CreateWithFullBody",
						InputType:  "CreateRequest",
						OutputType: "Response",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/items", Body: "*"},
						},
					},
					{
						Name:       "UpdateWithFieldBody",
						InputType:  "UpdateRequest",
						OutputType: "Response",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PATCH", Pattern: "/items/{id}", Body: "item", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "UpdateWithNestedBody",
						InputType:  "UpdateNestedRequest",
						OutputType: "Response",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PUT", Pattern: "/items/{id}", Body: "item.data", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "GetWithEmptyBody",
						InputType:  "GetRequest",
						OutputType: "Response",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/items/{id}", Body: "", PathParams: []string{"id"}},
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

	// Verify all HTTP body patterns are handled correctly
	expectedPatterns := []string{
		"package bodytest",
		"BodyTestServiceHandler interface",
		"HandleCreateWithFullBody",
		"HandleUpdateWithFieldBody",
		"HandleUpdateWithNestedBody",
		"HandleGetWithEmptyBody",
		`"POST", "/items"`,
		`"PATCH", "/items/{id}"`,
		`"PUT", "/items/{id}"`,
		`"GET", "/items/{id}"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}
}

// TestAdvancedEdgeCases_ComplexPathPatterns tests complex URL path patterns
func TestAdvancedEdgeCases_ComplexPathPatterns(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		pattern     string
		pathParams  []string
		expectError bool
	}{
		{
			name:       "simple_parameter",
			pattern:    "/users/{id}",
			pathParams: []string{"id"},
		},
		{
			name:       "multiple_parameters",
			pattern:    "/users/{user_id}/posts/{post_id}",
			pathParams: []string{"user_id", "post_id"},
		},
		{
			name:       "nested_field_parameter",
			pattern:    "/users/{user.id}",
			pathParams: []string{"user.id"},
		},
		{
			name:       "deep_nested_parameter",
			pattern:    "/orgs/{org.id}/projects/{project.metadata.id}/resources/{resource.spec.name}",
			pathParams: []string{"org.id", "project.metadata.id", "resource.spec.name"},
		},
		{
			name:       "mixed_parameters",
			pattern:    "/api/v1/{tenant}/users/{user.profile.id}/settings/{setting_key}",
			pathParams: []string{"tenant", "user.profile.id", "setting_key"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			serviceData := &httpinterface.ServiceData{
				PackageName: "pathtest",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "PathTestService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:       "TestMethod",
								InputType:  "TestRequest",
								OutputType: "TestResponse",
								HTTPRules: []httpinterface.HTTPRule{
									{
										Method:     "GET",
										Pattern:    tc.pattern,
										Body:       "",
										PathParams: tc.pathParams,
									},
								},
							},
						},
					},
				},
			}

			generated, err := generator.GenerateCode(serviceData)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError {
				if !strings.Contains(generated, tc.pattern) {
					t.Errorf("Generated code missing path pattern: %q", tc.pattern)
				}
			}
		})
	}
}

// TestAdvancedEdgeCases_ErrorHandling tests error handling with malformed inputs
func TestAdvancedEdgeCases_ErrorHandling(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name           string
		request        *pluginpb.CodeGeneratorRequest
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "invalid_paths_parameter",
			request: &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String("paths=invalid_value"),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			},
			expectError:    true,
			expectedErrMsg: "unknown paths option",
		},
		{
			name: "unknown_parameter",
			request: &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String("unknown_param=value"),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			},
			expectError:    true,
			expectedErrMsg: "unknown option",
		},
		{
			name: "malformed_parameter",
			request: &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String("malformed"),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			},
			expectError:    true,
			expectedErrMsg: "invalid parameter",
		},
		{
			name: "valid_parameter",
			request: &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String("paths=source_relative"),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := generator.Generate(tc.request)
			hasError := response.GetError() != ""

			if hasError != tc.expectError {
				t.Errorf("Expected error: %v, got error: %v (error: %s)",
					tc.expectError, hasError, response.GetError())
			}

			if tc.expectError && tc.expectedErrMsg != "" {
				if !strings.Contains(response.GetError(), tc.expectedErrMsg) {
					t.Errorf("Expected error message to contain %q, got: %s",
						tc.expectedErrMsg, response.GetError())
				}
			}
		})
	}
}

// TestAdvancedEdgeCases_LargeServiceFiles tests handling of large service definitions
func TestAdvancedEdgeCases_LargeServiceFiles(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping large service test in short mode")
	}

	generator := httpinterface.New()

	// Create a large service with many methods
	const numMethods = 100
	methods := make([]httpinterface.MethodInfo, numMethods)

	for i := range numMethods {
		methods[i] = httpinterface.MethodInfo{
			Name:       fmt.Sprintf("Method%d", i),
			InputType:  fmt.Sprintf("Request%d", i),
			OutputType: fmt.Sprintf("Response%d", i),
			HTTPRules: []httpinterface.HTTPRule{
				{
					Method:     "GET",
					Pattern:    fmt.Sprintf("/api/v1/resource%d/{id}", i),
					Body:       "",
					PathParams: []string{"id"},
				},
			},
		}
	}

	serviceData := &httpinterface.ServiceData{
		PackageName: "largeservice",
		Services: []httpinterface.ServiceInfo{
			{
				Name:    "LargeService",
				Methods: methods,
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Large service generation failed: %v", err)
	}

	// Verify basic structure is present
	expectedPatterns := []string{
		"package largeservice",
		"LargeServiceHandler interface",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Large service missing expected pattern: %q", pattern)
		}
	}

	// Verify some methods are present (spot check)
	methodChecks := []string{
		"HandleMethod0",
		"HandleMethod50",
		"HandleMethod99",
		`"/api/v1/resource0/{id}"`,
		`"/api/v1/resource50/{id}"`,
		`"/api/v1/resource99/{id}"`,
	}

	for _, check := range methodChecks {
		if !strings.Contains(generated, check) {
			t.Errorf("Large service missing method pattern: %q", check)
		}
	}

	t.Logf("Large service test completed. Generated %d bytes for %d methods", len(generated), numMethods)
}

// TestAdvancedEdgeCases_EmptyServiceHandling tests various empty/minimal service scenarios
func TestAdvancedEdgeCases_EmptyServiceHandling(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		serviceData *httpinterface.ServiceData
		expectEmpty bool
		description string
	}{
		{
			name: "completely_empty",
			serviceData: &httpinterface.ServiceData{
				PackageName: "empty",
				Services:    []httpinterface.ServiceInfo{},
			},
			expectEmpty: true,
			description: "No services at all",
		},
		{
			name: "empty_methods",
			serviceData: &httpinterface.ServiceData{
				PackageName: "emptymethods",
				Services: []httpinterface.ServiceInfo{
					{
						Name:    "EmptyService",
						Methods: []httpinterface.MethodInfo{},
					},
				},
			},
			expectEmpty: false, // Should still generate basic structure
			description: "Service with no methods",
		},
		{
			name: "methods_without_http_rules",
			serviceData: &httpinterface.ServiceData{
				PackageName: "nohttprules",
				Services: []httpinterface.ServiceInfo{
					{
						Name: "NoHTTPService",
						Methods: []httpinterface.MethodInfo{
							{
								Name:      "MethodWithoutHTTP",
								HTTPRules: []httpinterface.HTTPRule{},
							},
						},
					},
				},
			},
			expectEmpty: false, // Should generate structure but no routes
			description: "Methods without HTTP rules",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generated, err := generator.GenerateCode(tc.serviceData)
			if err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			if tc.expectEmpty {
				// Should contain minimal structure but no service handlers
				if strings.Contains(generated, "ServiceHandler interface") {
					t.Error("Expected no service handlers for empty services")
				}
			} else {
				// Should contain package declaration at minimum
				if !strings.Contains(generated, "package "+tc.serviceData.PackageName) {
					t.Error("Generated code missing package declaration")
				}
			}

			t.Logf("%s: Generated %d bytes", tc.description, len(generated))
		})
	}
}
