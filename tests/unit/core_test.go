package unit_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestCore_BasicServiceGeneration tests basic service generation
func TestCore_BasicServiceGeneration(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// Create a basic service definition
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
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/test/{id}",
								Body:       "",
								PathParams: []string{"id"},
							},
						},
					},
					{
						Name:       "CreateTest",
						InputType:  "CreateTestRequest",
						OutputType: "TestResponse",
						HTTPRules: []parser.HTTPRule{
							{
								Method:  "POST",
								Pattern: "/test",
								Body:    "*",
							},
						},
					},
				},
			},
		},
	}

	// Generate code
	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Test basic patterns
	expectedPatterns := []string{
		"package testpkg",
		"import (",
		"\"net/http\"",
		"type TestServiceHandler interface",
		"HandleGetTest(w http.ResponseWriter, r *http.Request)",
		"HandleCreateTest(w http.ResponseWriter, r *http.Request)",
		"type Routes interface",
		"RegisterTestServiceRoutes",
		`http.MethodGet, "/test/{id}"`,
		`http.MethodPost, "/test"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}

	// Ensure generated code is not empty
	if len(generated) < 100 {
		t.Errorf("Generated code seems too short: %d bytes", len(generated))
	}
}

// TestCore_MultipleServices tests generation with multiple services
func TestCore_MultipleServices(t *testing.T) {
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
						HTTPRules: []parser.HTTPRule{
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
						HTTPRules: []parser.HTTPRule{
							{Method: "GET", Pattern: "/orders/{id}", PathParams: []string{"id"}},
						},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Multi-service generation failed: %v", err)
	}

	// Verify both services are present
	expectedServices := []string{
		"type UserServiceHandler interface",
		"type OrderServiceHandler interface",
		"RegisterUserServiceRoutes",
		"RegisterOrderServiceRoutes",
		`http.MethodGet, "/users/{id}"`,
		`http.MethodGet, "/orders/{id}"`,
	}

	for _, pattern := range expectedServices {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Multi-service code missing expected pattern: %q", pattern)
		}
	}
}

// TestCore_HTTPMethods tests various HTTP method generation
func TestCore_HTTPMethods(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "httpmethods",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "HTTPService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "GetResource",
						HTTPRules: []parser.HTTPRule{
							{Method: "GET", Pattern: "/resources/{id}", PathParams: []string{"id"}},
						},
					},
					{
						Name: "CreateResource",
						HTTPRules: []parser.HTTPRule{
							{Method: "POST", Pattern: "/resources", Body: "*"},
						},
					},
					{
						Name: "UpdateResource",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "PUT",
								Pattern:    "/resources/{id}",
								Body:       "*",
								PathParams: []string{"id"},
							},
						},
					},
					{
						Name: "PatchResource",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "PATCH",
								Pattern:    "/resources/{id}",
								Body:       "patch",
								PathParams: []string{"id"},
							},
						},
					},
					{
						Name: "DeleteResource",
						HTTPRules: []parser.HTTPRule{
							{Method: "DELETE", Pattern: "/resources/{id}", PathParams: []string{"id"}},
						},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("HTTP methods generation failed: %v", err)
	}

	// Verify all HTTP methods
	expectedMethods := []string{
		`http.MethodGet, "/resources/{id}"`,
		`http.MethodPost, "/resources"`,
		`http.MethodPut, "/resources/{id}"`,
		`http.MethodPatch, "/resources/{id}"`,
		`http.MethodDelete, "/resources/{id}"`,
		"HandleGetResource",
		"HandleCreateResource",
		"HandleUpdateResource",
		"HandlePatchResource",
		"HandleDeleteResource",
	}

	for _, method := range expectedMethods {
		if !strings.Contains(generated, method) {
			t.Errorf("HTTP methods code missing expected method: %q", method)
		}
	}
}

// TestCore_ProtocPluginInterface tests the protoc plugin interface
func TestCore_ProtocPluginInterface(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// Create a minimal but valid protobuf request
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(""),
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/test;test"),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       proto.String("GetTest"),
								InputType:  proto.String("GetTestRequest"),
								OutputType: proto.String("TestResponse"),
								Options:    &descriptorpb.MethodOptions{},
							},
						},
					},
				},
			},
		},
	}

	response := generator.Generate(request)

	// Should not have errors for valid input
	if response.GetError() != "" {
		t.Errorf("Protoc plugin interface returned error: %s", response.GetError())
	}

	// Should generate no files since there are no HTTP annotations
	if len(response.GetFile()) > 0 {
		t.Errorf("Expected no files for service without HTTP annotations, got %d files", len(response.GetFile()))
	}
}

// TestCore_PackageHandling tests various package naming scenarios
func TestCore_PackageHandling(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		packageName string
		expected    string
	}{
		{
			name:        "simple_package",
			packageName: "simple",
			expected:    "package simple",
		},
		{
			name:        "versioned_package",
			packageName: "apiv1",
			expected:    "package apiv1",
		},
		{
			name:        "underscored_package",
			packageName: "user_service",
			expected:    "package user_service",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			serviceData := &httpinterface.ServiceData{
				PackageName: tc.packageName,
				Services: []httpinterface.ServiceInfo{
					{
						Name: "TestService",
						Methods: []httpinterface.MethodInfo{
							{
								Name: "TestMethod",
								HTTPRules: []parser.HTTPRule{
									{Method: "GET", Pattern: "/test"},
								},
							},
						},
					},
				},
			}

			generated, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Fatalf("Package handling failed: %v", err)
			}

			if !strings.Contains(generated, tc.expected) {
				t.Errorf("Generated code missing expected package: %q", tc.expected)
			}
		})
	}
}

// TestCore_EmptyGeneration tests handling of edge cases that should generate minimal code
func TestCore_EmptyGeneration(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	testCases := []struct {
		name        string
		serviceData *httpinterface.ServiceData
		description string
	}{
		{
			name: "no_services",
			serviceData: &httpinterface.ServiceData{
				PackageName: "empty",
				Services:    []httpinterface.ServiceInfo{},
			},
			description: "No services defined",
		},
		{
			name: "service_no_methods",
			serviceData: &httpinterface.ServiceData{
				PackageName: "nomethods",
				Services: []httpinterface.ServiceInfo{
					{
						Name:    "EmptyService",
						Methods: []httpinterface.MethodInfo{},
					},
				},
			},
			description: "Service with no methods",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generated, err := generator.GenerateCode(tc.serviceData)
			if err != nil {
				t.Fatalf("Empty generation failed: %v", err)
			}

			// Should generate basic package structure even for empty services
			if tc.serviceData.PackageName != "" {
				expectedPackage := "package " + tc.serviceData.PackageName
				if !strings.Contains(generated, expectedPackage) {
					t.Errorf("Empty generation missing package declaration: %q", expectedPackage)
				}
			}

			t.Logf("%s: Generated %d bytes", tc.description, len(generated))
		})
	}
}
