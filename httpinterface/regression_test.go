package httpinterface

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// Regression tests to ensure existing functionality is preserved

// TestRegressionBasicGeneration tests basic code generation hasn't regressed
func TestRegressionBasicGeneration(t *testing.T) {
	t.Parallel()
	g := New()

	// Create a service without HTTP annotations - this should generate no files
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"user.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("user.proto"),
				Package: proto.String("api"),
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("UserService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:       proto.String("GetUser"),
								InputType:  proto.String(".api.GetUserRequest"),
								OutputType: proto.String(".api.User"),
							},
							{
								Name:       proto.String("CreateUser"),
								InputType:  proto.String(".api.CreateUserRequest"),
								OutputType: proto.String(".api.User"),
							},
						},
					},
				},
			},
		},
	}

	resp := g.Generate(req)

	if resp.Error != nil {
		t.Fatalf("Generation failed: %s", *resp.Error)
	}

	// With no HTTP annotations, no files should be generated
	if len(resp.File) != 0 {
		t.Errorf("Expected 0 files (no HTTP annotations), got %d", len(resp.File))
	}
}

// TestRegressionRouteGroupGeneration ensures RouteGroup functionality is preserved
func TestRegressionRouteGroupGeneration(t *testing.T) {
	t.Parallel()
	g := New()

	data := &ServiceData{
		PackageName: "api",
		Services: []ServiceInfo{
			{
				Name: "TestService",
				Methods: []MethodInfo{
					{
						Name:       "GetData",
						InputType:  "GetDataRequest",
						OutputType: "GetDataResponse",
						HTTPRules: []parser.HTTPRule{
							{Method: "GET", Pattern: "/data/:id", PathParams: []string{"id"}},
						},
					},
				},
			},
		},
	}

	code, err := g.GenerateCode(data)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Check RouteGroup-specific patterns
	routeGroupPatterns := []string{
		"type RouteGroup struct",
		"func (g *RouteGroup) RegisterGetData",
		"func NewRouter(mux *http.ServeMux)",
		"func DefaultRouter()",
	}

	for _, pattern := range routeGroupPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing RouteGroup pattern: %q", pattern)
		}
	}
}

// TestRegressionMiddlewareSupport ensures middleware functionality is preserved
func TestRegressionMiddlewareSupport(t *testing.T) {
	t.Parallel()
	g := New()

	data := &ServiceData{
		PackageName: "middleware",
		Services: []ServiceInfo{
			{
				Name: "AuthService",
				Methods: []MethodInfo{
					{
						Name:       "Authenticate",
						InputType:  "AuthRequest",
						OutputType: "AuthResponse",
						HTTPRules: []parser.HTTPRule{
							{Method: "POST", Pattern: "/auth", Body: "*"},
						},
					},
				},
			},
		},
	}

	code, err := g.GenerateCode(data)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Check middleware-related patterns
	middlewarePatterns := []string{
		"type Middleware func",
		"middlewares ...Middleware",
	}

	for _, pattern := range middlewarePatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing middleware pattern: %q", pattern)
		}
	}
}

// TestRegressionPathParameterHandling ensures path parameter extraction works
func TestRegressionPathParameterHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern  string
		expected []string
	}{
		{"/users/{id}", []string{"id"}},
		{"/users/{user_id}/posts/{post_id}", []string{"user_id", "post_id"}},
		{"/static/path", []string{}},
		{"/mixed/{id}/static/{name}", []string{"id", "name"}},
	}

	for _, test := range tests {
		result := extractPathParams(test.pattern)
		if len(result) != len(test.expected) {
			t.Errorf("GetPathParams(%q) returned %d params, expected %d",
				test.pattern, len(result), len(test.expected))
			continue
		}

		for i, param := range result {
			if param != test.expected[i] {
				t.Errorf("GetPathParams(%q)[%d] = %q, expected %q",
					test.pattern, i, param, test.expected[i])
			}
		}
	}
}

// TestRegressionOutputFilenames ensures filename generation is preserved
func TestRegressionOutputFilenames(t *testing.T) {
	tests := []struct {
		name         string
		protoFile    string
		options      *Options
		expectedFile string
	}{
		{
			name:         "standard proto file",
			protoFile:    "service.proto",
			options:      &Options{},
			expectedFile: "service_http.pb.go",
		},
		{
			name:         "nested proto file",
			protoFile:    "api/v1/service.proto",
			options:      &Options{},
			expectedFile: "service_http.pb.go",
		},
		{
			name:         "custom prefix",
			protoFile:    "service.proto",
			options:      &Options{OutputPrefix: "generated"},
			expectedFile: "generated_service.pb.go",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			g := New()
			g.Options = test.options

			result := g.getOutputFilename(test.protoFile)
			if result != test.expectedFile {
				t.Errorf("getOutputFilename(%q) = %q, expected %q",
					test.protoFile, result, test.expectedFile)
			}
		})
	}
}

// TestRegressionMultipleHTTPBindings ensures multiple HTTP bindings work
func TestRegressionMultipleHTTPBindings(t *testing.T) {
	t.Parallel()

	// Mock function that returns multiple HTTP bindings
	httpRuleExtractor := func(method *descriptorpb.MethodDescriptorProto) []parser.HTTPRule {
		if method.GetName() == "GetResource" {
			return []parser.HTTPRule{
				{Method: "GET", Pattern: "/v1/resource/{id}", Body: ""},
				{Method: "GET", Pattern: "/resource/{id}", Body: ""},
			}
		}
		return nil
	}

	g := New(httpRuleExtractor)

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("ResourceService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("GetResource"),
						InputType:  proto.String(".test.GetResourceRequest"),
						OutputType: proto.String(".test.Resource"),
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
	if len(method.HTTPRules) != 2 {
		t.Fatalf("Expected 2 HTTP rules, got %d", len(method.HTTPRules))
	}

	expectedPatterns := []string{"/v1/resource/{id}", "/resource/{id}"}
	for i, rule := range method.HTTPRules {
		if rule.Pattern != expectedPatterns[i] {
			t.Errorf("HTTP rule %d pattern = %q, expected %q",
				i, rule.Pattern, expectedPatterns[i])
		}
		if rule.Method != "GET" {
			t.Errorf("HTTP rule %d method = %q, expected %q",
				i, rule.Method, "GET")
		}
	}
}

// TestRegressionServiceFiltering ensures services without HTTP rules are filtered out
func TestRegressionServiceFiltering(t *testing.T) {
	t.Parallel()

	// Mock function that only returns rules for specific methods
	httpRuleExtractor := func(method *descriptorpb.MethodDescriptorProto) []parser.HTTPRule {
		if method.GetName() == "HTTPMethod" {
			return []parser.HTTPRule{{Method: "GET", Pattern: "/test", Body: ""}}
		}
		return nil // No HTTP rules for other methods
	}

	g := New(httpRuleExtractor)

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("test"),
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("HTTPService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: proto.String("HTTPMethod")},
				},
			},
			{
				Name: proto.String("GRPCOnlyService"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: proto.String("GRPCMethod")},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	// Only the service with HTTP methods should be included
	if len(data.Services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(data.Services))
	}

	if data.Services[0].Name != "HTTPService" {
		t.Errorf("Expected service name 'HTTPService', got %q", data.Services[0].Name)
	}
}

// TestRegressionErrorMessages ensures error messages are preserved
func TestRegressionErrorMessages(t *testing.T) {
	g := New()

	// Test parameter parsing error
	req := &pluginpb.CodeGeneratorRequest{
		Parameter: proto.String("paths=invalid_value"),
	}

	resp := g.Generate(req)

	if resp.Error == nil {
		t.Fatal("Expected error for invalid parameter")
	}

	errorMsg := *resp.Error
	if !strings.Contains(errorMsg, "invalid options") {
		t.Errorf("Error message should contain 'invalid options', got: %s", errorMsg)
	}
}

// TestRegressionPackageNameGeneration ensures package name logic is preserved
func TestRegressionPackageNameGeneration(t *testing.T) {
	g := New()

	// Test cases that were working before protobuf 3 migration
	testCases := []struct {
		name     string
		file     *descriptorpb.FileDescriptorProto
		expected string
	}{
		{
			name: "simple package",
			file: &descriptorpb.FileDescriptorProto{
				Package: proto.String("simple"),
			},
			expected: "simple",
		},
		{
			name: "versioned package",
			file: &descriptorpb.FileDescriptorProto{
				Package: proto.String("api.v1"),
			},
			expected: "apiv1",
		},
		{
			name: "go_package override",
			file: &descriptorpb.FileDescriptorProto{
				Package: proto.String("original.pkg"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("example.com/custom;custompkg"),
				},
			},
			expected: "custompkg",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := g.getPackageName(tc.file)
			if result != tc.expected {
				t.Errorf("getPackageName() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

// TestRegressionTemplateExecution ensures template execution produces consistent output
func TestRegressionTemplateExecution(t *testing.T) {
	g := New()

	// Use the exact same data structure as before migration
	data := &ServiceData{
		PackageName: "testservice",
		Services: []ServiceInfo{
			{
				Name: "EchoService",
				Methods: []MethodInfo{
					{
						Name:       "Echo",
						InputType:  "EchoRequest",
						OutputType: "EchoResponse",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "POST",
								Pattern:    "/echo",
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
		t.Fatalf("Template execution failed: %v", err)
	}

	// Verify key template outputs that should be consistent
	expectedOutputs := []string{
		"package testservice",
		"type EchoServiceHandler interface {",
		"HandleEcho(w http.ResponseWriter, r *http.Request)",
		"func RegisterEchoServiceRoutes(r Routes, handler EchoServiceHandler)",
		"func RegisterEchoRoute(r Routes, handler EchoServiceHandler, middlewares ...Middleware)",
		"func (g *RouteGroup) RegisterEcho(handler EchoServiceHandler, middlewares ...Middleware)",
		"r.HandleFunc(http.MethodPost, \"/echo\", handler.HandleEcho, middlewares...)",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code missing expected output: %q", expected)
		}
	}
}
