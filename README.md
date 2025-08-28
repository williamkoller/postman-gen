# Postman Collection Generator

[![CI/CD Pipeline](https://github.com/williamkoller/postman-gen/actions/workflows/ci.yml/badge.svg)](https://github.com/williamkoller/postman-gen/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/williamkoller/postman-gen)](https://goreportcard.com/report/github.com/williamkoller/postman-gen)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/williamkoller/postman-gen)](https://github.com/williamkoller/postman-gen)

A powerful Go tool that automatically generates Postman collections and environments by scanning your Go codebase for API endpoints.

## Features

- 🔍 **Automatic API Discovery**: Scans Go source code to detect HTTP endpoints
- 📝 **Annotation Support**: Uses special comments to enhance endpoint documentation
- 🏗️ **Framework Support**: Works with popular Go web frameworks (Gin, Chi, Echo, Fiber, Gorilla Mux, net/http)
- 🚀 **GraphQL Support**: Detects and documents GraphQL endpoints with query/mutation/subscription support
- 📁 **Smart Organization**: Groups endpoints by folders with configurable depth
- 🏷️ **Tag-based Grouping**: Creates additional organization using `@tag` annotations
- 🌍 **Environment Generation**: Creates Postman environments with base URLs
- ⚡ **Fast AST Analysis**: Uses Go's AST parsing for reliable endpoint detection
- 🔄 **REST & GraphQL**: Full support for both REST and GraphQL API documentation

## Installation

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

## Usage

### Basic Usage

```bash
./postman-gen -dir /path/to/your/go/project -out collection.json
```

### Advanced Usage

```bash
./postman-gen \
  -dir ./my-api \
  -name "My API Collection" \
  -base-url "https://api.example.com" \
  -group-depth 2 \
  -group-by-method \
  -tag-folders \
  -out ./collection.postman.json \
  -env-out ./environment.postman_environment.json \
  -env-name "Production"
```

## Command Line Options

| Flag               | Type   | Default                   | Description                                              |
| ------------------ | ------ | ------------------------- | -------------------------------------------------------- |
| `-dir`             | string | `"."`                     | Root directory of the Go project to scan                 |
| `-name`            | string | `"Go API"`                | Name of the Postman collection                           |
| `-base-url`        | string | `"http://localhost:8080"` | Base URL for the {{baseUrl}} variable                    |
| `-out`             | string | `""`                      | Output file for the collection (empty = stdout)          |
| `-group-depth`     | int    | `1`                       | Folder grouping depth (0 = no grouping)                  |
| `-group-by-method` | bool   | `false`                   | Create HTTP method subfolders                            |
| `-tag-folders`     | bool   | `false`                   | Create additional 'By Tag' folder tree                   |
| `-use-types`       | bool   | `true`                    | Use enhanced type analysis (currently uses AST fallback) |
| `-build-tags`      | string | `""`                      | Build tags for type analysis                             |
| `-env-out`         | string | `""`                      | Output file for Postman environment (optional)           |
| `-env-name`        | string | `"Local"`                 | Name of the Postman environment                          |

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

```go
// @body {"name": "John Doe", "email": "john@example.com"}
// @route POST /api/users Create user
func CreateUser(c *gin.Context) {
    // handler implementation
}
```

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

The tool uses a two-phase approach:

1. **AST Scanning**: Parses Go source files to detect HTTP route registrations
2. **Annotation Processing**: Extracts documentation from special comments
3. **Collection Building**: Organizes endpoints into Postman collection format

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
