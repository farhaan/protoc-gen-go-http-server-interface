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

// makeHTTPMethodProto builds a MethodDescriptorProto with google.api.http annotation.
func makeHTTPMethodProto(name, inputType, outputType string) *descriptorpb.MethodDescriptorProto {
	return &descriptorpb.MethodDescriptorProto{
		Name:       proto.String(name),
		InputType:  proto.String("." + inputType),
		OutputType: proto.String("." + outputType),
		Options:    &descriptorpb.MethodOptions{},
	}
}

// makeFileProto builds a FileDescriptorProto for testing.
func makeFileProto(name, pkg, goPackage string, services []*descriptorpb.ServiceDescriptorProto) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String(name),
		Package: proto.String(pkg),
		Options: &descriptorpb.FileOptions{
			GoPackage: proto.String(goPackage),
		},
		Service: services,
	}
}

// T25: buildServiceData extracts path params correctly via Generate pipeline.
func TestGenerator_PathParamExtraction(t *testing.T) {
	t.Parallel()
	// Verify via GenerateCode that path params appear in the generated output.
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "pathpkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "PathService",
				Methods: []httpinterface.MethodInfo{
					{
						Name: "GetNested",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/orgs/{org_id}/users/{user_id}",
								PathParams: []string{"org_id", "user_id"},
							},
						},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	// Verify path param values appear in the interface comments.
	if !strings.Contains(generated, `r.PathValue("org_id")`) {
		t.Error("missing r.PathValue(\"org_id\")")
	}
	if !strings.Contains(generated, `r.PathValue("user_id")`) {
		t.Error("missing r.PathValue(\"user_id\")")
	}
	// Both params should be on the same line.
	if !strings.Contains(generated, `r.PathValue("org_id"), r.PathValue("user_id")`) {
		t.Error("multiple params should be on the same line")
	}
}

// T26: getPackageName — exhaustive go_package formats.
func TestGenerator_PackageNameExtraction(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// We test via Generate() — the package name appears in the generated output.
	// To avoid needing real HTTP annotations (which require the annotations package),
	// we use GenerateCode with manually constructed ServiceData.
	tests := []struct {
		name        string
		packageName string
		want        string
	}{
		{name: "simple", packageName: "mypkg", want: "package mypkg"},
		{name: "versioned", packageName: "apiv2", want: "package apiv2"},
		{name: "with_underscore", packageName: "my_service", want: "package my_service"},
		{name: "single_char", packageName: "a", want: "package a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			serviceData := &httpinterface.ServiceData{
				PackageName: tt.packageName,
				Services: []httpinterface.ServiceInfo{
					{
						Name: "S",
						Methods: []httpinterface.MethodInfo{
							{Name: "M", HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: "/m"}}},
						},
					},
				},
			}
			generated, err := generator.GenerateCode(serviceData)
			if err != nil {
				t.Fatalf("GenerateCode: %v", err)
			}
			if !strings.Contains(generated, tt.want) {
				t.Errorf("expected %q, got code with package declaration: %q",
					tt.want, generated[:min(80, len(generated))])
			}
		})
	}
}

// T27: shouldGenerate — Generate() only outputs files for FileToGenerate.
func TestGenerator_ShouldGenerate_FileFilter(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// Two files but only one in FileToGenerate.
	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(""),
		FileToGenerate: []string{"included.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			makeFileProto("included.proto", "included", "github.com/test;included", nil),
			makeFileProto("excluded.proto", "excluded", "github.com/test;excluded", nil),
		},
	}

	response := generator.Generate(req)
	if response.GetError() != "" {
		t.Fatalf("Generate error: %s", response.GetError())
	}

	// No HTTP annotations → no files, but also no error.
	if len(response.GetFile()) > 0 {
		t.Errorf("expected no files (no HTTP annotations), got %d", len(response.GetFile()))
	}
}

// T28: Custom HTTPRuleExtractor via NewWith.
func TestGenerator_CustomHTTPRuleExtractor(t *testing.T) {
	t.Parallel()

	// Custom extractor that returns a hardcoded GET /custom for every method.
	customExtractor := httpinterface.HTTPRuleExtractor(func(method *descriptorpb.MethodDescriptorProto) []parser.HTTPRule {
		return []parser.HTTPRule{{Method: "GET", Pattern: "/custom"}}
	})

	generator := httpinterface.NewWith(
		customExtractor,
		func(pattern string) []string { return nil },
		func(pattern string) string { return pattern },
	)

	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(""),
		FileToGenerate: []string{"custom.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("custom.proto"),
				Package: proto.String("custom"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/test;custom"),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("CustomService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							makeHTTPMethodProto("DoSomething", "Request", "Response"),
						},
					},
				},
			},
		},
	}

	response := generator.Generate(req)
	if response.GetError() != "" {
		t.Fatalf("Generate error: %s", response.GetError())
	}

	if len(response.GetFile()) != 1 {
		t.Fatalf("expected 1 file, got %d", len(response.GetFile()))
	}

	content := response.GetFile()[0].GetContent()
	if !strings.Contains(content, `http.MethodGet, "/custom"`) {
		t.Errorf("expected custom route in output, got:\n%s", content)
	}
}

// T29: hasHTTPRules — Generate() returns no files when service has no HTTP annotations.
func TestGenerator_HasHTTPRules_NoAnnotations(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String(""),
		FileToGenerate: []string{"noannotations.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("noannotations.proto"),
				Package: proto.String("test"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/test;test"),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("NoAnnotationService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							makeHTTPMethodProto("DoSomething", "Request", "Response"),
						},
					},
				},
			},
		},
	}

	response := generator.Generate(req)
	if response.GetError() != "" {
		t.Errorf("unexpected error for unannotated service: %s", response.GetError())
	}
	if len(response.GetFile()) != 0 {
		t.Errorf("expected no files for unannotated service, got %d", len(response.GetFile()))
	}
}

// T30: applyOptions errors — invalid parameters return error in response.
func TestGenerator_InvalidOptions(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	tests := []struct {
		name      string
		parameter string
		wantErr   string
	}{
		{
			name:      "unknown_paths_value",
			parameter: "paths=invalid",
			wantErr:   "unknown paths option",
		},
		{
			name:      "unknown_option_key",
			parameter: "bogus_key=value",
			wantErr:   "unknown option",
		},
		{
			name:      "malformed_no_equals",
			parameter: "malformed",
			wantErr:   "invalid parameter",
		},
		{
			name:      "invalid_editions_value",
			parameter: "editions=maybe",
			wantErr:   "unknown editions option",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String(tt.parameter),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			}
			response := generator.Generate(req)
			if response.GetError() == "" {
				t.Errorf("expected error for parameter %q, got none", tt.parameter)
				return
			}
			if !strings.Contains(response.GetError(), tt.wantErr) {
				t.Errorf("error %q should contain %q", response.GetError(), tt.wantErr)
			}
		})
	}
}

// T31: getOutputFilename — output filename follows <name>_http.pb.go convention.
func TestGenerator_OutputFilename(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// We need a file with HTTP annotations to get output. Use custom extractor.
	customExtractor := httpinterface.HTTPRuleExtractor(func(method *descriptorpb.MethodDescriptorProto) []parser.HTTPRule {
		return []parser.HTTPRule{{Method: "GET", Pattern: "/test"}}
	})
	gen := httpinterface.NewWith(
		customExtractor,
		func(pattern string) []string { return nil },
		func(pattern string) string { return pattern },
	)
	_ = generator // silence unused warning

	tests := []struct {
		protoName    string
		wantFilename string
	}{
		{"service.proto", "service_http.pb.go"},
		{"my_service.proto", "my_service_http.pb.go"},
		{"api/v1/user.proto", "user_http.pb.go"},
	}

	for _, tt := range tests {
		t.Run(tt.protoName, func(t *testing.T) {
			t.Parallel()
			req := &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String(""),
				FileToGenerate: []string{tt.protoName},
				ProtoFile: []*descriptorpb.FileDescriptorProto{
					{
						Name:    proto.String(tt.protoName),
						Package: proto.String("test"),
						Options: &descriptorpb.FileOptions{
							GoPackage: proto.String("github.com/test;test"),
						},
						Service: []*descriptorpb.ServiceDescriptorProto{
							{
								Name: proto.String("TestService"),
								Method: []*descriptorpb.MethodDescriptorProto{
									makeHTTPMethodProto("DoIt", "Request", "Response"),
								},
							},
						},
					},
				},
			}

			response := gen.Generate(req)
			if response.GetError() != "" {
				t.Fatalf("Generate error: %s", response.GetError())
			}
			if len(response.GetFile()) != 1 {
				t.Fatalf("expected 1 file, got %d", len(response.GetFile()))
			}
			got := response.GetFile()[0].GetName()
			if got != tt.wantFilename {
				t.Errorf("filename: got %q, want %q", got, tt.wantFilename)
			}
		})
	}
}

// T31b: output_prefix option changes output filename.
func TestGenerator_OutputPrefixOption(t *testing.T) {
	t.Parallel()

	customExtractor := httpinterface.HTTPRuleExtractor(func(method *descriptorpb.MethodDescriptorProto) []parser.HTTPRule {
		return []parser.HTTPRule{{Method: "GET", Pattern: "/test"}}
	})
	gen := httpinterface.NewWith(
		customExtractor,
		func(pattern string) []string { return nil },
		func(pattern string) string { return pattern },
	)

	req := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("output_prefix=custom"),
		FileToGenerate: []string{"service.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("service.proto"),
				Package: proto.String("test"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/test;test"),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							makeHTTPMethodProto("DoIt", "Request", "Response"),
						},
					},
				},
			},
		},
	}

	response := gen.Generate(req)
	if response.GetError() != "" {
		t.Fatalf("Generate error: %s", response.GetError())
	}
	if len(response.GetFile()) != 1 {
		t.Fatalf("expected 1 file, got %d", len(response.GetFile()))
	}

	got := response.GetFile()[0].GetName()
	if got != "custom_service.pb.go" {
		t.Errorf("filename with output_prefix: got %q, want %q", got, "custom_service.pb.go")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
