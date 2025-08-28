# Postman Collection Generator

[![CI/CD Pipeline](https://github.com/williamkoller/postman-gen/actions/workflows/ci.yml/badge.svg)](https://github.com/williamkoller/postman-gen/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/williamkoller/postman-gen)](https://goreportcard.com/report/github.com/williamkoller/postman-gen)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/williamkoller/postman-gen)](https://github.com/williamkoller/postman-gen)

A powerful Go tool that automatically generates Postman collections and environments by scanning your Go codebase for API endpoints.

## Features

- 🧠 **Intelligent Project Analysis**: Comprehensively analyzes entire Go projects to understand architecture patterns
- 🏗️ **Architecture-Agnostic**: Automatically detects and adapts to Clean Architecture, MVC, Layered, Microservices, and other patterns
- 🔍 **Automatic API Discovery**: Scans Go source code to detect HTTP endpoints across all packages
- 📦 **Smart Struct Detection**: Analyzes all project structs and generates accurate JSON bodies based on real type definitions
- 🎯 **Cross-Package Resolution**: Resolves types and relationships across different packages and modules
- 📝 **Annotation Support**: Uses special comments to enhance endpoint documentation
- 🏗️ **Framework Support**: Works with popular Go web frameworks (Gin, Chi, Echo, Fiber, Gorilla Mux, net/http)
- 🚀 **GraphQL Support**: Detects and documents GraphQL endpoints with query/mutation/subscription support
- 🔍 **Smart Body Detection**: Matches variables to actual struct definitions for precise JSON generation
- 📁 **Smart Organization**: Groups endpoints by folders with configurable depth
- 🏷️ **Tag-based Grouping**: Creates additional organization using `@tag` annotations
- 🌍 **Environment Generation**: Creates Postman environments with base URLs
- ⚡ **Fast AST Analysis**: Uses Go's AST parsing for reliable endpoint detection
- 🔄 **REST & GraphQL**: Full support for both REST and GraphQL API documentation

## Installation

### Via go install (Recommended)

```bash
go install github.com/williamkoller/postman-gen/cmd/postman-gen@latest
```

### Build from source

```bash
git clone https://github.com/williamkoller/postman-gen.git
cd postman-gen
go build -o postman-gen cmd/postman-gen/main.go
```

### Direct usage

```bash
go run cmd/postman-gen/main.go [flags]
```

## Quick Start

### Option 1: Automatic Detection (Recommended)

1. **Write your handlers with JSON processing:**

   ```go
   func CreatePayment(w http.ResponseWriter, r *http.Request) {
       var payment map[string]interface{}
       json.NewDecoder(r.Body).Decode(&payment) // ← Automatically detected!

       // your handler logic
   }
   ```

2. **Generate Postman collection:**

   ```bash
   ./postman-gen -dir . -name "My API" -out collection.json
   ```

3. **Import `collection.json` into Postman - JSON bodies included automatically!**

### Option 2: Manual Annotations

1. **Add annotations to your Go handlers:**

   ```go
   // @route POST /api/users Create user
   // @header Content-Type: application/json
   // @body {"name":"John Doe","email":"john@example.com","age":30}
   func CreateUser(c *gin.Context) {
       // your handler code
   }
   ```

2. **Generate and import as above**

## Usage

### Basic Usage

```bash
# Generate collection from current directory
./postman-gen -dir . -out collection.json

# Generate collection with custom name and base URL
./postman-gen -dir . -name "My API" -base-url "http://localhost:3000" -out api-collection.json
```

### Advanced Usage

```bash
# Complete setup with environment and organization
./postman-gen \
  -dir ./project \
  -name "My REST API" \
  -base-url "https://api.example.com" \
  -group-depth 2 \
  -group-by-method \
  -tag-folders \
  -out ./collections/api.postman_collection.json \
  -env-out ./collections/dev.postman_environment.json \
  -env-name "Development"
```

### Quick Test with JSON Bodies

```bash
# Test the tool with a single Go file containing @body annotations
echo 'package main

// @route POST /api/users Create user
// @header Content-Type: application/json
// @body {"name":"John Doe","email":"john@example.com","age":30}
func CreateUser() {}

func main() {}' > test.go

./postman-gen -dir test.go -name "Test API" -out test-collection.json
```

### Complete Example with JSON Bodies

```go
package main

import "github.com/gin-gonic/gin"

// @route POST /api/v1/users Create user
// @header Content-Type: application/json
// @header Authorization: Bearer {{token}}
// @body {"name":"John Silva","email":"john@example.com","age":30,"active":true}
// @tag users
// @tag v1
func CreateUser(c *gin.Context) {
    // Handler implementation
}

// @route PUT /api/v1/users/{id} Update user
// @header Authorization: Bearer {{token}}
// @body {"name":"John Silva Updated","email":"john.new@example.com"}
// @tag users
func UpdateUser(c *gin.Context) {
    // Handler implementation
}
```

**Result in Postman collection:**

- ✅ POST and PUT requests with JSON bodies included
- ✅ Authorization and content-type headers configured
- ✅ Organized by tags in folders
- ✅ Variables {{token}} and {{baseUrl}} ready to use

## Real-World Examples

### Payment API with Automatic Detection

```go
// This code will automatically generate appropriate JSON bodies
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user map[string]interface{}
    json.NewDecoder(r.Body).Decode(&user)
    // Generates: {"name":"string","email":"string","id":"string"}
}

func UpdateProfile(w http.ResponseWriter, r *http.Request) {
    var updateRequest map[string]interface{}
    json.NewDecoder(r.Body).Decode(&updateRequest)
    // Generates: {"id":"string","name":"string","value":"string"}
}

func ProcessData(c *gin.Context) {
    var request map[string]interface{}
    c.ShouldBindJSON(&request)
    // Generates: {"data":"string","parameters":{}}
}
```

**Generated Postman Collection includes:**

- ✅ Realistic JSON bodies for each endpoint
- ✅ Proper Content-Type headers
- ✅ Context-aware examples (payment, webhook, user data)
- ✅ No manual annotations required

## Command Line Options

### Essential Options

| Flag        | Type   | Default                   | Description                                     | Example                             |
| ----------- | ------ | ------------------------- | ----------------------------------------------- | ----------------------------------- |
| `-dir`      | string | `"."`                     | Root directory of the Go project to scan        | `-dir ./my-api`                     |
| `-name`     | string | `"Go API"`                | Name of the Postman collection                  | `-name "User Service API"`          |
| `-base-url` | string | `"http://localhost:8080"` | Base URL for the {{baseUrl}} variable           | `-base-url "https://api.myapp.com"` |
| `-out`      | string | `""`                      | Output file for the collection (empty = stdout) | `-out api-collection.json`          |

### Organization Options

| Flag               | Type | Default | Description                             |
| ------------------ | ---- | ------- | --------------------------------------- |
| `-group-depth`     | int  | `1`     | Folder grouping depth (0 = no grouping) |
| `-group-by-method` | bool | `false` | Create HTTP method subfolders           |
| `-tag-folders`     | bool | `false` | Create additional 'By Tag' folder tree  |

### Environment Options

| Flag        | Type   | Default   | Description                                    |
| ----------- | ------ | --------- | ---------------------------------------------- |
| `-env-out`  | string | `""`      | Output file for Postman environment (optional) |
| `-env-name` | string | `"Local"` | Name of the Postman environment                |

### Advanced Options

| Flag          | Type   | Default | Description                                              |
| ------------- | ------ | ------- | -------------------------------------------------------- |
| `-use-types`  | bool   | `true`  | Use enhanced type analysis (currently uses AST fallback) |
| `-build-tags` | string | `""`    | Build tags for type analysis                             |

### Common Command Examples

```bash
# Basic collection
./postman-gen -dir . -out api.json

# Production-ready setup
./postman-gen -dir ./src -name "Production API" -base-url "https://api.prod.com" \
  -group-depth 2 -tag-folders -out prod-collection.json \
  -env-out prod-environment.json -env-name "Production"

# Development with method grouping
./postman-gen -dir . -name "Dev API" -group-by-method -out dev-collection.json
```

## Annotation Support

Enhance your API documentation using special comments in your Go code:

### REST Annotations

#### Route Annotation

```go
// @route GET /api/users/{id} Get user by ID
// @rest GET /api/users/{id} Get user by ID (alternative syntax)
func GetUser(c *gin.Context) {
    // handler implementation
}
```

#### Headers

```go
// @header Authorization: Bearer {token}
// @header Content-Type: application/json
// @route POST /api/users Create new user
func CreateUser(c *gin.Context) {
    // handler implementation
}
```

#### Request Body

Use the `@body` annotation to include JSON bodies in Postman collection requests:

**Basic example:**

```go
// @body {"name": "John Doe", "email": "john@example.com"}
// @route POST /api/users Create user
func CreateUser(c *gin.Context) {
    // handler implementation
}
```

**Example with complex data:**

```go
// @header Content-Type: application/json
// @header Authorization: Bearer {{token}}
// @body {"user_id":1,"items":[{"product_id":123,"quantity":2,"price":29.99}],"total":59.98}
// @route POST /api/orders Create order
// @tag orders
func CreateOrder(c *gin.Context) {
    // handler implementation
}
```

**Example with multiple annotations:**

```go
// @header Authorization: Bearer {{token}}
// @header Content-Type: application/json
// @body {"name":"John Silva","email":"john@example.com","age":30,"active":true}
// @tag users
// @tag v1
// @route POST /api/v1/users Create user
func CreateUser(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    // creation logic
}
```

**Example for updates (PUT/PATCH):**

```go
// @header Authorization: Bearer {{token}}
// @header Content-Type: application/json
// @body {"name":"John Silva Updated","email":"john.new@example.com"}
// @route PUT /api/v1/users/{id} Update user
// @tag users
func UpdateUser(c *gin.Context) {
    // handler implementation
}
```

> **💡 Tip:** The JSON body defined in `@body` will be automatically included in the Postman request with `Content-Type: application/json` and proper formatting.

#### Intelligent Project Analysis & Body Detection

**NEW!** postman-gen now features **intelligent project analysis** that comprehensively scans your entire Go project to understand its architecture and generate accurate JSON bodies based on **real struct definitions** - **no annotations required!**

```go
// Define your structs anywhere in the project
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    Category    string  `json:"category"`
    InStock     bool    `json:"in_stock"`
}

// Use them in handlers - postman-gen will automatically match and generate accurate JSON
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var createUserReq CreateUserRequest  // ← Automatically generates: {"name":"string","email":"string","password":"string"}
    json.NewDecoder(r.Body).Decode(&createUserReq)
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
    var product Product  // ← Automatically generates: {"id":0,"name":"string","description":"string","price":0.0,"category":"string","in_stock":false}
    json.NewDecoder(r.Body).Decode(&product)
}
```

**🧠 Intelligent Analysis Features:**

- **🏗️ Architecture Detection**: Automatically detects Clean Architecture, MVC, Layered, Microservices patterns
- **📦 Cross-Package Resolution**: Finds struct definitions across all packages in your project
- **🎯 Smart Variable Matching**: Matches handler variables to actual struct definitions
- **🔍 Type-Aware Generation**: Generates JSON with correct Go types (int → 0, bool → false, []string → ["string"])
- **🏷️ JSON Tag Support**: Respects `json:"fieldname"` tags and validation rules

**Supported Detection Patterns:**

- `json.NewDecoder(r.Body).Decode(&variable)`
- `c.ShouldBindJSON(&variable)` / `c.BindJSON(&variable)`
- `json.Unmarshal(data, &variable)`
- `io.ReadAll(r.Body)` followed by JSON processing

**Smart Fallback System:**

If specific structs aren't found, falls back to intelligent variable name analysis:

- Variables containing `user` → User JSON with name, email, id
- Variables containing `create`/`post` → Creation JSON with name, value, type
- Variables containing `update`/`put`/`patch` → Update JSON with id, name, value
- Variables containing `request`/`req` → Generic request JSON with data and parameters

### GraphQL Annotations

#### GraphQL Endpoint

```go
// @graphql query /graphql Get users query
// @query query GetUsers { users { id name email } }
// @variables {"limit": 10, "offset": 0}
func GraphQLHandler(c *gin.Context) {
    // GraphQL handler implementation
}
```

#### GraphQL Mutations

```go
// @graphql mutation /graphql Create user mutation
// @query mutation CreateUser($input: UserInput!) { createUser(input: $input) { id name } }
// @variables {"input": {"name": "John", "email": "john@example.com"}}
func GraphQLMutationHandler(c *gin.Context) {
    // GraphQL mutation handler
}
```

#### GraphQL Subscriptions

```go
// @graphql subscription /graphql User updates subscription
// @query subscription UserUpdates { userUpdated { id name status } }
func GraphQLSubscriptionHandler(c *gin.Context) {
    // GraphQL subscription handler
}
```

#### GraphQL Schema Documentation

```go
// @schema type User { id: ID! name: String! email: String! }
// @graphql query /graphql Get users
func GraphQLHandler(c *gin.Context) {
    // handler implementation
}
```

### Universal Annotations

#### Tags

```go
// @tag users
// @tag authentication
// @route GET /api/users List users
func ListUsers(c *gin.Context) {
    // handler implementation
}
```

## Supported Frameworks

The tool automatically detects endpoints from these popular Go web frameworks:

### REST Frameworks

- **Gin**: `router.GET()`, `router.POST()`, etc.
- **Chi**: `r.Get()`, `r.Post()`, `r.Route()`, `r.Group()`
- **Echo**: `e.GET()`, `e.POST()`, `e.Group()`
- **Fiber**: `app.Get()`, `app.Post()`, `app.Group()`
- **Gorilla Mux**: `router.HandleFunc()`, `router.Handle()`, `router.PathPrefix()`
- **net/http**: `http.HandleFunc()`, `mux.Handle()`

### GraphQL Frameworks

- **gqlgen**: Automatic detection of `/graphql` endpoints
- **graphql-go**: Detection of GraphQL handlers
- **99designs/gqlgen**: Schema-first GraphQL support
- **Custom GraphQL**: Any endpoint with "graphql" or "graph" in the path
- **Manual Annotation**: Use `@graphql` annotations for complete control

## Examples

### Basic Gin Application with REST

```go
package main

import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()

    // @route GET /ping Simple health check
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    // @tag users
    // @header Authorization: Bearer {token}
    // @route GET /users/{id} Get user by ID
    r.GET("/users/:id", getUserHandler)

    r.Run()
}
```

### GraphQL Application Example

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/99designs/gqlgen/graphql/handler"
)

func main() {
    r := gin.Default()

    // @graphql query /graphql Get users and posts
    // @query query GetData { users { id name } posts { title } }
    // @header Authorization: Bearer {token}
    // @tag graphql
    r.POST("/graphql", gin.WrapH(handler.NewDefaultServer(generated.NewExecutableSchema())))

    // @graphql mutation /graphql Create user mutation
    // @query mutation CreateUser($input: UserInput!) { createUser(input: $input) { id name } }
    // @variables {"input": {"name": "John", "email": "john@example.com"}}
    // @tag users
    r.POST("/graphql", graphqlHandler)

    r.Run()
}
```

### Mixed REST and GraphQL API

```go
package main

import "github.com/gin-gonic/gin"

func main() {
    r := gin.Default()

    // REST endpoints
    // @rest GET /api/health Check API health
    r.GET("/api/health", healthHandler)

    // @rest POST /api/users Create user via REST
    // @body {"name": "John", "email": "john@example.com"}
    r.POST("/api/users", createUserHandler)

    // GraphQL endpoint
    // @graphql query /graphql GraphQL endpoint for complex queries
    // @query query { users { id name email } }
    r.POST("/graphql", graphqlHandler)

    r.Run()
}
```

### Generated Postman Collection Structure

```
My API Collection/
├── ping/
│   └── GET /ping
└── users/
    └── GET /users/{id}
└── By Tag/
    └── users/
        └── GET /users/{id}
```

## Project Structure

```
postman-gen/
├── cmd/
│   └── postman-gen/
│       └── main.go          # Main application entry point
├── internal/
│   ├── postman/
│   │   ├── postman.go       # Postman collection builder
│   │   └── env.go           # Environment file generator
│   └── scan/
│       ├── scan.go          # AST-based endpoint scanner
│       └── typescan.go      # Type-aware analysis (fallback to AST)
├── go.mod
├── go.sum
└── README.md
```

## Architecture

The tool uses an intelligent multi-phase approach:

1. **🧠 Project Analysis**: Comprehensively scans the entire Go project to understand architecture patterns, struct definitions, and type relationships
2. **🏗️ Architecture Detection**: Automatically detects Clean Architecture, MVC, Layered, Microservices patterns with confidence scoring
3. **📦 Cross-Package Resolution**: Maps all structs, interfaces, and types across packages for intelligent type matching
4. **🔍 AST Scanning**: Parses Go source files to detect HTTP route registrations and handler functions
5. **🎯 Smart Body Generation**: Matches handler variables to actual project struct definitions for accurate JSON generation
6. **📝 Annotation Processing**: Extracts additional documentation from special comments (optional enhancement)
7. **🔧 Collection Building**: Organizes endpoints into Postman collection format with precisely generated request bodies

## Tips and Best Practices

### 🧠 Intelligent Analysis Tips

**1. Define clear struct types (Recommended):**

```go
// ✅ Best - will generate accurate JSON based on struct definition
type CreateUserRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
    var createUserReq CreateUserRequest  // ← Perfect match!
    json.NewDecoder(r.Body).Decode(&createUserReq)
}
```

**2. Use descriptive variable names as fallback:**

```go
// ✅ Good - will generate context-aware JSON if struct not found
var userRequest map[string]interface{}
json.NewDecoder(r.Body).Decode(&userRequest)

// ❌ Less optimal - will generate generic JSON
var data map[string]interface{}
json.NewDecoder(r.Body).Decode(&data)
```

**3. Supported frameworks and patterns:**

```go
// ✅ Standard library
json.NewDecoder(r.Body).Decode(&variable)

// ✅ Gin framework
c.ShouldBindJSON(&variable)
c.BindJSON(&variable)

// ✅ Direct unmarshaling
json.Unmarshal(bodyBytes, &variable)

// ✅ Read-then-unmarshal pattern
body, _ := io.ReadAll(r.Body)
json.Unmarshal(body, &variable)
```

**4. Architecture Compatibility:**

postman-gen automatically adapts to various Go project architectures:

```go
// 🏗️ Clean Architecture
domain/
  entities/user.go     // ← User struct detected
usecase/
  user_usecase.go      // ← Business logic
delivery/http/
  user_handler.go      // ← Handler using User struct

// 🌐 MVC Pattern
models/user.go         // ← Model detected
controllers/
  user_controller.go   // ← Controller using model

// ⚡ Microservice
internal/
  domain/user.go       // ← Domain entity
  handlers/user.go     // ← HTTP handlers
```

**5. Context-aware naming for better examples:**

- `paymentRequest`, `payment` → Payment JSON
- `userRequest`, `user` → User profile JSON
- `webhookData`, `webhook` → Webhook event JSON
- `orderData`, `order` → Order with items JSON
- `cancelRequest` → Cancellation reason JSON

### 💡 Manual @body Usage (Optional)

**1. Keep JSONs valid:**

```go
// ✅ Correct
// @body {"name":"John","age":30,"active":true}

// ❌ Incorrect (invalid JSON)
// @body {name:"John",age:30,active:true}
```

**2. Use realistic data:**

```go
// ✅ Good example with realistic data
// @body {"email":"user@example.com","password":"password123","remember_me":true}

// ❌ Too generic example
// @body {"field1":"value1","field2":"value2"}
```

**3. Combine with appropriate headers:**

```go
// ✅ Consistent headers
// @header Content-Type: application/json
// @header Authorization: Bearer {{token}}
// @body {"data":"example"}
```

**4. Organize by complexity:**

```go
// For simple endpoints
// @body {"name":"John","email":"john@example.com"}

// For complex endpoints with arrays and nested objects
// @body {"user":{"name":"John","profile":{"age":30}},"preferences":["email","sms"]}
```

**5. Use Postman variables:**

```go
// ✅ Use variables for dynamic data
// @body {"user_id":"{{user_id}}","timestamp":"{{$timestamp}}"}
```

### 🚀 Recommended Workflow

#### Modern Approach (Automatic Detection)

1. **Write your handlers with descriptive variable names:**

   ```go
   func CreatePayment(w http.ResponseWriter, r *http.Request) {
       var payment map[string]interface{}  // ← Descriptive name
       json.NewDecoder(r.Body).Decode(&payment)
       // handler logic
   }
   ```

2. **Generate** the collection:

   ```bash
   ./postman-gen -dir . -name "My API v1" -base-url "http://localhost:8080" -out api-collection.json
   ```

3. **Import** into Postman and test:

   - JSON bodies automatically included based on code analysis
   - Context-aware examples (payment, user, webhook, etc.)
   - Headers automatically configured

#### Traditional Approach (Manual Annotations)

1. **Add annotations** for custom control:

   ```go
   // @route POST /api/users Create user
   // @header Authorization: Bearer {{token}}
   // @body {"name":"John","email":"john@example.com"}
   func CreateUser() {}
   ```

2. **Generate and import** as above

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Troubleshooting

### Common Issues

**Issue**: `internal error: package "errors" without types`
**Solution**: The tool automatically falls back to AST scanning mode to avoid this issue.

**Issue**: No endpoints detected
**Solution**: Make sure your code uses supported frameworks or add `@route` annotations.

**Issue**: Missing endpoints
**Solution**: Check that your route registrations are in the scanned directory and use supported patterns.

## Roadmap

- [x] ✅ **GraphQL Support**: Full support for GraphQL queries, mutations, and subscriptions
- [x] ✅ **REST API Enhancement**: Improved REST endpoint detection and documentation
- [x] ✅ **Multi-protocol Support**: Support for both REST and GraphQL in the same API
- [ ] 🔄 **gRPC Support**: Detection and documentation of gRPC services
- [ ] 📊 **OpenAPI/Swagger Integration**: Generate OpenAPI specs alongside Postman collections
- [ ] 🎨 **Custom Annotation Types**: User-defined annotation types for specialized documentation
- [ ] 🧪 **Request/Response Examples**: Auto-generate example requests and responses
- [x] ✅ **CI/CD Integration**: GitHub Actions workflows for automated testing, building, and releases
- [ ] 📝 **TypeScript Support**: Generate TypeScript definitions from API documentation
- [ ] 🔍 **WebSocket Support**: Documentation for WebSocket endpoints

## Support

If you encounter any issues or have questions, please [open an issue](https://github.com/williamkoller/postman-gen/issues) on GitHub.
