package httpinterface

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Test protobuf 3 specific features and compliance
func TestProtobuf3Compliance(t *testing.T) {
	t.Parallel()
	g := New()

	// Test that the plugin declares support for proto3 optional
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
			},
		},
	}

	resp := g.Generate(req)

	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	// Check that supported features includes FEATURE_PROTO3_OPTIONAL
	if resp.SupportedFeatures == nil {
		t.Fatal("SupportedFeatures is nil")
	}

	features := resp.GetSupportedFeatures()
	expectedFeature := uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

	if (features & expectedFeature) == 0 {
		t.Errorf("Plugin does not declare support for FEATURE_PROTO3_OPTIONAL")
	}
}

// Test handling of proto3 optional fields in messages
func TestProto3OptionalFields(t *testing.T) {
	t.Parallel()
	g := NewWith(mockGetHTTPRules, mockGetPathParams, mockConvertPathPattern)

	// Create a proto file with proto3 optional fields (simulated)
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Syntax:  proto.String("proto3"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.OptionalRequest"),
						OutputType: proto.String(".test.OptionalResponse"),
					},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if len(data.Services) == 0 {
		t.Fatal("No services generated")
	}

	service := data.Services[0]
	if len(service.Methods) == 0 {
		t.Fatal("No methods generated")
	}

	method := service.Methods[0]
	if method.InputType != "OptionalRequest" {
		t.Errorf("InputType = %q, want %q", method.InputType, "OptionalRequest")
	}

	if method.OutputType != "OptionalResponse" {
		t.Errorf("OutputType = %q, want %q", method.OutputType, "OptionalResponse")
	}
}

// Test that enum handling works correctly with proto3
func TestProto3EnumHandling(t *testing.T) {
	t.Parallel()
	g := New()

	// Test enum type name extraction
	tests := []struct {
		input    string
		expected string
	}{
		{".test.Status", "Status"},
		{".api.v1.UserRole", "UserRole"},
		{".enterprise.department.AccessLevel", "AccessLevel"},
	}

	for _, test := range tests {
		result := g.getTypeName(test.input)
		if result != test.expected {
			t.Errorf("getTypeName(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

// Test comprehensive HTTP method support
func TestHTTPMethodSupport(t *testing.T) {
	t.Parallel()

	// Mock function that returns different HTTP methods
	httpRuleExtractor := func(method *descriptorpb.MethodDescriptorProto) []HTTPRule {
		methodName := method.GetName()
		switch methodName {
		case "GetMethod":
			return []HTTPRule{{Method: "GET", Pattern: "/api/get", Body: ""}}
		case "PostMethod":
			return []HTTPRule{{Method: "POST", Pattern: "/api/post", Body: "*"}}
		case "PutMethod":
			return []HTTPRule{{Method: "PUT", Pattern: "/api/put", Body: "*"}}
		case "DeleteMethod":
			return []HTTPRule{{Method: "DELETE", Pattern: "/api/delete", Body: ""}}
		case "PatchMethod":
			return []HTTPRule{{Method: "PATCH", Pattern: "/api/patch", Body: "*"}}
		case "CustomMethod":
			return []HTTPRule{{Method: "OPTIONS", Pattern: "/api/custom", Body: ""}}
		}
		return nil
	}

	g := New(httpRuleExtractor)

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: proto.String("GetMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
					{Name: proto.String("PostMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
					{Name: proto.String("PutMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
					{Name: proto.String("DeleteMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
					{Name: proto.String("PatchMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
					{Name: proto.String("CustomMethod"), InputType: proto.String(".test.Request"), OutputType: proto.String(".test.Response")},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if len(data.Services) == 0 {
		t.Fatal("No services generated")
	}

	service := data.Services[0]
	expectedMethods := map[string]string{
		"GetMethod":    "GET",
		"PostMethod":   "POST",
		"PutMethod":    "PUT",
		"DeleteMethod": "DELETE",
		"PatchMethod":  "PATCH",
		"CustomMethod": "OPTIONS",
	}

	if len(service.Methods) != len(expectedMethods) {
		t.Fatalf("Expected %d methods, got %d", len(expectedMethods), len(service.Methods))
	}

	for _, method := range service.Methods {
		expectedHTTPMethod, exists := expectedMethods[method.Name]
		if !exists {
			t.Errorf("Unexpected method: %s", method.Name)
			continue
		}

		if len(method.HTTPRules) == 0 {
			t.Errorf("Method %s has no HTTP rules", method.Name)
			continue
		}

		if method.HTTPRules[0].Method != expectedHTTPMethod {
			t.Errorf("Method %s: expected HTTP method %s, got %s",
				method.Name, expectedHTTPMethod, method.HTTPRules[0].Method)
		}
	}
}

// Test nested message type handling
func TestNestedMessageTypes(t *testing.T) {
	t.Parallel()
	g := New(mockGetHTTPRules)

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.outer.InnerRequest"),
						OutputType: proto.String(".test.outer.InnerResponse"),
					},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if len(data.Services) == 0 {
		t.Fatal("No services generated")
	}

	method := data.Services[0].Methods[0]
	if method.InputType != "InnerRequest" {
		t.Errorf("InputType = %q, want %q", method.InputType, "InnerRequest")
	}

	if method.OutputType != "InnerResponse" {
		t.Errorf("OutputType = %q, want %q", method.OutputType, "InnerResponse")
	}
}

// Test that generated code contains expected protobuf 3 compatible patterns
func TestGeneratedCodePatterns(t *testing.T) {
	t.Parallel()
	g := NewWith(mockGetHTTPRules, mockGetPathParams, mockConvertPathPattern)

	data := &ServiceData{
		PackageName: "testpkg",
		Services: []ServiceInfo{
			{
				Name: "UserService",
				Methods: []MethodInfo{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "User",
						HTTPRules: []HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/users/:id",
								Body:       "",
								PathParams: []string{"id"},
							},
						},
					},
					{
						Name:       "CreateUser",
						InputType:  "CreateUserRequest",
						OutputType: "User",
						HTTPRules: []HTTPRule{
							{
								Method:     "POST",
								Pattern:    "/users",
								Body:       "*",
								PathParams: []string{},
							},
						},
					},
				},
			},
		},
	}

	code, err := g.GenerateCode(data)
	if err != nil {
		t.Fatalf("generateCode() error = %v", err)
	}

	// Check for protobuf 3 compatible patterns
	expectedPatterns := []string{
		// Package declaration
		"package testpkg",

		// Interface definitions
		"type UserServiceHandler interface",
		"HandleGetUser",
		"HandleCreateUser",

		// Route registration functions
		"func RegisterGetUserRoute",
		"func RegisterCreateUserRoute",

		// Router functions
		"func NewRouter(mux *http.ServeMux)",
		"func DefaultRouter()",

		// RouteGroup methods
		"func (g *RouteGroup) RegisterGetUser",
		"func (g *RouteGroup) RegisterCreateUser",

		// HTTP method handling
		"GET",
		"POST",

		// Path patterns
		"/users",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}

	// Check that the generated code doesn't contain deprecated patterns
	deprecatedPatterns := []string{
		"github.com/golang/protobuf",
		"protoc-gen-go/descriptor",
		"protoc-gen-go/plugin",
	}

	for _, pattern := range deprecatedPatterns {
		if strings.Contains(code, pattern) {
			t.Errorf("Generated code contains deprecated pattern: %q", pattern)
		}
	}
}

// Test empty service handling
func TestEmptyService(t *testing.T) {
	t.Parallel()
	g := New()

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:   proto.String("EmptyService"),
				Method: []*descriptorpb.MethodDescriptorProto{},
			},
		},
	}

	data := g.buildServiceData(file)

	// Empty service should not be included in the data
	if len(data.Services) != 0 {
		t.Errorf("Expected 0 services for empty service, got %d", len(data.Services))
	}
}

// Test multiple services in one file
func TestMultipleServices(t *testing.T) {
	t.Parallel()
	g := New(mockGetHTTPRules)

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("UserService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.UserRequest"),
						OutputType: proto.String(".test.User"),
					},
				},
			},
			{
				Name: proto.String("ProductService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.ProductRequest"),
						OutputType: proto.String(".test.Product"),
					},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if len(data.Services) != 2 {
		t.Fatalf("Expected 2 services, got %d", len(data.Services))
	}

	serviceNames := []string{data.Services[0].Name, data.Services[1].Name}
	expectedNames := []string{"UserService", "ProductService"}

	for _, expected := range expectedNames {
		found := false
		for _, actual := range serviceNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service %s not found", expected)
		}
	}
}

// Test error handling in code generation
func TestCodeGenerationErrorHandling(t *testing.T) {
	t.Parallel()
	g := New()

	// Test with invalid parameter format (no = sign)
	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("invalid-format-no-equals"),
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
			},
		},
	}

	resp := g.Generate(req)

	if resp.Error == nil {
		t.Error("Expected error for invalid parameter format")
	}

	if resp.Error != nil && !strings.Contains(*resp.Error, "invalid parameter") {
		t.Errorf("Error message should mention 'invalid parameter', got: %s", *resp.Error)
	}
}

// Test that plugin version is properly set
func TestPluginVersion(t *testing.T) {
	// This tests the main function's version flag, but we can test
	// that the generator maintains consistent behavior
	g := New()

	if g == nil {
		t.Fatal("Generator should not be nil")
	}

	if g.ParsedTemplates == nil {
		t.Fatal("ParsedTemplates should not be nil")
	}

	if g.Options == nil {
		t.Fatal("Options should not be nil")
	}
}
