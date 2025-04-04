package httpinterface

import (
	"strings"
	"testing"
	"text/template"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"google.golang.org/protobuf/proto"
)

// Test New function
func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.ParsedTemplates == nil {
		t.Error("ParsedTemplates is nil")
	}

	// Check if templates were parsed
	if g.ParsedTemplates.Lookup("header") == nil {
		t.Error("header template not found")
	}
	if g.ParsedTemplates.Lookup("service") == nil {
		t.Error("service template not found")
	}
}

// Test shouldGenerate function
func TestShouldGenerate(t *testing.T) {
	g := New()

	tests := []struct {
		file            string
		filesToGenerate []string
		expected        bool
	}{
		{"foo.proto", []string{"foo.proto", "bar.proto"}, true},
		{"baz.proto", []string{"foo.proto", "bar.proto"}, false},
		{"foo.proto", []string{}, false},
	}

	for _, test := range tests {
		result := g.shouldGenerate(test.file, test.filesToGenerate)
		if result != test.expected {
			t.Errorf("shouldGenerate(%q, %v) = %v, want %v", test.file, test.filesToGenerate, result, test.expected)
		}
	}
}

// Mock GetHTTPRules function for testing
func mockGetHTTPRules(method *descriptor.MethodDescriptorProto) []HTTPRule {
	// For test purposes, if the method name contains "HTTP", return a rule
	if strings.Contains(method.GetName(), "HTTP") {
		return []HTTPRule{
			{
				Method:  "GET",
				Pattern: "/api/{id}",
				Body:    "",
			},
		}
	}
	return nil
}

// Test hasHTTPRules function
func TestHasHTTPRules(t *testing.T) {
	// Save the original function and restore it after the test
	originalGetHTTPRules := GetHTTPRules
	GetHTTPRules = mockGetHTTPRules
	defer func() { GetHTTPRules = originalGetHTTPRules }()

	g := New()

	// Create mock file with services
	file := &descriptor.FileDescriptorProto{
		Service: []*descriptor.ServiceDescriptorProto{
			{
				Name: proto.String("ServiceWithHTTP"),
				Method: []*descriptor.MethodDescriptorProto{
					{
						Name: proto.String("MethodWithHTTP"),
					},
				},
			},
			{
				Name: proto.String("ServiceWithoutHTTP"),
				Method: []*descriptor.MethodDescriptorProto{
					{
						Name: proto.String("PlainMethod"),
					},
				},
			},
		},
	}

	if !g.hasHTTPRules(file) {
		t.Error("hasHTTPRules() should return true for file with HTTP rules")
	}

	// Remove the service with HTTP rules
	file.Service = file.Service[1:]

	if g.hasHTTPRules(file) {
		t.Error("hasHTTPRules() should return false for file without HTTP rules")
	}
}

// Mock GetPathParams and ConvertPathPattern for testing
func mockGetPathParams(pattern string) []string {
	if pattern == "/api/{id}" {
		return []string{"id"}
	}
	return nil
}

func mockConvertPathPattern(pattern string) string {
	if pattern == "/api/{id}" {
		return "/api/:id"
	}
	return pattern
}

// Test buildServiceData function
func TestBuildServiceData(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	originalGetPathParams := GetPathParams
	originalConvertPathPattern := ConvertPathPattern

	GetHTTPRules = mockGetHTTPRules
	GetPathParams = mockGetPathParams
	ConvertPathPattern = mockConvertPathPattern

	defer func() {
		GetHTTPRules = originalGetHTTPRules
		GetPathParams = originalGetPathParams
		ConvertPathPattern = originalConvertPathPattern
	}()

	g := New()

	// Create mock file
	file := &descriptor.FileDescriptorProto{
		Package: proto.String("test"),
		Service: []*descriptor.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptor.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.Request"),
						OutputType: proto.String(".test.Response"),
					},
					{
						Name:       proto.String("PlainMethod"),
						InputType:  proto.String(".test.Request"),
						OutputType: proto.String(".test.Response"),
					},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if data.PackageName != "test" {
		t.Errorf("PackageName = %q, want %q", data.PackageName, "test")
	}

	if len(data.Services) != 1 {
		t.Fatalf("len(Services) = %d, want %d", len(data.Services), 1)
	}

	service := data.Services[0]
	if service.Name != "TestService" {
		t.Errorf("Service.Name = %q, want %q", service.Name, "TestService")
	}

	if len(service.Methods) != 1 {
		t.Fatalf("len(Methods) = %d, want %d", len(service.Methods), 1)
	}

	method := service.Methods[0]
	if method.Name != "MethodWithHTTP" {
		t.Errorf("Method.Name = %q, want %q", method.Name, "MethodWithHTTP")
	}

	if method.InputType != "Request" {
		t.Errorf("Method.InputType = %q, want %q", method.InputType, "Request")
	}

	if method.OutputType != "Response" {
		t.Errorf("Method.OutputType = %q, want %q", method.OutputType, "Response")
	}

	if len(method.HTTPRules) != 1 {
		t.Fatalf("len(HTTPRules) = %d, want %d", len(method.HTTPRules), 1)
	}

	rule := method.HTTPRules[0]
	if rule.Method != "GET" {
		t.Errorf("Rule.Method = %q, want %q", rule.Method, "GET")
	}

	if rule.Pattern != "/api/:id" {
		t.Errorf("Rule.Pattern = %q, want %q", rule.Pattern, "/api/:id")
	}

	if len(rule.PathParams) != 1 || rule.PathParams[0] != "id" {
		t.Errorf("Rule.PathParams = %v, want %v", rule.PathParams, []string{"id"})
	}
}

// Test generateCode function
func TestGenerateCode(t *testing.T) {
	g := New()

	// Create simple service data
	data := &ServiceData{
		PackageName: "test",
		Services: []ServiceInfo{
			{
				Name: "TestService",
				Methods: []MethodInfo{
					{
						Name:       "GetItem",
						InputType:  "GetItemRequest",
						OutputType: "GetItemResponse",
						HTTPRules: []HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/items/:id",
								PathParams: []string{"id"},
							},
						},
					},
				},
			},
		},
	}

	code, err := g.generateCode(data)
	if err != nil {
		t.Fatalf("generateCode() error = %v", err)
	}

	// Check for expected content in the generated code
	expectedContents := []string{
		"package test",
		"type TestServiceHandler interface",
		"HandleGetItem",
		"func RegisterGetItemRoute",
		// Check for updated content related to the new ServeMux parameter
		"func NewRouter(mux *http.ServeMux)",
		"func DefaultRouter()",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code doesn't contain %q", expected)
		}
	}
}

// Test outputFilename method with different options
func TestOutputFilename(t *testing.T) {
	tests := []struct {
		name          string
		protoFilename string
		options       *Options
		want          string
	}{
		{
			name:          "default options",
			protoFilename: "service.proto",
			options:       &Options{},
			want:          "service_http.pb.go",
		},
		{
			name:          "with output prefix",
			protoFilename: "service.proto",
			options:       &Options{OutputPrefix: "api"},
			want:          "api_service.pb.go",
		},
		{
			name:          "nested proto file with default options",
			protoFilename: "api/v1/service.proto",
			options:       &Options{},
			want:          "service_http.pb.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()
			g.Options = tt.options

			got := g.outputFilename(tt.protoFilename)
			if got != tt.want {
				t.Errorf("outputFilename(%q) = %q, want %q", tt.protoFilename, got, tt.want)
			}
		})
	}
}

// Test getPackageName function
func TestGetPackageName(t *testing.T) {
	g := New()

	tests := []struct {
		name        string
		protoFile   *descriptor.FileDescriptorProto
		wantPackage string
	}{
		// Basic cases
		{
			name: "simple proto package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("test"),
			},
			wantPackage: "test",
		},
		{
			name: "empty proto package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String(""),
			},
			wantPackage: "",
		},

		// Proto package versioning patterns
		{
			name: "standard versioned package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("oauth.v1"),
			},
			wantPackage: "oauthv1",
		},
		{
			name: "version with beta suffix",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("oauth.v1beta1"),
			},
			wantPackage: "oauthv1beta1",
		},
		{
			name: "version with alpha suffix",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("oauth.v1alpha1"),
			},
			wantPackage: "oauthv1alpha1",
		},

		// Complex package hierarchies
		{
			name: "deep package hierarchy",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("company.department.service.feature.v1"),
			},
			wantPackage: "featurev1",
		},
		{
			name: "typical microservice package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.core.oauth.v1"),
			},
			wantPackage: "oauthv1",
		},
		{
			name: "internal service package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("internal.auth.oauth.v1"),
			},
			wantPackage: "oauthv1",
		},

		// go_package option variations
		{
			name: "go_package with explicit name",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.core.oauth.v1"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("example.com/api/oauth/v1;oauthv1"),
				},
			},
			wantPackage: "oauthv1",
		},
		{
			name: "go_package with multiple path segments",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.core.oauth.v1"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("example.com/internal/api/core/oauth/v1"),
				},
			},
			wantPackage: "v1",
		},
		{
			name: "go_package with organization prefix",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.core.oauth.v1"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("github.com/organization/project/api/oauth;oauthapi"),
				},
			},
			wantPackage: "oauthapi",
		},

		// Special cases and edge cases
		{
			name: "dots in service name",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("auth.oauth2.v1"),
			},
			wantPackage: "oauth2v1",
		},
		{
			name: "numbers in package segments",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api2.service3.v1"),
			},
			wantPackage: "service3v1",
		},
		{
			name: "very short segments",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("a.b.c.v1"),
			},
			wantPackage: "cv1",
		},
		{
			name: "repeated segments",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("oauth.oauth.v1"),
			},
			wantPackage: "oauthv1",
		},

		// Go module style package paths
		{
			name: "module path with version",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.oauth.v1"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("example.com/api/v1;apiv1"),
				},
			},
			wantPackage: "apiv1",
		},
		{
			name: "module subpath package",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("api.oauth.v1"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("example.com/api/oauth/internal/v1;oauthv1"),
				},
			},
			wantPackage: "oauthv1",
		},

		// Enterprise patterns
		{
			name: "enterprise service with region",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("enterprise.eu.oauth.v1"),
			},
			wantPackage: "oauthv1",
		},
		{
			name: "enterprise service with product",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("enterprise.product.oauth.v1"),
			},
			wantPackage: "oauthv1",
		},

		// Validation cases
		{
			name: "unusual version format",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("service.version1"),
			},
			wantPackage: "serviceversion1",
		},
		{
			name: "single letter segments",
			protoFile: &descriptor.FileDescriptorProto{
				Package: proto.String("a.b.c.d"),
			},
			wantPackage: "cd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPackage := g.getPackageName(tt.protoFile)
			if gotPackage != tt.wantPackage {
				t.Errorf("getPackageName() = %q, want %q", gotPackage, tt.wantPackage)
			}
		})
	}
}

// Test getTypeName function
func TestGetTypeName(t *testing.T) {
	g := New()

	tests := []struct {
		input    string
		expected string
	}{
		{".test.Request", "Request"},
		{"Request", "Request"},
		{".com.example.foo.Bar", "Bar"},
	}

	for _, test := range tests {
		result := g.getTypeName(test.input)
		if result != test.expected {
			t.Errorf("getTypeName(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

// Test the full Generate function
func TestGenerate(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	originalGetPathParams := GetPathParams
	originalConvertPathPattern := ConvertPathPattern

	GetHTTPRules = mockGetHTTPRules
	GetPathParams = mockGetPathParams
	ConvertPathPattern = mockConvertPathPattern

	defer func() {
		GetHTTPRules = originalGetHTTPRules
		GetPathParams = originalGetPathParams
		ConvertPathPattern = originalConvertPathPattern
	}()

	g := New()

	// Create a mock CodeGeneratorRequest
	req := &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("MethodWithHTTP"),
								InputType:  proto.String(".test.Request"),
								OutputType: proto.String(".test.Response"),
							},
						},
					},
				},
			},
		},
	}

	resp := g.Generate(req)

	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Error != nil {
		t.Fatalf("Generate() returned error: %s", *resp.Error)
	}

	if len(resp.File) != 1 {
		t.Fatalf("len(resp.File) = %d, want %d", len(resp.File), 1)
	}

	file := resp.File[0]
	if file.GetName() != "test_http.pb.go" {
		t.Errorf("File.Name = %q, want %q", file.GetName(), "test_http.pb.go")
	}

	// Check for expected content in the generated file
	expectedContents := []string{
		"package test",
		"type TestServiceHandler interface",
		"HandleMethodWithHTTP",
		// Check for new shared mux-related content
		"func NewRouter(mux *http.ServeMux)",
		"func DefaultRouter()",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(file.GetContent(), expected) {
			t.Errorf("Generated file doesn't contain %q", expected)
		}
	}
}

// Test the Generate function with no services that have HTTP rules
func TestGenerateNoHTTPRules(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	GetHTTPRules = func(method *descriptor.MethodDescriptorProto) []HTTPRule { return nil }
	defer func() { GetHTTPRules = originalGetHTTPRules }()

	g := New()

	req := &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("PlainMethod"),
								InputType:  proto.String(".test.Request"),
								OutputType: proto.String(".test.Response"),
							},
						},
					},
				},
			},
		},
	}

	resp := g.Generate(req)

	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Error != nil {
		t.Fatalf("Generate() returned error: %s", *resp.Error)
	}

	if len(resp.File) != 0 {
		t.Errorf("len(resp.File) = %d, want %d", len(resp.File), 0)
	}
}

// Test error case for generateCode
func TestGenerateCodeError(t *testing.T) {
	// Create a generator with a bad template to force an error
	g := New()
	g.ParsedTemplates = g.ParsedTemplates.New("header")
	g.ParsedTemplates = template.Must(g.ParsedTemplates.Parse("{{.UndefinedField}}")) // This will cause an error

	data := &ServiceData{
		PackageName: "test",
		Services:    []ServiceInfo{},
	}

	_, err := g.generateCode(data)
	if err == nil {
		t.Error("generateCode() should return an error for invalid template")
	}
}

// TestBuildServiceDataWithOptions tests buildServiceData with options
func TestBuildServiceDataWithOptions(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	originalGetPathParams := GetPathParams
	originalConvertPathPattern := ConvertPathPattern

	GetHTTPRules = mockGetHTTPRules
	GetPathParams = mockGetPathParams
	ConvertPathPattern = mockConvertPathPattern

	defer func() {
		GetHTTPRules = originalGetHTTPRules
		GetPathParams = originalGetPathParams
		ConvertPathPattern = originalConvertPathPattern
	}()

	g := New()

	// Create mock file with options
	file := &descriptor.FileDescriptorProto{
		Package: proto.String("test"),
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("github.com/example/testpkg"),
		},
		Service: []*descriptor.ServiceDescriptorProto{
			{
				Name: proto.String("TestService"),
				Method: []*descriptor.MethodDescriptorProto{
					{
						Name:       proto.String("MethodWithHTTP"),
						InputType:  proto.String(".test.Request"),
						OutputType: proto.String(".test.Response"),
					},
				},
			},
		},
	}

	data := g.buildServiceData(file)

	if data.PackageName != "testpkg" {
		t.Errorf("PackageName = %q, want %q", data.PackageName, "testpkg")
	}
}

// Mock implementation of HTTP-related functions
// var GetHTTPRules = func(method *descriptor.MethodDescriptorProto) []HTTPRule { return nil }
// var GetPathParams = func(pattern string) []string { return nil }
// var ConvertPathPattern = func(pattern string) string { return pattern }

// TestTemplateExecution tests that the template execution produces expected output
func TestTemplateExecution(t *testing.T) {
	g := New()

	// Create a simple service data for testing
	data := &ServiceData{
		PackageName: "testpkg",
		Services: []ServiceInfo{
			{
				Name: "EchoService",
				Methods: []MethodInfo{
					{
						Name:       "Echo",
						InputType:  "EchoRequest",
						OutputType: "EchoResponse",
						HTTPRules: []HTTPRule{
							{
								Method:     "POST",
								Pattern:    "/v1/echo",
								Body:       "*",
								PathParams: []string{},
							},
						},
					},
				},
			},
		},
	}

	// Generate code
	code, err := g.generateCode(data)
	if err != nil {
		t.Fatalf("generateCode() error = %v", err)
	}

	// Check for RouteGroup and RegisterEchoRoute in generated code
	if !strings.Contains(code, "type RouteGroup struct {") {
		t.Error("Generated code doesn't contain RouteGroup struct")
	}

	if !strings.Contains(code, "func RegisterEchoRoute(r Routes, handler EchoServiceHandler, middlewares ...Middleware)") {
		t.Error("Generated code doesn't contain RegisterEchoRoute function")
	}

	if !strings.Contains(code, "func (g *RouteGroup) RegisterEcho(handler EchoServiceHandler, middlewares ...Middleware)") {
		t.Error("Generated code doesn't contain RouteGroup.RegisterEcho method")
	}

	// Check for new shared mux constructor
	if !strings.Contains(code, "func NewRouter(mux *http.ServeMux)") {
		t.Error("Generated code doesn't contain NewRouter function with mux parameter")
	}

	if !strings.Contains(code, "func DefaultRouter()") {
		t.Error("Generated code doesn't contain DefaultRouter function")
	}

	// Check for proper middleware chaining
	if !strings.Contains(code, "Apply route-specific middlewares first") {
		t.Error("Generated code doesn't have proper middleware ordering comments")
	}
}

// TestGenerateWithMultipleFiles tests generating code with multiple files
func TestGenerateWithMultipleFiles(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	GetHTTPRules = mockGetHTTPRules
	defer func() { GetHTTPRules = originalGetHTTPRules }()

	g := New()

	// Create a mock CodeGeneratorRequest with multiple files
	req := &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{"file1.proto", "file2.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("file1.proto"),
				Package: proto.String("pkg1"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("Service1"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("MethodWithHTTP"),
								InputType:  proto.String(".pkg1.Request"),
								OutputType: proto.String(".pkg1.Response"),
							},
						},
					},
				},
			},
			{
				Name:    proto.String("file2.proto"),
				Package: proto.String("pkg2"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("Service2"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("PlainMethod"),
								InputType:  proto.String(".pkg2.Request"),
								OutputType: proto.String(".pkg2.Response"),
							},
						},
					},
				},
			},
		},
	}

	resp := g.Generate(req)

	if resp == nil {
		t.Fatal("Generate() returned nil response")
	}

	if resp.Error != nil {
		t.Fatalf("Generate() returned error: %s", *resp.Error)
	}

	// Only file1.proto should generate code as file2.proto doesn't have HTTP rules
	if len(resp.File) != 1 {
		t.Fatalf("len(resp.File) = %d, want %d", len(resp.File), 1)
	}

	file := resp.File[0]
	if file.GetName() != "file1_http.pb.go" {
		t.Errorf("File.Name = %q, want %q", file.GetName(), "file1_http.pb.go")
	}
}

// Test ParseOptions function
func TestParseOptions(t *testing.T) {
	tests := []struct {
		name           string
		parameter      string
		wantRelative   bool
		wantPrefix     string
		wantErrContain string
	}{
		{
			name:         "empty parameter",
			parameter:    "",
			wantRelative: false,
			wantPrefix:   "",
		},
		{
			name:         "paths=source_relative",
			parameter:    "paths=source_relative",
			wantRelative: true,
			wantPrefix:   "",
		},
		{
			name:         "paths=import",
			parameter:    "paths=import",
			wantRelative: false,
			wantPrefix:   "",
		},
		{
			name:         "output_prefix=custom",
			parameter:    "output_prefix=custom",
			wantRelative: false,
			wantPrefix:   "custom",
		},
		{
			name:         "multiple valid options",
			parameter:    "paths=source_relative,output_prefix=custom",
			wantRelative: true,
			wantPrefix:   "custom",
		},
		{
			name:           "invalid paths value",
			parameter:      "paths=unknown",
			wantErrContain: "unknown paths option",
		},
		{
			name:           "invalid parameter format",
			parameter:      "invalid-format",
			wantErrContain: "invalid parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseOptions(tt.parameter)

			// Check error cases
			if tt.wantErrContain != "" {
				if err == nil {
					t.Errorf("ParseOptions(%q) should have returned an error containing %q", tt.parameter, tt.wantErrContain)
				} else if !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("ParseOptions(%q) error = %v, should contain %q", tt.parameter, err, tt.wantErrContain)
				}
				return
			}

			// Check non-error cases
			if err != nil {
				t.Errorf("ParseOptions(%q) unexpected error: %v", tt.parameter, err)
				return
			}

			if opts.PathsSourceRelative != tt.wantRelative {
				t.Errorf("ParseOptions(%q) PathsSourceRelative = %v, want %v", tt.parameter, opts.PathsSourceRelative, tt.wantRelative)
			}

			if opts.OutputPrefix != tt.wantPrefix {
				t.Errorf("ParseOptions(%q) OutputPrefix = %q, want %q", tt.parameter, opts.OutputPrefix, tt.wantPrefix)
			}
		})
	}
}

// Test Generate function with different options
func TestGenerateWithOptions(t *testing.T) {
	// Save the original functions and restore them after the test
	originalGetHTTPRules := GetHTTPRules
	GetHTTPRules = mockGetHTTPRules
	defer func() { GetHTTPRules = originalGetHTTPRules }()

	tests := []struct {
		name           string
		parameter      string
		fileToGenerate string
		protoFile      *descriptor.FileDescriptorProto
		wantFilename   string
	}{
		{
			name:           "default options",
			parameter:      "",
			fileToGenerate: "test.proto",
			protoFile: &descriptor.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("MethodWithHTTP"),
								InputType:  proto.String(".test.Request"),
								OutputType: proto.String(".test.Response"),
							},
						},
					},
				},
			},
			wantFilename: "test_http.pb.go",
		},
		{
			name:           "paths=source_relative with nested proto",
			parameter:      "paths=source_relative",
			fileToGenerate: "api/v1/test.proto",
			protoFile: &descriptor.FileDescriptorProto{
				Name:    proto.String("api/v1/test.proto"),
				Package: proto.String("test"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("MethodWithHTTP"),
								InputType:  proto.String(".test.Request"),
								OutputType: proto.String(".test.Response"),
							},
						},
					},
				},
			},
			wantFilename: "api/v1/test_http.pb.go",
		},
		{
			name:           "custom output prefix",
			parameter:      "output_prefix=api",
			fileToGenerate: "test.proto",
			protoFile: &descriptor.FileDescriptorProto{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
				Service: []*descriptor.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptor.MethodDescriptorProto{
							{
								Name:       proto.String("MethodWithHTTP"),
								InputType:  proto.String(".test.Request"),
								OutputType: proto.String(".test.Response"),
							},
						},
					},
				},
			},
			wantFilename: "api_test.pb.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New()

			req := &plugin.CodeGeneratorRequest{
				Parameter:      proto.String(tt.parameter),
				FileToGenerate: []string{tt.fileToGenerate},
				ProtoFile:      []*descriptor.FileDescriptorProto{tt.protoFile},
			}

			resp := g.Generate(req)

			if resp.Error != nil {
				t.Fatalf("Generate() returned error: %s", *resp.Error)
			}

			if len(resp.File) != 1 {
				t.Fatalf("len(resp.File) = %d, want 1", len(resp.File))
			}

			file := resp.File[0]
			if file.GetName() != tt.wantFilename {
				t.Errorf("File.Name = %q, want %q", file.GetName(), tt.wantFilename)
			}
		})
	}
}

// Test Generate function with invalid options
func TestGenerateWithInvalidOptions(t *testing.T) {
	g := New()

	req := &plugin.CodeGeneratorRequest{
		Parameter:      proto.String("paths=invalid"),
		FileToGenerate: []string{"test.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{
				Name:    proto.String("test.proto"),
				Package: proto.String("test"),
			},
		},
	}

	resp := g.Generate(req)

	if resp.Error == nil {
		t.Fatal("Generate() should have returned an error for invalid options")
	}

	if !strings.Contains(*resp.Error, "invalid options") {
		t.Errorf("Error = %q, should contain 'invalid options'", *resp.Error)
	}

	// The response should not contain any files when there's an error
	if len(resp.File) != 0 {
		t.Errorf("len(resp.File) = %d, want 0", len(resp.File))
	}
}
