# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-08-28

### 🚀 Added

- **GraphQL Support**: Full support for GraphQL endpoints
  - `@graphql` annotation for queries, mutations, and subscriptions
  - `@query` annotation for GraphQL query definitions
  - `@variables` annotation for GraphQL variables
  - `@schema` annotation for GraphQL schema documentation
  - Automatic detection of GraphQL endpoints by path patterns
- **Enhanced REST Support**:
  - `@rest` annotation as explicit alternative to `@route`
  - Improved REST endpoint detection and classification
- **Multi-Protocol API Support**:
  - Support for mixed REST and GraphQL APIs in the same project
  - Clear distinction between endpoint types in generated collections
- **Enhanced Postman Collections**:
  - Automatic Content-Type headers based on endpoint type
  - GraphQL-specific request body formatting
  - Enhanced descriptions with endpoint type and operation information
  - Improved JSON formatting for GraphQL queries

### 🔧 Changed

- **Code Architecture**: Refactored endpoint detection for better extensibility
- **Documentation**: Complete rewrite in English with comprehensive examples
- **Type System**: Added endpoint type classification (REST/GraphQL)
- **Error Handling**: Improved fallback mechanisms for type analysis issues

### 🐛 Fixed

- **Package Analysis**: Resolved "package without types" errors with robust fallback
- **Endpoint Detection**: Improved accuracy of automatic endpoint discovery
- **Collection Generation**: Fixed issues with header and body generation

### 📚 Documentation

- **README**: Comprehensive documentation with 349 lines
- **Examples**: Multiple real-world examples for REST, GraphQL, and mixed APIs
- **Annotations**: Complete documentation of all supported annotations
- **Framework Support**: Detailed list of supported web frameworks
- **Contributing**: Added CONTRIBUTING.md with development guidelines
- **Changelog**: Added CHANGELOG.md for version tracking

## [1.0.0] - 2025-08-27

### 🎉 Initial Release

- **Basic REST Support**: Detection of common REST endpoints
- **Framework Support**: Gin, Chi, Echo, Fiber, Gorilla Mux, net/http
- **Annotation System**: Basic `@route`, `@header`, `@body`, `@tag` annotations
- **Postman Integration**: Generate Postman collections and environments
- **CLI Interface**: Command-line tool with various configuration options
- **AST Analysis**: Go AST parsing for endpoint detection
- **Folder Organization**: Configurable folder grouping and organization

---

### Legend

- 🚀 **Added**: New features
- 🔧 **Changed**: Changes in existing functionality
- 🐛 **Fixed**: Bug fixes
- 📚 **Documentation**: Documentation improvements
- ⚠️ **Deprecated**: Soon-to-be removed features
- 🗑️ **Removed**: Removed features
- 🔒 **Security**: Security improvements
