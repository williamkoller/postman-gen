# Contributing to Postman Collection Generator

Thank you for your interest in contributing to postman-gen! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Basic understanding of Go web frameworks

### Development Setup

1. **Fork and Clone**

   ```bash
   git clone https://github.com/williamkoller/postman-gen.git
   cd postman-gen
   ```

2. **Install Dependencies**

   ```bash
   go mod download
   ```

3. **Build and Test**
   ```bash
   go build -o postman-gen cmd/postman-gen/main.go
   go test ./...
   ```

## 🏗️ Project Structure

```
postman-gen/
├── cmd/postman-gen/     # Main application entry point
├── internal/
│   ├── postman/         # Postman collection/environment builders
│   └── scan/            # Code scanning and annotation parsing
├── README.md            # Project documentation
└── LICENSE              # MIT License
```

## 📝 How to Contribute

### 🐛 Bug Reports

When filing an issue, please include:

- **Go version**: `go version`
- **Operating system**: Linux, macOS, Windows
- **Expected behavior**
- **Actual behavior**
- **Minimal reproducible example**

### ✨ Feature Requests

For new features, please:

1. **Check existing issues** to avoid duplicates
2. **Describe the use case** clearly
3. **Provide examples** of the desired functionality
4. **Consider backward compatibility**

### 🔧 Pull Requests

1. **Create a feature branch**

   ```bash
   git checkout -b feature/awesome-feature
   ```

2. **Follow Go conventions**

   - Use `gofmt` for formatting
   - Follow effective Go practices
   - Add tests for new functionality
   - Update documentation as needed

3. **Write clear commit messages**

   ```
   feat: add support for gRPC endpoints

   - Implement gRPC service detection
   - Add @grpc annotation support
   - Update documentation with examples
   ```

4. **Test thoroughly**

   ```bash
   go test ./...
   go test -race ./...
   go test -fuzz=. ./internal/scan
   ```

5. **Update documentation**
   - Add examples to README.md
   - Document new annotations
   - Update roadmap if applicable

## 🧪 Testing Guidelines

### Unit Tests

- Test files should end with `_test.go`
- Use table-driven tests when appropriate
- Mock external dependencies

### Integration Tests

- Test real-world scenarios
- Include multiple framework examples
- Validate generated Postman collections

### Fuzz Tests

- Test annotation parsing with random inputs
- Ensure robustness against malformed data

## 📚 Documentation Standards

### Code Comments

- Use clear, concise comments
- Document public functions and types
- Explain complex logic or algorithms

### README Updates

- Add examples for new features
- Update the features list
- Keep the roadmap current

## 🎨 Code Style

### Go Guidelines

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable names
- Keep functions focused and small
- Handle errors explicitly

### Annotation Design

- Keep annotations simple and intuitive
- Maintain backward compatibility
- Follow existing patterns

## 🔍 Review Process

All submissions require review. We use GitHub pull requests for this purpose.

### What We Look For

- **Code quality**: Clean, readable, maintainable
- **Test coverage**: Adequate tests for new functionality
- **Documentation**: Clear documentation and examples
- **Backward compatibility**: No breaking changes without good reason

## 🌟 Recognition

Contributors will be:

- Listed in the project's contributor list
- Mentioned in release notes for significant contributions
- Invited to help shape the project's future direction

## 🤝 Community Guidelines

- **Be respectful** and inclusive
- **Help others** learn and contribute
- **Share knowledge** and best practices
- **Give constructive feedback**

## 📞 Getting Help

- **GitHub Issues**: For bugs and feature requests
- **Discussions**: For general questions and ideas
- **Email**: For security-related issues

## 🎯 Contribution Ideas

Looking for ways to contribute? Here are some ideas:

### Beginner Friendly

- Add support for new web frameworks
- Improve error messages
- Add more examples to documentation
- Fix typos or improve clarity

### Intermediate

- Implement new annotation types
- Add output format options
- Improve test coverage
- Optimize performance

### Advanced

- Add gRPC support
- Implement OpenAPI integration
- Create CI/CD workflows
- Add WebSocket support

Thank you for contributing! 🙏
