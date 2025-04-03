# Example Project

This guide walks through setting up a complete project with the HTTP Interface Generator that demonstrates creating RESTful services from Protocol Buffer definitions.

## Project Structure

```
project/
├── handler/
│   ├── product/
│   │   └── handler.go
│   └── user/
│       └── handler.go
├── proto/
│   ├── product/
│   │   └── product.proto
│   └── user/
│       └── user.proto
├── service/
│   ├── product/
│   │   └── service.go
│   └── user/
│       └── service.go
├── buf.gen.yaml
├── buf.lock
├── buf.yaml
├── go.mod
└── main.go
```

## Project Setup

First, create your project structure:

```bash
mkdir -p myproject/{handler,proto,service}/{product,user}
cd myproject
go mod init myproject
```

## Add Dependencies

Install required dependencies:

```bash
# Install buf for proto compilation
brew install bufbuild/buf/buf   # or equivalent for your OS

# Install required dependencies
go get google.golang.org/protobuf/cmd/protoc-gen-go
go get github.com/farhaan/protoc-gen-http-server-interface

```
## Protocol Buffer Definitions

The project uses two main proto definitions:

### Product Service (proto/product/product.proto)
Defines products with basic CRUD operations and includes:
- Product message with fields like id, name, description, price, and stock
- APIs for creating, reading, updating, and deleting products
- List operation with pagination and search
- Health check endpoint

### User Service (proto/user/user.proto)
Defines users with similar CRUD operations and includes:
- User message with fields like id, username, email, fullName, and status
- User status enum for managing user states
- APIs for user management operations
- List operation with pagination, search, and status filtering
- Health check endpoint

## Code Generation

The project uses buf for code generation. Configuration is in three files:

### buf.yaml
```yaml
version: v2
deps:
  - buf.build/googleapis/googleapis
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

### buf.gen.yaml
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/yourusername/project/pb
plugins:
  - remote: buf.build/protocolbuffers/go
    out: pb
    opt: paths=source_relative
  - local: protoc-gen-go-http-server-interface
    out: pb
    opt: paths=source_relative
```

Generate code by running:

```bash
buf generate --include-imports --config buf.yaml
```

## Implementation

### Service Layer
Both product and user services implement in-memory storage with thread-safe operations using sync.RWMutex. Key features include:

- CRUD operations with proper error handling
- Search functionality with case-insensitive matching
- Pagination support
- Timestamp tracking for user operations
- Status management for users

### HTTP Handlers
The handlers provide the HTTP interface by:

- Parsing request parameters and bodies
- Converting between proto and JSON formats
- Implementing proper error handling
- Supporting content negotiation
- Providing health check endpoints

### Main Application

The main.go file demonstrates several routing patterns:

1. Basic routing with prefixes:
```go
productsApi := productRouter.Group("/api/products")
productsApi.RegisterProductServiceRoutes(productHandler)

usersApi := userRouter.Group("/api/users")
usersApi.RegisterUserServiceRoutes(userHandler)
```

2. Versioned API routes with middleware:
```go
v1Products := productRouter.Group("/api/v1/products").Use(productPb.Middleware(Auth()))
v1Products.RegisterGetProduct(productHandler)
v1Products.RegisterListProducts(productHandler, productPb.Middleware(RateLimiter(60)))
```

Common middleware implementations include:
- Logging
- Authentication
- Rate limiting

## Testing the API

Start the server:
```bash
go run main.go
```

Example API calls:

```bash
# Create a product
curl -X POST http://localhost:8080/api/products/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Product","description":"A test product","price":99.99,"stock":100}'

# Create a user
curl -X POST http://localhost:8080/api/users/users \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","full_name":"Test User","password":"securepass"}'

# List products with search
curl "http://localhost:8080/api/products/products?search_query=test&page=1&page_size=10"

# List users with status filter
curl "http://localhost:8080/api/users/users?status=1&page=1&page_size=10"

# Health checks
curl http://localhost:8080/api/products/liveness
curl http://localhost:8080/api/users/liveness
```
