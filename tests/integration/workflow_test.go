package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "tests/internal"
)

// TestWorkflow_FullPipeline tests the complete workflow from options to generated code
func TestWorkflow_FullPipeline(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		packageName     string
		serviceName     string
		options         string
		expectFileMatch string
	}{
		{
			name:            "default_generation",
			packageName:     "workflowv1",
			serviceName:     "WorkflowService",
			options:         "",
			expectFileMatch: "_http.pb.go",
		},
		{
			name:            "custom_prefix",
			packageName:     "apiv2",
			serviceName:     "APIService",
			options:         "output_prefix=service",
			expectFileMatch: "service_",
		},
		{
			name:            "source_relative",
			packageName:     "internalcore",
			serviceName:     "CoreService",
			options:         "paths=source_relative",
			expectFileMatch: "_http.pb.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Skip if dependencies not available
			if !internal.HasProtoc() {
				t.Skip("protoc not available")
			}

			pluginPath, err := internal.FindPluginBinary()
			if err != nil {
				t.Skip("Plugin binary not found")
			}

			// Create temporary directory
			tmpDir := t.TempDir()

			// Generate proto file
			protoPath := filepath.Join(tmpDir, "test.proto")
			err = internal.GenerateBasicServiceProto(protoPath, tc.packageName, tc.serviceName)
			if err != nil {
				t.Fatalf("Failed to generate proto: %v", err)
			}

			// Run protoc
			outputArg := "--go-http-server-interface_out=" + tmpDir
			if tc.options != "" {
				outputArg = "--go-http-server-interface_out=" + tc.options + ":" + tmpDir
			}

			err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "test.proto")
			if err != nil {
				t.Fatalf("protoc execution failed: %v", err)
			}

			// Verify output files
			outputFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.go"))
			if err != nil {
				t.Fatalf("Failed to list output files: %v", err)
			}

			if len(outputFiles) == 0 {
				t.Fatal("No output files generated")
			}

			// Check filename pattern
			foundMatch := false
			for _, file := range outputFiles {
				if strings.Contains(filepath.Base(file), tc.expectFileMatch) {
					foundMatch = true
					break
				}
			}

			if !foundMatch {
				t.Errorf("Expected filename pattern %q not found in: %v",
					tc.expectFileMatch, outputFiles)
			}

			// Verify generated content
			content, err := os.ReadFile(outputFiles[0])
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			generatedCode := string(content)

			// Basic content verification
			expectedPatterns := []string{
				"package " + tc.packageName,
				tc.serviceName + "Handler interface",
				"Register" + tc.serviceName + "Routes",
			}

			for _, pattern := range expectedPatterns {
				if !strings.Contains(generatedCode, pattern) {
					t.Errorf("Generated code missing pattern: %q", pattern)
				}
			}
		})
	}
}

// TestWorkflow_MultiService tests multi-service generation workflow
func TestWorkflow_MultiService(t *testing.T) {
	t.Parallel()
	internal.RunMultiServiceWorkflowTest(t)
}

// TestWorkflow_RealWorldScenario tests a complete e-commerce scenario
func TestWorkflow_RealWorldScenario(t *testing.T) {
	t.Parallel()
	testMicroserviceScenario(t, "Real-world scenario")
}
