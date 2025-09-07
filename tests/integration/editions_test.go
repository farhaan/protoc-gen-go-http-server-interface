package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "tests/internal"
)

// TestEditions_FullPipeline tests the complete workflow for protobuf editions support
func TestEditions_FullPipeline(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		packageName         string
		serviceName         string
		options             string
		expectFileMatch     string
		expectedPackage     string
		expectedServiceName string
	}{
		{
			name:                "editions_basic_generation",
			packageName:         "editionsv1",
			serviceName:         "UserService",
			options:             "",
			expectFileMatch:     "_http.pb.go",
			expectedPackage:     "editionsv1",
			expectedServiceName: "UserServiceHandler",
		},
		{
			name:                "editions_with_source_relative",
			packageName:         "api.editions",
			serviceName:         "EditionsService",
			options:             "paths=source_relative",
			expectFileMatch:     "_http.pb.go",
			expectedPackage:     "editions",
			expectedServiceName: "EditionsServiceHandler",
		},
		{
			name:                "editions_with_output_prefix",
			packageName:         "editions.core",
			serviceName:         "CoreService",
			options:             "output_prefix=generated_",
			expectFileMatch:     "generated_",
			expectedPackage:     "core",
			expectedServiceName: "CoreServiceHandler",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := internal.TestWorkflowConfig{
				Name:                tc.name,
				PackageName:         tc.packageName,
				ServiceName:         tc.serviceName,
				Options:             tc.options,
				ExpectedPackage:     tc.expectedPackage,
				ExpectedServiceName: tc.expectedServiceName,
				ExpectPrefixInFile:  tc.expectFileMatch,
			}

			runEditionsWorkflowTest(t, config)
		})
	}
}

// TestEditions_ComplexBindings tests editions with complex HTTP bindings
func TestEditions_ComplexBindings(t *testing.T) {
	t.Parallel()

	// Skip if dependencies not available
	if !internal.HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	// Generate editions proto file
	protoPath := filepath.Join(tmpDir, "complex_editions.proto")
	err = internal.GenerateEditionsServiceProto(protoPath, "complexeditions", "ComplexService")
	if err != nil {
		t.Fatalf("Failed to generate editions proto: %v", err)
	}

	// Run protoc with our plugin
	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, protoPath)
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	// Verify the generated file exists
	generatedFiles, err := filepath.Glob(filepath.Join(tmpDir, "*_http.pb.go"))
	if err != nil {
		t.Fatalf("Error finding generated files: %v", err)
	}
	if len(generatedFiles) == 0 {
		t.Fatal("No generated files found")
	}

	// Read and verify the generated content
	content, err := os.ReadFile(generatedFiles[0])
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify editions-specific features are properly handled
	expectedContent := []string{
		"package complexeditions",
		"type ComplexServiceHandler interface",
		"HandleCreateUser(w http.ResponseWriter, r *http.Request)",
		"HandleGetUser(w http.ResponseWriter, r *http.Request)",
		"HandleUpdateUser(w http.ResponseWriter, r *http.Request)",
		"HandleDeleteUser(w http.ResponseWriter, r *http.Request)",
		"HandleListUsers(w http.ResponseWriter, r *http.Request)",
		"HandleActivateUser(w http.ResponseWriter, r *http.Request)",
		"HandleBulkUpdateUsers(w http.ResponseWriter, r *http.Request)",
		// Verify multiple HTTP bindings are generated
		`r.HandleFunc("POST", "/v1/users", handler.HandleCreateUser)`,
		`r.HandleFunc("GET", "/v1/users/{user_id}", handler.HandleGetUser)`,
		`r.HandleFunc("PUT", "/v1/users/{user_id}", handler.HandleUpdateUser)`,
		`r.HandleFunc("PATCH", "/v1/users/{user_id}", handler.HandleUpdateUser)`,
		`r.HandleFunc("MERGE", "/v1/users/{user_id}/merge", handler.HandleUpdateUser)`,
		`r.HandleFunc("DELETE", "/v1/users/{user_id}", handler.HandleDeleteUser)`,
		`r.HandleFunc("GET", "/v1/users", handler.HandleListUsers)`,
		`r.HandleFunc("PATCH", "/v1/users/{user_id}/activate", handler.HandleActivateUser)`,
		`r.HandleFunc("BULK", "/v1/users/bulk", handler.HandleBulkUpdateUsers)`,
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated content missing expected part: %q", expected)
		}
	}

	t.Logf("Successfully generated %d bytes of editions code with complex bindings", len(content))
}

// TestEditions_FeatureCompatibility tests that editions support is correctly declared
func TestEditions_FeatureCompatibility(t *testing.T) {
	t.Parallel()

	// Skip if dependencies not available
	if !internal.HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	// Generate a simple editions proto
	protoPath := filepath.Join(tmpDir, "feature_test.proto")
	err = internal.GenerateEditionsServiceProto(protoPath, "featuretest", "FeatureService")
	if err != nil {
		t.Fatalf("Failed to generate editions proto: %v", err)
	}

	// Run protoc - if our plugin doesn't properly declare editions support,
	// protoc should reject the editions proto file
	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, protoPath)
	if err != nil {
		// If protoc fails with editions-related error, our feature flag is wrong
		if strings.Contains(err.Error(), "editions") || strings.Contains(err.Error(), "edition") {
			t.Fatalf("protoc rejected editions proto - plugin may not declare editions support correctly: %v", err)
		}
		// Other errors might be related to missing dependencies
		t.Fatalf("protoc execution failed: %v", err)
	}

	// If we get here, protoc accepted our editions proto file, which means
	// our plugin correctly declared FEATURE_SUPPORTS_EDITIONS
	generatedFiles, err := filepath.Glob(filepath.Join(tmpDir, "*_http.pb.go"))
	if err != nil {
		t.Fatalf("Error finding generated files: %v", err)
	}
	if len(generatedFiles) == 0 {
		t.Fatal("No generated files found - editions support may not be working")
	}

	t.Logf("Editions support correctly declared and working - protoc accepted editions proto")
}

// TestEditions_vs_Proto3_Compatibility tests that editions and proto3 generate equivalent output
func TestEditions_vs_Proto3_Compatibility(t *testing.T) {
	t.Parallel()

	// Skip if dependencies not available
	if !internal.HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	// Generate proto3 version
	proto3Dir := t.TempDir()
	proto3Path := filepath.Join(proto3Dir, "proto3.proto")
	err = internal.GenerateBasicServiceProto(proto3Path, "compatibility", "TestService")
	if err != nil {
		t.Fatalf("Failed to generate proto3 file: %v", err)
	}

	// Generate editions version
	editionsDir := t.TempDir()
	editionsPath := filepath.Join(editionsDir, "editions.proto")
	err = internal.GenerateEditionsServiceProto(editionsPath, "compatibility", "TestService")
	if err != nil {
		t.Fatalf("Failed to generate editions file: %v", err)
	}

	// Generate code for both
	outputArg3 := "--go-http-server-interface_out=" + proto3Dir
	err = internal.RunProtoc(proto3Dir, pluginPath, outputArg3, proto3Path)
	if err != nil {
		t.Fatalf("proto3 generation failed: %v", err)
	}

	outputArgE := "--go-http-server-interface_out=" + editionsDir
	err = internal.RunProtoc(editionsDir, pluginPath, outputArgE, editionsPath)
	if err != nil {
		t.Fatalf("editions generation failed: %v", err)
	}

	// Both should generate valid files
	proto3Files, _ := filepath.Glob(filepath.Join(proto3Dir, "*_http.pb.go"))
	editionsFiles, _ := filepath.Glob(filepath.Join(editionsDir, "*_http.pb.go"))

	if len(proto3Files) == 0 {
		t.Fatal("No proto3 files generated")
	}
	if len(editionsFiles) == 0 {
		t.Fatal("No editions files generated")
	}

	// Read both generated files
	proto3Content, err := os.ReadFile(proto3Files[0])
	if err != nil {
		t.Fatalf("Failed to read proto3 file: %v", err)
	}

	editionsContent, err := os.ReadFile(editionsFiles[0])
	if err != nil {
		t.Fatalf("Failed to read editions file: %v", err)
	}

	// Both should contain similar core structures (allowing for different service definitions)
	commonExpected := []string{
		"package compatibility",
		"type TestServiceHandler interface",
		"RegisterTestServiceRoutes",
		"HandleFunc",
	}

	proto3Str := string(proto3Content)
	editionsStr := string(editionsContent)

	for _, expected := range commonExpected {
		if !strings.Contains(proto3Str, expected) {
			t.Errorf("Proto3 output missing expected part: %q", expected)
		}
		if !strings.Contains(editionsStr, expected) {
			t.Errorf("Editions output missing expected part: %q", expected)
		}
	}

	t.Logf("Both proto3 and editions generated compatible code structures")
}

// runEditionsWorkflowTest is a specialized version of RunWorkflowTest for editions
func runEditionsWorkflowTest(t *testing.T, config internal.TestWorkflowConfig) {
	if !internal.HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	// Generate editions proto file instead of basic proto
	protoPath := filepath.Join(tmpDir, "test_editions.proto")
	err = internal.GenerateEditionsServiceProto(protoPath, config.PackageName, config.ServiceName)
	if err != nil {
		t.Fatalf("Failed to generate editions proto: %v", err)
	}

	// Run protoc with options
	outputArg := "--go-http-server-interface_out=" + config.Options + ":" + tmpDir
	if config.Options == "" {
		outputArg = "--go-http-server-interface_out=" + tmpDir
	}

	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, protoPath)
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	// Verify generated output using standard verification
	verifyEditionsGeneratedOutput(t, config, tmpDir)
}

// verifyEditionsGeneratedOutput verifies the output from editions code generation
func verifyEditionsGeneratedOutput(t *testing.T, config internal.TestWorkflowConfig, tmpDir string) {
	// Handle both prefix patterns (starts with) and suffix patterns (ends with)
	var pattern string
	if strings.HasSuffix(config.ExpectPrefixInFile, "_") {
		// For output_prefix like "generated_", look for files that start with it
		pattern = filepath.Join(tmpDir, config.ExpectPrefixInFile+"*")
	} else {
		// For regular patterns like "_http.pb.go", look for files that contain it
		pattern = filepath.Join(tmpDir, "*"+config.ExpectPrefixInFile)
	}

	generatedFiles, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Error finding generated files: %v", err)
	}

	if len(generatedFiles) == 0 {
		// Try alternative patterns
		allFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.go"))
		t.Fatalf("No generated files found matching pattern %s. Found files: %v", pattern, allFiles)
	}

	// Verify at least one file was generated
	if len(generatedFiles) < 1 {
		t.Fatalf("Expected at least 1 generated file, got %d", len(generatedFiles))
	}

	// Verify content of the generated file
	content, err := os.ReadFile(generatedFiles[0])
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify package name
	if !strings.Contains(contentStr, "package "+config.ExpectedPackage) {
		t.Errorf("Generated file should contain 'package %s'", config.ExpectedPackage)
	}

	// Verify service handler interface
	if !strings.Contains(contentStr, "type "+config.ExpectedServiceName+" interface") {
		t.Errorf("Generated file should contain 'type %s interface'", config.ExpectedServiceName)
	}

	// Verify editions-specific features (multiple HTTP methods)
	editionsFeatures := []string{
		"HandleCreateUser",
		"HandleGetUser",
		"HandleUpdateUser",
		"HandleDeleteUser",
		"HandleListUsers",
		"HandleActivateUser",
		"HandleBulkUpdateUsers",
	}

	foundFeatures := 0
	for _, feature := range editionsFeatures {
		if strings.Contains(contentStr, feature) {
			foundFeatures++
		}
	}

	if foundFeatures < 5 {
		t.Errorf("Expected editions features in generated code, found only %d/%d features",
			foundFeatures, len(editionsFeatures))
	}

	t.Logf("Editions workflow test passed: generated %d bytes with %d/%d expected features",
		len(content), foundFeatures, len(editionsFeatures))
}
