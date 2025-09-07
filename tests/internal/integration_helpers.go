// Package internal provides helper utilities for testing the protoc-gen-go-http-server-interface plugin.
package internal

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// HasProtoc checks if protoc is available in the system
func HasProtoc() bool {
	_, err := exec.LookPath("protoc")
	return err == nil
}

// FindPluginBinary locates the protoc plugin binary
func FindPluginBinary() (string, error) {
	// First check current working directory
	cwd, _ := os.Getwd()
	localBinary := filepath.Join(cwd, "..", "protoc-gen-go-http-server-interface")
	if _, err := os.Stat(localBinary); err == nil {
		return localBinary, nil
	}

	// Check two directories up (for tests in subdirectories)
	localBinary = filepath.Join(cwd, "../..", "protoc-gen-go-http-server-interface")
	if _, err := os.Stat(localBinary); err == nil {
		return localBinary, nil
	}

	// Check in PATH
	pathBinary, err := exec.LookPath("protoc-gen-go-http-server-interface")
	if err == nil {
		return pathBinary, nil
	}

	return "", fmt.Errorf("protoc-gen-go-http-server-interface binary not found")
}

// RunProtoc runs protoc with our plugin for a single proto file
func RunProtoc(workDir, pluginPath, outputArg, protoFile string) error {
	// Get absolute path to parent directory for google proto files
	parentDir, _ := filepath.Abs("..")

	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-go-http-server-interface="+pluginPath,
		outputArg,
		"--proto_path="+workDir,
		"--proto_path="+parentDir, // For google/api imports
		protoFile)

	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("protoc failed: %v\nStderr: %s", err, stderr.String())
	}

	return nil
}

// RunProtocMultiple runs protoc with our plugin for multiple proto files
func RunProtocMultiple(workDir, pluginPath string, protoFiles ...string) error {
	// Get absolute path to parent directory for google proto files
	parentDir, _ := filepath.Abs("..")

	args := []string{
		"--plugin=protoc-gen-go-http-server-interface=" + pluginPath,
		"--go-http-server-interface_out=" + workDir,
		"--proto_path=" + workDir,
		"--proto_path=" + parentDir, // For google/api imports
	}
	args = append(args, protoFiles...)

	cmd := exec.Command("protoc", args...)
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("protoc failed: %v\nStderr: %s", err, stderr.String())
	}

	return nil
}

// TestWorkflowConfig represents a workflow test configuration
type TestWorkflowConfig struct {
	Name                string
	PackageName         string
	ServiceName         string
	Options             string
	ExpectedPackage     string
	ExpectedServiceName string
	ExpectPrefixInFile  string
}

// RunWorkflowTest runs a standard workflow test with the given configuration
func RunWorkflowTest(t *testing.T, config TestWorkflowConfig) {
	pluginPath := setupWorkflowTest(t, config)
	tmpDir := generateProtoAndRunProtoc(t, config, pluginPath)
	verifyGeneratedOutput(t, config, tmpDir)
}

// setupWorkflowTest handles initial setup and validation
func setupWorkflowTest(t *testing.T, config TestWorkflowConfig) string {
	if !HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}
	
	return pluginPath
}

// generateProtoAndRunProtoc generates proto file and runs protoc
func generateProtoAndRunProtoc(t *testing.T, config TestWorkflowConfig, pluginPath string) string {
	tmpDir := t.TempDir()

	// Generate proto file
	protoPath := filepath.Join(tmpDir, "test.proto")
	err := GenerateBasicServiceProto(protoPath, config.PackageName, config.ServiceName)
	if err != nil {
		t.Fatalf("Failed to generate proto: %v", err)
	}

	// Run protoc
	outputArg := "--go-http-server-interface_out=" + tmpDir
	if config.Options != "" {
		outputArg = "--go-http-server-interface_out=" + config.Options + ":" + tmpDir
	}

	err = RunProtoc(tmpDir, pluginPath, outputArg, "test.proto")
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}
	
	return tmpDir
}

// verifyGeneratedOutput verifies the generated files and content
func verifyGeneratedOutput(t *testing.T, config TestWorkflowConfig, tmpDir string) {
	// Verify output files
	outputFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.go"))
	if err != nil {
		t.Fatalf("Failed to list output files: %v", err)
	}

	if len(outputFiles) == 0 {
		t.Fatal("No output files generated")
	}

	// Check filename pattern
	checkFilenamePattern(t, config, outputFiles)
	
	// Verify generated content
	verifyGeneratedContent(t, config, outputFiles[0])
}

// checkFilenamePattern verifies the expected filename pattern exists
func checkFilenamePattern(t *testing.T, config TestWorkflowConfig, outputFiles []string) {
	foundMatch := false
	for _, file := range outputFiles {
		if strings.Contains(filepath.Base(file), config.ExpectPrefixInFile) {
			foundMatch = true
			break
		}
	}

	if !foundMatch {
		t.Errorf("Expected filename pattern %q not found in: %v",
			config.ExpectPrefixInFile, outputFiles)
	}
}

// verifyGeneratedContent verifies the content of generated code
func verifyGeneratedContent(t *testing.T, config TestWorkflowConfig, outputFile string) {
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	generatedCode := string(content)
	expectedPatterns := []string{
		"package " + config.ExpectedPackage,
		config.ExpectedServiceName + "Handler interface",
		"Register" + config.ExpectedServiceName + "Routes",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("Generated code missing pattern: %q", pattern)
		}
	}
}

// RunMultiServiceWorkflowTest runs a multi-service workflow test
func RunMultiServiceWorkflowTest(t *testing.T) {
	if !HasProtoc() {
		t.Skip("protoc not available")
	}

	pluginPath, err := FindPluginBinary()
	if err != nil {
		t.Skip("Plugin binary not found")
	}

	tmpDir := t.TempDir()

	// Generate multiple proto files
	protoFiles := []struct {
		filename string
		pkg      string
		service  string
	}{
		{"user.proto", "users", "UserService"},
		{"order.proto", "orders", "OrderService"},
		{"payment.proto", "payments", "PaymentService"},
	}

	generateMultipleProtos(t, tmpDir, protoFiles)
	runProtocOnMultipleFiles(t, tmpDir, pluginPath, protoFiles)
	verifyMultiServiceOutput(t, tmpDir, protoFiles)
}

// generateMultipleProtos generates multiple proto files
func generateMultipleProtos(t *testing.T, tmpDir string, protoFiles []struct {
	filename string
	pkg      string
	service  string
}) {
	for _, proto := range protoFiles {
		protoPath := filepath.Join(tmpDir, proto.filename)
		if genErr := GenerateBasicServiceProto(protoPath, proto.pkg, proto.service); genErr != nil {
			t.Fatalf("Failed to generate %s: %v", proto.filename, genErr)
		}
	}
}

// runProtocOnMultipleFiles runs protoc on multiple files
func runProtocOnMultipleFiles(t *testing.T, tmpDir, pluginPath string, protoFiles []struct {
	filename string
	pkg      string
	service  string
}) {
	protoNames := make([]string, len(protoFiles))
	for i, proto := range protoFiles {
		protoNames[i] = proto.filename
	}

	err := RunProtocMultiple(tmpDir, pluginPath, protoNames...)
	if err != nil {
		t.Fatalf("protoc execution failed: %v", err)
	}
}

// verifyMultiServiceOutput verifies the output for multiple services
func verifyMultiServiceOutput(t *testing.T, tmpDir string, protoFiles []struct {
	filename string
	pkg      string
	service  string
}) {
	for _, proto := range protoFiles {
		pattern := filepath.Join(tmpDir, "*"+strings.TrimSuffix(proto.filename, ".proto")+"*.go")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("Failed to search for %s: %v", pattern, err)
		}

		if len(matches) == 0 {
			t.Errorf("No generated file for %s", proto.filename)
			continue
		}

		verifyMultiServiceFileContent(t, matches[0], proto.pkg, proto.service)
	}
}

// verifyMultiServiceFileContent verifies individual file content
func verifyMultiServiceFileContent(t *testing.T, filename, pkg, service string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filename, err)
	}

	generatedCode := string(content)
	expectedPatterns := []string{
		"package " + pkg,
		service + "Handler interface",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("File %s missing pattern: %q", filename, pattern)
		}
	}
}
