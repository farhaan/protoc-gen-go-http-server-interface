package unit_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

// TestGenerator_BasicServiceGeneration tests basic service generation functionality
func TestGenerator_BasicServiceGeneration(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "testpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "TestService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetTest",
						InputType:  "GetTestRequest",
						OutputType: "TestResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/test/{id}",
								Body:       "",
								PathParams: []string{"id"},
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

	// Verify expected patterns
	expectedPatterns := []string{
		"package testpkg",
		"type TestServiceHandler interface",
		"HandleGetTest(w http.ResponseWriter, r *http.Request)",
		"RegisterTestServiceRoutes",
		`"GET", "/test/{id}"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}
}

// TestGenerator_MultipleHTTPMethods tests various HTTP method generation
func TestGenerator_MultipleHTTPMethods(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "testpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "RESTService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetResource",
						InputType:  "GetResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/resources/{id}", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "CreateResource",
						InputType:  "CreateResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/resources", Body: "*"},
						},
					},
					{
						Name:       "UpdateResource",
						InputType:  "UpdateResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PUT", Pattern: "/resources/{id}", Body: "*", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "PatchResource",
						InputType:  "PatchResourceRequest",
						OutputType: "Resource",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PATCH", Pattern: "/resources/{id}", Body: "patch", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "DeleteResource",
						InputType:  "DeleteResourceRequest",
						OutputType: "DeleteResourceResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "DELETE", Pattern: "/resources/{id}", PathParams: []string{"id"}},
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

	// Verify all HTTP methods are generated
	expectedMethods := []string{
		`"GET", "/resources/{id}"`,
		`"POST", "/resources"`,
		`"PUT", "/resources/{id}"`,
		`"PATCH", "/resources/{id}"`,
		`"DELETE", "/resources/{id}"`,
	}

	for _, method := range expectedMethods {
		if !strings.Contains(generated, method) {
			t.Errorf("Generated code missing HTTP method: %q", method)
		}
	}
}

// TestGenerator_MultipleServices tests multiple service generation
func TestGenerator_MultipleServices(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "multisvc",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "UserService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "GetUser",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/users/{id}", PathParams: []string{"id"}},
						},
					},
				},
			},
			{
				Name: "OrderService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "GetOrder",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/orders/{id}", PathParams: []string{"id"}},
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

	// Verify both services are generated
	expectedServices := []string{
		"type UserServiceHandler interface",
		"type OrderServiceHandler interface",
		"RegisterUserServiceRoutes",
		"RegisterOrderServiceRoutes",
	}

	for _, service := range expectedServices {
		if !strings.Contains(generated, service) {
			t.Errorf("Generated code missing service element: %q", service)
		}
	}
}

// TestGenerator_ComplexPathParameters tests complex path parameter handling
func TestGenerator_ComplexPathParameters(t *testing.T) {
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
			pattern:    "/users/{user_id}/orders/{order_id}",
			pathParams: []string{"user_id", "order_id"},
		},
		{
			name:       "nested_field_parameter",
			pattern:    "/users/{user.id}/profile",
			pathParams: []string{"user.id"},
		},
		{
			name:       "deep_nested_parameter",
			pattern:    "/orgs/{org}/projects/{project}/resources/{resource.metadata.id}",
			pathParams: []string{"org", "project", "resource.metadata.id"},
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

// TestGenerator_EmptyServiceHandling tests handling of services without HTTP rules
func TestGenerator_EmptyServiceHandling(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		serviceData *httpinterface.ServiceData
		expectEmpty bool
	}{
		{
			name: "completely_empty_services",
			serviceData: &httpinterface.ServiceData{
				PackageName: "empty",
				Services:    []httpinterface.ServiceInfo{},
			},
			expectEmpty: true,
		},
		{
			name: "service_with_empty_methods",
			serviceData: &httpinterface.ServiceData{
				PackageName: "emptymethods",
				Services: []httpinterface.ServiceInfo{
					{
						Name:    "EmptyService",
						Methods: []httpinterface.MethodInfo{},
					},
				},
			},
			expectEmpty: false, // Should still generate handler interface
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generated, err := generator.GenerateCode(tc.serviceData)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			if tc.expectEmpty {
				// Should contain basic structure but no service handlers
				if strings.Contains(generated, "ServiceHandler") {
					t.Error("Expected no service handlers for empty services")
				}
			} else {
				// Should contain package and basic structure
				if !strings.Contains(generated, "package "+tc.serviceData.PackageName) {
					t.Error("Generated code missing package declaration")
				}
			}
		})
	}
}
