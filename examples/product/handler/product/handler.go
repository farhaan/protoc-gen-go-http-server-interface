package producthandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/pb/product" // Your generated proto package
	service "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/service/product"
)

// ProductHandler implements the HTTP handlers for the Product service
type ProductHandler struct {
	service *service.ProductService
}

// NewProductHandler creates a new product handler
func NewProductHandler(service *service.ProductService) pb.ProductServiceHandler {
	return &ProductHandler{
		service: service,
	}
}

// HandleGetProduct handles GET /products/{product_id}
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from path
	productID := r.PathValue("product_id")
	if productID == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Call service
	product, err := h.service.GetProduct(r.Context(), &pb.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, product)
}

// HandleListProducts handles GET /products
func (h *ProductHandler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var page, pageSize int32 = 1, 10
	var err error

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		pageInt, err := strconv.Atoi(pageStr)
		if err == nil && pageInt > 0 {
			page = int32(pageInt)
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		pageSizeInt, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSizeInt > 0 {
			pageSize = int32(pageSizeInt)
		}
	}

	searchQuery := r.URL.Query().Get("q")

	// Call service
	response, err := h.service.ListProducts(r.Context(), &pb.ListProductsRequest{
		Page:        page,
		PageSize:    pageSize,
		SearchQuery: searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, response)
}

// HandleCreateProduct handles POST /products
func (h *ProductHandler) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	product, err := readProductFromBody(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Call service
	createdProduct, err := h.service.CreateProduct(r.Context(), product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusCreated)
	writeJSONResponse(w, createdProduct)
}

// HandleUpdateProduct handles PUT /products/{id}
func (h *ProductHandler) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from path
	productID := r.PathValue("id")
	if productID == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	product, err := readProductFromBody(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Set the ID from the path
	product.Id = productID

	// Call service
	updatedProduct, err := h.service.UpdateProduct(r.Context(), product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, updatedProduct)
}

// HandleDeleteProduct handles DELETE /products/{product_id}
func (h *ProductHandler) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from path
	productID := r.PathValue("product_id")
	if productID == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Call service
	_, err := h.service.DeleteProduct(r.Context(), &pb.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusNoContent)
}

// HandleLiveness handles GET /liveness
func (h *ProductHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Call service
	_, err := h.service.Liveness(r.Context(), &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

// readProductFromBody reads a product from the request body
func readProductFromBody(r *http.Request) (*pb.Product, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var product pb.Product

	// Check content type
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "application/json", "":
		// Use protojson for better handling of proto-specific features
		err = protojson.Unmarshal(body, &product)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return &product, nil
}

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, msg any) {
	var data []byte
	var err error

	data, err = json.Marshal(msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
