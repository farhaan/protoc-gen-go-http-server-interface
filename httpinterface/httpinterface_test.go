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
	}

	for _, expected := range expectedContents {
		if !strings.Contains(code, expected) {
			t.Errorf("Generated code doesn't contain %q", expected)
		}
	}
}

// Test outputFilename function
func TestOutputFilename(t *testing.T) {
	g := New()

	tests := []struct {
		input    string
		expected string
	}{
		{"foo.proto", "foo_http.pb.go"},
		{"path/to/bar.proto", "bar_http.pb.go"},
		{"baz", "baz_http.pb.go"},
	}

	for _, test := range tests {
		result := g.outputFilename(test.input)
		if result != test.expected {
			t.Errorf("outputFilename(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

// Test getPackageName function
func TestGetPackageName(t *testing.T) {
	g := New()

	tests := []struct {
		file     *descriptor.FileDescriptorProto
		expected string
	}{
		{
			&descriptor.FileDescriptorProto{
				Package: proto.String("test"),
			},
			"test",
		},
		{
			&descriptor.FileDescriptorProto{
				Package: proto.String("foo"),
				Options: &descriptor.FileOptions{
					GoPackage: proto.String("example.com/foo"),
				},
			},
			"foo",
		},
	}

	for i, test := range tests {
		result := g.getPackageName(test.file)
		if result != test.expected {
			t.Errorf("Test %d: getPackageName() = %q, want %q", i, result, test.expected)
		}
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

	// Check for the updated pattern with registeredPaths map
	if !strings.Contains(code, "registeredPaths map[string]bool") {
		t.Error("Generated code doesn't contain registeredPaths map")
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
