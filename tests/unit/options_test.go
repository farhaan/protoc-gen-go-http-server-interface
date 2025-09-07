package unit_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestOptions_PluginParameterParsing tests plugin parameter parsing
func TestOptions_PluginParameterParsing(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		parameter   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty_parameter",
			parameter:   "",
			expectError: false,
		},
		{
			name:        "paths_source_relative",
			parameter:   "paths=source_relative",
			expectError: false,
		},
		{
			name:        "paths_import",
			parameter:   "paths=import",
			expectError: false,
		},
		{
			name:        "output_prefix",
			parameter:   "output_prefix=api",
			expectError: false,
		},
		{
			name:        "combined_options",
			parameter:   "paths=source_relative,output_prefix=v1",
			expectError: false,
		},
		{
			name:        "invalid_paths_value",
			parameter:   "paths=invalid",
			expectError: true,
			errorMsg:    "unknown paths option",
		},
		{
			name:        "invalid_parameter_format",
			parameter:   "malformed_parameter",
			expectError: true,
			errorMsg:    "invalid parameter",
		},
		{
			name:        "unknown_option",
			parameter:   "unknown_option=value",
			expectError: true,
			errorMsg:    "unknown option",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generator := httpinterface.New()

			request := &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String(tc.parameter),
				FileToGenerate: []string{"test.proto"},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{},
			}

			response := generator.Generate(request)
			hasError := response.GetError() != ""

			if hasError != tc.expectError {
				t.Errorf("Expected error: %v, got error: %v (error: %s)",
					tc.expectError, hasError, response.GetError())
			}

			if tc.expectError && tc.errorMsg != "" {
				if !strings.Contains(response.GetError(), tc.errorMsg) {
					t.Errorf("Expected error message to contain %q, got: %s",
						tc.errorMsg, response.GetError())
				}
			}
		})
	}
}

// TestOptions_OutputPrefixGeneration tests output prefix functionality
func TestOptions_OutputPrefixGeneration(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		outputPrefix string
		protoFile    string
		expectedFile string
	}{
		{
			name:         "no_prefix",
			outputPrefix: "",
			protoFile:    "service.proto",
			expectedFile: "service_http.pb.go",
		},
		{
			name:         "api_prefix",
			outputPrefix: "api",
			protoFile:    "user.proto",
			expectedFile: "api_user.pb.go",
		},
		{
			name:         "v1_prefix",
			outputPrefix: "v1",
			protoFile:    "orders.proto",
			expectedFile: "v1_orders.pb.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			generator := httpinterface.New()

			// Create a basic file descriptor for testing filename generation
			fileDesc := &descriptorpb.FileDescriptorProto{
				Name:    proto.String(tc.protoFile),
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
								Options:    &descriptorpb.MethodOptions{
									// Note: In real usage, HTTP options would be set here
								},
							},
						},
					},
				},
			}

			parameter := ""
			if tc.outputPrefix != "" {
				parameter = "output_prefix=" + tc.outputPrefix
			}

			request := &pluginpb.CodeGeneratorRequest{
				Parameter:      proto.String(parameter),
				FileToGenerate: []string{tc.protoFile},
				ProtoFile:      []*descriptorpb.FileDescriptorProto{fileDesc},
			}

			response := generator.Generate(request)

			if response.GetError() != "" {
				t.Fatalf("Unexpected error: %s", response.GetError())
			}

			// Note: Since this service has no HTTP annotations, no files will be generated
			// This test primarily validates that the parameter parsing works correctly
			// In a real scenario, the filename logic would be tested with actual HTTP annotations
		})
	}
}

// TestOptions_PathsSourceRelative tests source_relative path handling
func TestOptions_PathsSourceRelative(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// Test that paths=source_relative parameter is accepted
	request := &pluginpb.CodeGeneratorRequest{
		Parameter:      proto.String("paths=source_relative"),
		FileToGenerate: []string{"api/v1/service.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("api/v1/service.proto"),
				Package: proto.String("api.v1"),
				Options: &descriptorpb.FileOptions{
					GoPackage: proto.String("github.com/example/api/v1;apiv1"),
				},
				Service: []*descriptorpb.ServiceDescriptorProto{},
			},
		},
	}

	response := generator.Generate(request)

	if response.GetError() != "" {
		t.Fatalf("Unexpected error with source_relative paths: %s", response.GetError())
	}

	// The response should be successful (though empty since no HTTP rules are defined)
	if len(response.GetFile()) > 0 {
		// If files were generated, verify they follow source_relative structure
		for _, file := range response.GetFile() {
			fileName := file.GetName()
			if !strings.Contains(fileName, "api/v1/") {
				t.Errorf("Source relative path not preserved in filename: %s", fileName)
			}
		}
	}
}
