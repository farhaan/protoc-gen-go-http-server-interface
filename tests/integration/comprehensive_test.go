package integration_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
)

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

// createECommerceServiceData creates a complete e-commerce service data structure
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
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/api/v1/users/{user_id}",
								Body:       "",
								PathParams: []string{"user_id"},
							},
						},
					},
					{
						Name:       "CreateUser",
						InputType:  "CreateUserRequest",
						OutputType: "User",
						HTTPRules: []parser.HTTPRule{
							{
								Method:  "POST",
								Pattern: "/api/v1/users",
								Body:    "*",
							},
						},
					},
					{
						Name:       "UpdateUserProfile",
						InputType:  "UpdateUserProfileRequest",
						OutputType: "User",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "PATCH",
								Pattern:    "/api/v1/users/{user_id}/profile",
								Body:       "profile",
								PathParams: []string{"user_id"},
							},
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
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "GET",
								Pattern:    "/api/v1/users/{user_id}/orders",
								Body:       "",
								PathParams: []string{"user_id"},
							},
						},
					},
					{
						Name:       "CreateOrder",
						InputType:  "CreateOrderRequest",
						OutputType: "Order",
						HTTPRules: []parser.HTTPRule{
							{
								Method:  "POST",
								Pattern: "/api/v1/orders",
								Body:    "*",
							},
						},
					},
					{
						Name:       "CancelOrder",
						InputType:  "CancelOrderRequest",
						OutputType: "CancelOrderResponse",
						HTTPRules: []parser.HTTPRule{
							{
								Method:     "DELETE",
								Pattern:    "/api/v1/orders/{order_id}",
								Body:       "",
								PathParams: []string{"order_id"},
							},
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
						HTTPRules: []parser.HTTPRule{
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
		`http.MethodGet, "/api/v1/users/{user_id}"`, `http.MethodPost, "/api/v1/users"`, `http.MethodPatch, "/api/v1/users/{user_id}/profile"`,
		`http.MethodGet, "/api/v1/users/{user_id}/orders"`, `http.MethodPost, "/api/v1/orders"`, `http.MethodDelete, "/api/v1/orders/{order_id}"`,
		`http.MethodPost, "/api/v1/orders/{order_id}/payment"`,
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
