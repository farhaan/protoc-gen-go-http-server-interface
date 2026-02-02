package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/pb/product" // Your generated proto package
)

// ProductService implements the ProductService defined in the proto
type ProductService struct {
	mu       sync.RWMutex
	products map[string]*pb.Product
}

// NewProductService creates a new product service
func NewProductService() *ProductService {
	return &ProductService{
		products: make(map[string]*pb.Product),
	}
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[req.ProductId]
	if !ok {
		return nil, errors.New("product not found")
	}

	// Return a deep copy of the product
	return &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}, nil
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(ctx context.Context, product *pb.Product) (*pb.Product, error) {
	if product.Name == "" {
		return nil, errors.New("product name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if product.Id == "" {
		product.Id = uuid.New().String()
	}

	// Store a copy
	s.products[product.Id] = &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}

	// Return a copy
	return &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}, nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(ctx context.Context, product *pb.Product) (*pb.Product, error) {
	if product.Id == "" {
		return nil, errors.New("product ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.products[product.Id]
	if !ok {
		return nil, errors.New("product not found")
	}

	// Update with a copy
	s.products[product.Id] = &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}

	// Return a copy
	return &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}, nil
}

// DeleteProduct deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, req *pb.GetProductRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.products[req.ProductId]; !ok {
		return nil, errors.New("product not found")
	}

	delete(s.products, req.ProductId)
	return &emptypb.Empty{}, nil
}

// ListProducts lists all products with optional filtering
func (s *ProductService) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start and end indices for pagination
	startIndex := int((req.Page - 1) * req.PageSize)
	endIndex := int(req.Page * req.PageSize)

	// Filter and collect products
	var filteredProducts []*pb.Product
	for _, product := range s.products {
		// Apply search filter if provided
		if req.SearchQuery != "" && !containsIgnoreCase(product.Name, req.SearchQuery) &&
			!containsIgnoreCase(product.Description, req.SearchQuery) {
			continue
		}

		// Add a copy of the product
		filteredProducts = append(filteredProducts, &pb.Product{
			Id:          product.Id,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Stock:       product.Stock,
		})
	}

	// Prepare pagination
	totalCount := len(filteredProducts)
	if startIndex >= totalCount {
		// Return empty page if start index is out of range
		return &pb.ListProductsResponse{
			Products:   []*pb.Product{},
			TotalCount: int32(totalCount),
		}, nil
	}

	if endIndex > totalCount {
		endIndex = totalCount
	}

	// Return paginated results
	return &pb.ListProductsResponse{
		Products:   filteredProducts[startIndex:endIndex],
		TotalCount: int32(totalCount),
	}, nil
}

// Liveness handles health check requests
func (s *ProductService) Liveness(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}
