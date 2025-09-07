package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internal "tests/internal"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

// TestIntegration_FullWorkflow tests the complete workflow from options to generated code
func TestIntegration_FullWorkflow(t *testing.T) {
	t.Parallel()
	tests := getWorkflowTestCases()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runWorkflowTestCase(t, test)
		})
	}
}

func getWorkflowTestCases() []struct {
	name                string
	packageName         string
	serviceName         string
	expectedPackage     string
	expectedServiceName string
	options             string
	expectPrefixInFile  string
} {
	return []struct {
		name                string
		packageName         string
		serviceName         string
		expectedPackage     string
		expectedServiceName string
		options             string
		expectPrefixInFile  string
	}{
		{
			name:                "default_workflow",
			packageName:         "workflowv1",
			serviceName:         "WorkflowService",
			expectedPackage:     "workflowv1",
			expectedServiceName: "WorkflowService",
			options:             "",
			expectPrefixInFile:  "test_http.pb.go",
		},
		{
			name:                "custom_prefix_workflow",
			packageName:         "apiv2",
			serviceName:         "APIService",
			expectedPackage:     "apiv2",
			expectedServiceName: "APIService",
			options:             "output_prefix=service",
			expectPrefixInFile:  "service_",
		},
		{
			name:                "source_relative_workflow",
			packageName:         "internalcore",
			serviceName:         "CoreService",
			expectedPackage:     "internalcore",
			expectedServiceName: "CoreService",
			options:             "paths=source_relative",
			expectPrefixInFile:  "test_http.pb.go",
		},
		{
			name:                "editions_workflow",
			packageName:         "editionsv1",
			serviceName:         "EditionsService",
			expectedPackage:     "editionsv1",
			expectedServiceName: "EditionsService",
			options:             "",
			expectPrefixInFile:  "test_http.pb.go",
		},
	}
}

func runWorkflowTestCase(t *testing.T, test struct {
	name                string
	packageName         string
	serviceName         string
	expectedPackage     string
	expectedServiceName string
	options             string
	expectPrefixInFile  string
}) {
	config := internal.TestWorkflowConfig{
		Name:                test.name,
		PackageName:         test.packageName,
		ServiceName:         test.serviceName,
		Options:             test.options,
		ExpectedPackage:     test.expectedPackage,
		ExpectedServiceName: test.expectedServiceName,
		ExpectPrefixInFile:  test.expectPrefixInFile,
	}

	if strings.Contains(test.name, "editions") {
		runEditionsIntegrationTest(t, config)
	} else {
		internal.RunWorkflowTest(t, config)
	}
}

// TestIntegration_MultiServiceWorkflow tests multi-service generation workflow
func TestIntegration_MultiServiceWorkflow(t *testing.T) {
	t.Parallel()
	internal.RunMultiServiceWorkflowTest(t)
}

// TestIntegration_MicroserviceScenario tests a complete e-commerce microservice scenario
func TestIntegration_MicroserviceScenario(t *testing.T) {
	t.Parallel()
	testMicroserviceScenario(t, "Microservice scenario")
}

// testMicroserviceScenario is the shared implementation for microservice testing
func testMicroserviceScenario(t *testing.T, scenarioName string) {
	generator := httpinterface.New()

	serviceData := createECommerceServiceData()
	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	validateMicroservicePatterns(t, generated, scenarioName)
}

// createECommerceServiceData creates the test service data
func createECommerceServiceData() *httpinterface.ServiceData {
	return &httpinterface.ServiceData{
		PackageName: "ecommerce",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "UserService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetUser",
						InputType:  "GetUserRequest",
						OutputType: "User",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/api/v1/users/{user_id}", PathParams: []string{"user_id"}},
						},
					},
					{
						Name:       "CreateUser",
						InputType:  "CreateUserRequest",
						OutputType: "User",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/api/v1/users", Body: "*"},
						},
					},
					{
						Name:       "UpdateUserProfile",
						InputType:  "UpdateUserProfileRequest",
						OutputType: "User",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PATCH", Pattern: "/api/v1/users/{user_id}/profile", Body: "profile", PathParams: []string{"user_id"}},
						},
					},
				},
			},
			{
				Name: "OrderService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "ListUserOrders",
						InputType:  "ListUserOrdersRequest",
						OutputType: "ListUserOrdersResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/api/v1/users/{user_id}/orders", PathParams: []string{"user_id"}},
						},
					},
					{
						Name:       "CreateOrder",
						InputType:  "CreateOrderRequest",
						OutputType: "Order",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/api/v1/orders", Body: "*"},
						},
					},
					{
						Name:       "CancelOrder",
						InputType:  "CancelOrderRequest",
						OutputType: "CancelOrderResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "DELETE", Pattern: "/api/v1/orders/{order_id}", PathParams: []string{"order_id"}},
						},
					},
				},
			},
			{
				Name: "PaymentService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "ProcessPayment",
						InputType:  "ProcessPaymentRequest",
						OutputType: "ProcessPaymentResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{
								Method:     "POST",
								Pattern:    "/api/v1/orders/{order_id}/payment",
								Body:       "payment_details",
								PathParams: []string{"order_id"},
							},
						},
					},
				},
			},
		},
	}
}

// validateMicroservicePatterns validates the generated microservice code
func validateMicroservicePatterns(t *testing.T, generated, scenarioName string) {
	expectedPatterns := []string{
		"package ecommerce", "Routes interface", "RouteGroup",
		"type UserServiceHandler interface", "type OrderServiceHandler interface", "type PaymentServiceHandler interface",
		"HandleGetUser", "HandleCreateUser", "HandleUpdateUserProfile",
		"HandleListUserOrders", "HandleCreateOrder", "HandleCancelOrder", "HandleProcessPayment",
		"RegisterUserServiceRoutes", "RegisterOrderServiceRoutes", "RegisterPaymentServiceRoutes",
		`"GET", "/api/v1/users/{user_id}"`, `"POST", "/api/v1/users"`, `"PATCH", "/api/v1/users/{user_id}/profile"`,
		`"GET", "/api/v1/users/{user_id}/orders"`, `"POST", "/api/v1/orders"`, `"DELETE", "/api/v1/orders/{order_id}"`,
		`"POST", "/api/v1/orders/{order_id}/payment"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("%s missing pattern: %q", scenarioName, pattern)
		}
	}

	forbiddenPatterns := []string{"Serializer", "Adapter", "CustomImport"}
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(generated, pattern) {
			t.Errorf("Generated code contains forbidden pattern: %q", pattern)
		}
	}
}

// runEditionsIntegrationTest is a specialized version of RunWorkflowTest for editions
func runEditionsIntegrationTest(t *testing.T, config internal.TestWorkflowConfig) {
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

	// Verify generated output
	verifyEditionsIntegrationOutput(t, config, tmpDir)
}

// verifyEditionsIntegrationOutput verifies the output from editions code generation
func verifyEditionsIntegrationOutput(t *testing.T, config internal.TestWorkflowConfig, tmpDir string) {
	pattern := filepath.Join(tmpDir, "*"+config.ExpectPrefixInFile)
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
	if !strings.Contains(contentStr, "type "+config.ExpectedServiceName+"Handler interface") {
		t.Errorf("Generated file should contain 'type %sHandler interface'", config.ExpectedServiceName)
	}

	t.Logf("Editions workflow test passed for %s: generated %d bytes", config.Name, len(content))
}
