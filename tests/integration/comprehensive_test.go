package integration_test

import (
	"strings"
	"testing"

	internal "tests/internal"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

// TestIntegration_FullWorkflow tests the complete workflow from options to generated code
func TestIntegration_FullWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
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
			expectPrefixInFile:  "test_http.pb.go", // Based on proto filename
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config := internal.TestWorkflowConfig{
				Name:                test.name,
				PackageName:         test.packageName,
				ServiceName:         test.serviceName,
				Options:             test.options,
				ExpectedPackage:     test.expectedPackage,
				ExpectedServiceName: test.expectedServiceName,
				ExpectPrefixInFile:  test.expectPrefixInFile,
			}
			internal.RunWorkflowTest(t, config)
		})
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
	generator := httpinterface.New()

	// Realistic e-commerce microservice setup
	serviceData := &httpinterface.ServiceData{
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

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Verify realistic microservice patterns
	expectedPatterns := []string{
		// Package and structure
		"package ecommerce",
		"Routes interface",
		"RouteGroup",

		// All services
		"type UserServiceHandler interface",
		"type OrderServiceHandler interface",
		"type PaymentServiceHandler interface",

		// All HTTP methods
		"HandleGetUser", "HandleCreateUser", "HandleUpdateUserProfile",
		"HandleListUserOrders", "HandleCreateOrder", "HandleCancelOrder",
		"HandleProcessPayment",

		// All registration functions
		"RegisterUserServiceRoutes",
		"RegisterOrderServiceRoutes",
		"RegisterPaymentServiceRoutes",

		// HTTP routes
		`"GET", "/api/v1/users/{user_id}"`,
		`"POST", "/api/v1/users"`,
		`"PATCH", "/api/v1/users/{user_id}/profile"`,
		`"GET", "/api/v1/users/{user_id}/orders"`,
		`"POST", "/api/v1/orders"`,
		`"DELETE", "/api/v1/orders/{order_id}"`,
		`"POST", "/api/v1/orders/{order_id}/payment"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Microservice scenario missing pattern: %q", pattern)
		}
	}

	// Verify no forbidden patterns (legacy adapter system)
	forbiddenPatterns := []string{"Serializer", "Adapter", "CustomImport"}
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(generated, pattern) {
			t.Errorf("Generated code contains forbidden pattern: %q", pattern)
		}
	}
}
