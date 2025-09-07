package compatibility_test

import (
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
)

// TestOpenAPICompatibility_BasicGeneration tests that our plugin works correctly
// with proto files that are designed to also work with protoc-gen-openapi-v2
func TestOpenAPICompatibility_BasicGeneration(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	// Simulate a service that would have both HTTP and OpenAPI annotations
	serviceData := &httpinterface.ServiceData{
		PackageName: "openapiv1",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "ProductService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetProduct",
						InputType:  "GetProductRequest",
						OutputType: "Product",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/products/{product_id}", Body: "", PathParams: []string{"product_id"}},
						},
					},
					{
						Name:       "CreateProduct",
						InputType:  "CreateProductRequest",
						OutputType: "Product",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/products", Body: "*"},
						},
					},
					{
						Name:       "ListProducts",
						InputType:  "ListProductsRequest",
						OutputType: "ListProductsResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/products", Body: ""},
						},
					},
					{
						Name:       "UpdateProduct",
						InputType:  "UpdateProductRequest",
						OutputType: "Product",
						HTTPRules: []httpinterface.HTTPRule{
							{
								Method:     "PUT",
								Pattern:    "/products/{product.product_id}",
								Body:       "product",
								PathParams: []string{"product.product_id"},
							},
						},
					},
					{
						Name:       "DeleteProduct",
						InputType:  "DeleteProductRequest",
						OutputType: "DeleteProductResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "DELETE", Pattern: "/products/{product_id}", Body: "", PathParams: []string{"product_id"}},
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

	// Verify the generated code works as expected with OpenAPI-structured services
	expectedPatterns := []string{
		"package openapiv1",
		"type ProductServiceHandler interface",
		"HandleGetProduct(w http.ResponseWriter, r *http.Request)",
		"HandleCreateProduct(w http.ResponseWriter, r *http.Request)",
		"HandleListProducts(w http.ResponseWriter, r *http.Request)",
		"HandleUpdateProduct(w http.ResponseWriter, r *http.Request)",
		"HandleDeleteProduct(w http.ResponseWriter, r *http.Request)",
		"RegisterProductServiceRoutes",
		`"GET", "/products/{product_id}"`,
		`"POST", "/products"`,
		`"GET", "/products"`,
		`"PUT", "/products/{product.product_id}"`,
		`"DELETE", "/products/{product_id}"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}

	// Verify the generated code does NOT contain OpenAPI-specific artifacts
	// (Our plugin should ignore OpenAPI annotations and only process HTTP ones)
	forbiddenPatterns := []string{
		"swagger",
		"schema",
		"responses",
		"operation",
		"@example",
		"description",
		"summary",
		"openapi_v2", // Allow "openapi" in package names but not specific artifacts
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(strings.ToLower(generated), pattern) {
			t.Errorf("Generated code should not contain OpenAPI-specific pattern: %q", pattern)
		}
	}
}

// TestOpenAPICompatibility_NestedMessages tests compatibility with complex nested message structures
// commonly used in OpenAPI-compatible proto definitions
func TestOpenAPICompatibility_NestedMessages(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "nestedapi",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "NestedService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "GetNestedResource",
						InputType:  "GetNestedResourceRequest",
						OutputType: "NestedResourceResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/api/v1/orgs/{org}/projects/{project}/resources/{resource.metadata.id}",
								Body: "", PathParams: []string{"org", "project", "resource.metadata.id"}},
						},
					},
					{
						Name:       "UpdateNestedResource",
						InputType:  "UpdateNestedResourceRequest",
						OutputType: "NestedResourceResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PATCH", Pattern: "/api/v1/orgs/{org}/projects/{project}/resources/{resource.metadata.id}",
								Body: "resource.data", PathParams: []string{"org", "project", "resource.metadata.id"}},
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

	// Verify nested path parameters are handled correctly
	expectedPatterns := []string{
		"package nestedapi",
		"NestedServiceHandler interface",
		"HandleGetNestedResource",
		"HandleUpdateNestedResource",
		`"/api/v1/orgs/{org}/projects/{project}/resources/{resource.metadata.id}"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}
}

// TestOpenAPICompatibility_HTTPBodyVariations tests various HTTP body patterns
// that are commonly used alongside OpenAPI specifications
func TestOpenAPICompatibility_HTTPBodyVariations(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "bodyapi",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "BodyVariationService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:       "CreateWithFullBody",
						InputType:  "CreateRequest",
						OutputType: "CreateResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "POST", Pattern: "/resources", Body: "*"},
						},
					},
					{
						Name:       "UpdateWithFieldBody",
						InputType:  "UpdateRequest",
						OutputType: "UpdateResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PATCH", Pattern: "/resources/{id}", Body: "update_mask", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "UpdateWithNestedBody",
						InputType:  "UpdateNestedRequest",
						OutputType: "UpdateNestedResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "PUT", Pattern: "/resources/{id}", Body: "resource.data", PathParams: []string{"id"}},
						},
					},
					{
						Name:       "GetWithoutBody",
						InputType:  "GetRequest",
						OutputType: "GetResponse",
						HTTPRules: []httpinterface.HTTPRule{
							{Method: "GET", Pattern: "/resources/{id}", Body: "", PathParams: []string{"id"}},
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

	// Verify all HTTP body variations are properly handled
	expectedPatterns := []string{
		"package bodyapi",
		"BodyVariationServiceHandler interface",
		"HandleCreateWithFullBody",
		"HandleUpdateWithFieldBody",
		"HandleUpdateWithNestedBody",
		"HandleGetWithoutBody",
		`"POST", "/resources"`,
		`"PATCH", "/resources/{id}"`,
		`"PUT", "/resources/{id}"`,
		`"GET", "/resources/{id}"`,
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Generated code missing expected pattern: %q", pattern)
		}
	}
}

// TestOpenAPICompatibility_MultiServiceGeneration tests multiple services in a single proto
// which is common in OpenAPI-structured microservice definitions
func TestOpenAPICompatibility_MultiServiceGeneration(t *testing.T) {
	t.Parallel()
	generator := httpinterface.New()

	serviceData := &httpinterface.ServiceData{
		PackageName: "multiapi",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "UserService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "GetUser",
						HTTPRules: []httpinterface.HTTPRule{{Method: "GET", Pattern: "/users/{id}", PathParams: []string{"id"}}},
					},
					{
						Name:      "CreateUser",
						HTTPRules: []httpinterface.HTTPRule{{Method: "POST", Pattern: "/users", Body: "*"}},
					},
				},
			},
			{
				Name: "OrderService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "GetOrder",
						HTTPRules: []httpinterface.HTTPRule{{Method: "GET", Pattern: "/orders/{id}", PathParams: []string{"id"}}},
					},
					{
						Name:      "CreateOrder",
						HTTPRules: []httpinterface.HTTPRule{{Method: "POST", Pattern: "/orders", Body: "*"}},
					},
				},
			},
			{
				Name: "PaymentService",
				Methods: []httpinterface.MethodInfo{
					{
						Name:      "ProcessPayment",
						HTTPRules: []httpinterface.HTTPRule{{Method: "POST", Pattern: "/payments", Body: "*"}},
					},
				},
			},
		},
	}

	generated, err := generator.GenerateCode(serviceData)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	// Verify all services are generated correctly
	expectedServices := []string{
		"type UserServiceHandler interface",
		"type OrderServiceHandler interface",
		"type PaymentServiceHandler interface",
		"RegisterUserServiceRoutes",
		"RegisterOrderServiceRoutes",
		"RegisterPaymentServiceRoutes",
	}

	for _, service := range expectedServices {
		if !strings.Contains(generated, service) {
			t.Errorf("Generated code missing service element: %q", service)
		}
	}

	// Verify all HTTP endpoints are present
	expectedEndpoints := []string{
		`"GET", "/users/{id}"`,
		`"POST", "/users"`,
		`"GET", "/orders/{id}"`,
		`"POST", "/orders"`,
		`"POST", "/payments"`,
	}

	for _, endpoint := range expectedEndpoints {
		if !strings.Contains(generated, endpoint) {
			t.Errorf("Generated code missing endpoint: %q", endpoint)
		}
	}
}
