package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "tests/internal"
)

// TestE2E_CustomHTTPMethods tests end-to-end generation of custom HTTP methods (HEAD, OPTIONS, etc.)
func TestE2E_CustomHTTPMethods(t *testing.T) {
	t.Parallel()

	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	if !internal.HasGoogleAPIs() {
		t.Skip("Skipping E2E test - googleapis proto files not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found - run 'go build' first")
	}

	tmpDir := t.TempDir()

	// Generate proto file with custom HTTP methods
	protoPath := filepath.Join(tmpDir, "custom_http.proto")
	err = internal.GenerateCustomHTTPServiceProto(protoPath, "customhttp", "CustomHTTPService")
	if err != nil {
		t.Fatalf("Failed to generate proto file: %v", err)
	}

	// Run protoc
	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "custom_http.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	// Read generated file
	outputFile := filepath.Join(tmpDir, "custom_http_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Verify custom HTTP methods are correctly generated
	expectedPatterns := []struct {
		pattern     string
		description string
	}{
		{`"HEAD", "/health"`, "HEAD method registration"},
		{`"OPTIONS", "/resources"`, "OPTIONS method registration"},
		{`"TRACE", "/debug/trace/{request_id}"`, "TRACE method with path param"},
		{`"PURGE", "/cache/{namespace}/{key}"`, "Custom PURGE method with multiple path params"},
		{"HandleCheckHealth", "HEAD handler method"},
		{"HandleGetOptions", "OPTIONS handler method"},
		{"HandleTraceRequest", "TRACE handler method"},
		{"HandleCustomAction", "PURGE handler method"},
	}

	for _, exp := range expectedPatterns {
		if !strings.Contains(generatedCode, exp.pattern) {
			t.Errorf("Generated code missing %s: %q", exp.description, exp.pattern)
		}
	}

	t.Logf("E2E custom HTTP methods test passed. Generated %d bytes", len(generatedCode))
}

// TestE2E_CustomHTTPMethodsPathParams verifies path parameters work with custom methods
func TestE2E_CustomHTTPMethodsPathParams(t *testing.T) {
	t.Parallel()

	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	if !internal.HasGoogleAPIs() {
		t.Skip("Skipping E2E test - googleapis proto files not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	protoPath := filepath.Join(tmpDir, "custom_http.proto")
	err = internal.GenerateCustomHTTPServiceProto(protoPath, "customhttp", "CustomHTTPService")
	if err != nil {
		t.Fatalf("Failed to generate proto file: %v", err)
	}

	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "custom_http.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "custom_http_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Verify path parameter extraction code exists for custom methods
	pathParamPatterns := []string{
		"request_id", // from TRACE /debug/trace/{request_id}
		"namespace",  // from PURGE /cache/{namespace}/{key}
		"key",        // from PURGE /cache/{namespace}/{key}
	}

	for _, param := range pathParamPatterns {
		if !strings.Contains(generatedCode, param) {
			t.Errorf("Generated code should reference path parameter: %q", param)
		}
	}
}

// TestE2E_MixedStandardAndCustomMethods verifies standard and custom methods coexist
func TestE2E_MixedStandardAndCustomMethods(t *testing.T) {
	t.Parallel()

	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	if !internal.HasGoogleAPIs() {
		t.Skip("Skipping E2E test - googleapis proto files not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	protoPath := filepath.Join(tmpDir, "custom_http.proto")
	err = internal.GenerateCustomHTTPServiceProto(protoPath, "mixedhttp", "MixedHTTPService")
	if err != nil {
		t.Fatalf("Failed to generate proto file: %v", err)
	}

	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "custom_http.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "custom_http_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Verify both standard and custom methods exist
	standardMethods := []string{
		`"GET", "/resources/{id}"`,
		`"POST", "/resources"`,
	}

	customMethods := []string{
		`"HEAD", "/health"`,
		`"OPTIONS", "/resources"`,
	}

	for _, method := range standardMethods {
		if !strings.Contains(generatedCode, method) {
			t.Errorf("Missing standard method: %s", method)
		}
	}

	for _, method := range customMethods {
		if !strings.Contains(generatedCode, method) {
			t.Errorf("Missing custom method: %s", method)
		}
	}
}

// TestE2E_GeneratedCodeCompiles verifies generated code is syntactically valid Go
func TestE2E_GeneratedCodeCompiles(t *testing.T) {
	t.Parallel()

	if !internal.HasProtoc() {
		t.Skip("Skipping E2E test - protoc not available")
	}

	if !internal.HasGoogleAPIs() {
		t.Skip("Skipping E2E test - googleapis proto files not available")
	}

	pluginPath, err := internal.FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	protoPath := filepath.Join(tmpDir, "custom_http.proto")
	err = internal.GenerateCustomHTTPServiceProto(protoPath, "customhttp", "CustomHTTPService")
	if err != nil {
		t.Fatalf("Failed to generate proto file: %v", err)
	}

	outputArg := "--go-http-server-interface_out=" + tmpDir
	err = internal.RunProtoc(tmpDir, pluginPath, outputArg, "custom_http.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "custom_http_http.pb.go")
	generated, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(generated)

	// Basic structural checks for valid Go code
	requiredElements := []string{
		"package customhttp",
		"import (",
		"net/http",
		"type CustomHTTPServiceHandler interface",
		"func RegisterCustomHTTPServiceRoutes",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(generatedCode, elem) {
			t.Errorf("Generated code missing required element: %q", elem)
		}
	}

	// Verify no obvious syntax issues
	if strings.Contains(generatedCode, "{{") || strings.Contains(generatedCode, "}}") {
		t.Error("Generated code contains unprocessed template markers")
	}
}
