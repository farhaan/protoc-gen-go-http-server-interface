package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	internal "tests/internal"
)

// TestBasicGeneration tests end-to-end code generation
func TestBasicGeneration(t *testing.T) {
	t.Parallel()
	// Check if protoc is available
	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	// Create temporary directory
	tmpDir := t.TempDir()

	// Generate proto file from template
	protoPath := filepath.Join(tmpDir, "test.proto")
	err := internal.GenerateBasicServiceProto(protoPath, "testv1", "TestService")
	if err != nil {
		t.Fatalf("Failed to generate proto file: %v", err)
	}

	// Find our plugin binary
	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Fatalf("Failed to find plugin binary: %v", err)
	}

	// Generate code using protoc
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http-server-interface="+pluginPath,
		"--go-http-server-interface_out="+tmpDir,
		"--proto_path="+tmpDir,
		"--proto_path=../..", // For google/api imports
		"test.proto")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("protoc failed: %v\nStderr: %s", err, stderr.String())
	}

	// Check that output file was generated
	outputFile := filepath.Join(tmpDir, "test_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Verify expected patterns in generated code
	expectedPatterns := []string{
		"package testv1",
		"type TestServiceHandler interface",
		"HandleGetTest(w http.ResponseWriter, r *http.Request)",
		"HandleCreateTest(w http.ResponseWriter, r *http.Request)",
		"RegisterTestServiceRoutes",
		`"GET", "/test/{id}"`,
		`"POST", "/test"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}

	t.Logf("E2E test completed successfully. Generated %d bytes of code", len(generatedCode))
}

// TestAdvancedGeneration tests E2E generation with more complex scenarios
func TestAdvancedGeneration(t *testing.T) {
	t.Parallel()
	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	tmpDir := t.TempDir()

	// Generate advanced service proto from template
	protoPath := filepath.Join(tmpDir, "advanced.proto")
	err := internal.GenerateAdvancedServiceProto(protoPath, "advancedv1", "AdvancedService")
	if err != nil {
		t.Fatalf("Failed to generate advanced proto file: %v", err)
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Fatalf("Failed to find plugin binary: %v", err)
	}

	// Run protoc
	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "advanced.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	// Verify advanced features
	outputFile := filepath.Join(tmpDir, "advanced_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Check for advanced patterns
	advancedPatterns := []string{
		"package advancedv1",
		"AdvancedServiceHandler interface",
		`"GET", "/advanced/resources/{id}"`,
		`"POST", "/advanced/resources"`,
		`"PUT", "/advanced/resources/{id}"`,
		`"PATCH", "/advanced/resources/{id}"`,
		`"DELETE", "/advanced/resources/{id}"`,
	}

	for _, pattern := range advancedPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("Advanced generated code missing expected pattern: %q", pattern)
		}
	}
}

// TestWithOptions tests E2E generation with plugin options
func TestWithOptions(t *testing.T) {
	t.Parallel()
	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	testCases := []struct {
		name            string
		options         string
		expectedPattern string
	}{
		{
			name:            "output_prefix_option",
			options:         "output_prefix=custom",
			expectedPattern: "custom_test.pb.go",
		},
		{
			name:            "source_relative_option",
			options:         "paths=source_relative",
			expectedPattern: "test_http.pb.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()

			protoPath := filepath.Join(tmpDir, "test.proto")
			err := internal.GenerateBasicServiceProto(protoPath, "testv1", "TestService")
			if err != nil {
				t.Fatalf("Failed to generate proto file: %v", err)
			}

			pluginPath, err := internal.FindPluginBinary()
			if err != nil {
				t.Fatalf("Failed to find plugin binary: %v", err)
			}

			outputArg := fmt.Sprintf("--go-http-server-interface_out=%s:%s", tc.options, tmpDir)
			cmd := exec.Command("protoc",
				"--plugin=protoc-gen-go-http-server-interface="+pluginPath,
				outputArg,
				"--proto_path="+tmpDir,
				"--proto_path=../..",
				"test.proto")

			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err = cmd.Run()
			if err != nil {
				t.Fatalf("protoc failed: %v\nStderr: %s", err, stderr.String())
			}

			// Check for expected output file pattern
			files, err := filepath.Glob(filepath.Join(tmpDir, "*.go"))
			if err != nil {
				t.Fatalf("Failed to list generated files: %v", err)
			}

			found := false
			for _, file := range files {
				if strings.Contains(filepath.Base(file), tc.expectedPattern) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected file pattern %q not found in generated files: %v", tc.expectedPattern, files)
			}
		})
	}
}
