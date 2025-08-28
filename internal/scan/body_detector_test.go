package scan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestDetectJSONBody_ShouldBindJSON(t *testing.T) {
	code := `
package main

import "github.com/gin-gonic/gin"

func CreateUser(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Find the CreateUser function
	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "CreateUser" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("CreateUser function not found")
	}

	result := DetectJSONBody(fn, fset)

	if !result.HasBody {
		t.Error("Expected to detect JSON body, but didn't")
	}

	// Should generate generic JSON since variable is "req"
	expected := `{"data":"string","parameters":{}}`
	if result.BodyExample != expected {
		t.Errorf("Expected body example %q, got %q", expected, result.BodyExample)
	}
}

func TestDetectJSONBody_JSONNewDecoder(t *testing.T) {
	code := `
package main

import (
	"encoding/json"
	"net/http"
)

func HandlePayment(w http.ResponseWriter, r *http.Request) {
	var payment map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		return
	}
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "HandlePayment" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("HandlePayment function not found")
	}

	result := DetectJSONBody(fn, fset)

	if !result.HasBody {
		t.Error("Expected to detect JSON body, but didn't")
	}

	// Should generate generic JSON based on variable name "payment"
	expected := `{"id":"string","name":"string","value":"string","timestamp":"2024-01-01T00:00:00Z"}`
	if result.BodyExample != expected {
		t.Errorf("Expected body example %q, got %q", expected, result.BodyExample)
	}
}

func TestDetectJSONBody_IOReadAll(t *testing.T) {
	code := `
package main

import (
	"io"
	"net/http"
)

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "HandleWebhook" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("HandleWebhook function not found")
	}

	result := DetectJSONBody(fn, fset)

	if !result.HasBody {
		t.Error("Expected to detect JSON body, but didn't")
	}

	// Should generate generic JSON based on fallback
	expected := `{"data":"string","parameters":{}}`
	if result.BodyExample != expected {
		t.Errorf("Expected body example %q, got %q", expected, result.BodyExample)
	}
}

func TestDetectJSONBody_JSONUnmarshal(t *testing.T) {
	code := `
package main

import (
	"encoding/json"
)

func ProcessUser(data []byte) {
	var user map[string]interface{}
	json.Unmarshal(data, &user)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "ProcessUser" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("ProcessUser function not found")
	}

	result := DetectJSONBody(fn, fset)

	if !result.HasBody {
		t.Error("Expected to detect JSON body, but didn't")
	}

	// Should generate user-specific JSON
	expected := `{"name":"string","email":"string","id":"string"}`
	if result.BodyExample != expected {
		t.Errorf("Expected body example %q, got %q", expected, result.BodyExample)
	}
}

func TestDetectJSONBody_NoJSONProcessing(t *testing.T) {
	code := `
package main

func GetUser(id string) string {
	return "user-" + id
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "GetUser" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("GetUser function not found")
	}

	result := DetectJSONBody(fn, fset)

	if result.HasBody {
		t.Error("Expected NOT to detect JSON body, but did")
	}

	if result.BodyExample != "" {
		t.Errorf("Expected empty body example, got %q", result.BodyExample)
	}
}

func TestGenerateBodyByVariableName(t *testing.T) {
	tests := []struct {
		varName  string
		expected string
	}{
		{
			varName:  "user",
			expected: `{"name":"string","email":"string","id":"string"}`,
		},
		{
			varName:  "userRequest",
			expected: `{"name":"string","email":"string","id":"string"}`,
		},
		{
			varName:  "createRequest",
			expected: `{"name":"string","value":"string","type":"string"}`,
		},
		{
			varName:  "postData",
			expected: `{"name":"string","value":"string","type":"string"}`,
		},
		{
			varName:  "updateRequest",
			expected: `{"id":"string","name":"string","value":"string"}`,
		},
		{
			varName:  "putData",
			expected: `{"id":"string","name":"string","value":"string"}`,
		},
		{
			varName:  "patchRequest",
			expected: `{"id":"string","name":"string","value":"string"}`,
		},
		{
			varName:  "deleteRequest",
			expected: `{"id":"string","reason":"string"}`,
		},
		{
			varName:  "request",
			expected: `{"data":"string","parameters":{}}`,
		},
		{
			varName:  "req",
			expected: `{"data":"string","parameters":{}}`,
		},
		{
			varName:  "genericData",
			expected: `{"id":"string","name":"string","value":"string","timestamp":"2024-01-01T00:00:00Z"}`,
		},
		{
			varName:  "payload",
			expected: `{"id":"string","name":"string","value":"string","timestamp":"2024-01-01T00:00:00Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.varName, func(t *testing.T) {
			result := generateBodyByVariableName(tt.varName)
			if result != tt.expected {
				t.Errorf("For variable %q, expected %q, got %q", tt.varName, tt.expected, result)
			}
		})
	}
}

func TestCheckGinJSONBinding(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{
			code:     "c.ShouldBindJSON(&req)",
			expected: true,
		},
		{
			code:     "c.BindJSON(&req)",
			expected: true,
		},
		{
			code:     "c.ShouldBind(&req)",
			expected: true,
		},
		{
			code:     "c.Bind(&req)",
			expected: true,
		},
		{
			code:     "c.ShouldBindQuery(&req)",
			expected: false,
		},
		{
			code:     "someOtherMethod(&req)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			expr := parseCallExpr(t, tt.code)
			result := checkGinJSONBinding(expr)
			if result != tt.expected {
				t.Errorf("For %q, expected %v, got %v", tt.code, tt.expected, result)
			}
		})
	}
}

func TestCheckJSONDecoder(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{
			code:     "json.NewDecoder(r.Body).Decode(&req)",
			expected: true,
		},
		{
			code:     "decoder.Decode(&req)",
			expected: false,
		},
		{
			code:     "json.Decode(&req)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			expr := parseCallExpr(t, tt.code)
			result := checkJSONDecoder(expr)
			if result != tt.expected {
				t.Errorf("For %q, expected %v, got %v", tt.code, tt.expected, result)
			}
		})
	}
}

func TestCheckJSONUnmarshal(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{
			code:     "json.Unmarshal(data, &req)",
			expected: true,
		},
		{
			code:     "Unmarshal(data, &req)",
			expected: false,
		},
		{
			code:     "json.Marshal(req)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			expr := parseCallExpr(t, tt.code)
			result := checkJSONUnmarshal(expr)
			if result != tt.expected {
				t.Errorf("For %q, expected %v, got %v", tt.code, tt.expected, result)
			}
		})
	}
}

func TestCheckIOReadAll(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{
			code:     "io.ReadAll(r.Body)",
			expected: true,
		},
		{
			code:     "ReadAll(r.Body)",
			expected: false,
		},
		{
			code:     "io.ReadFile(filename)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			expr := parseCallExpr(t, tt.code)
			result := checkIOReadAll(expr)
			if result != tt.expected {
				t.Errorf("For %q, expected %v, got %v", tt.code, tt.expected, result)
			}
		})
	}
}

func TestDetectBodyFromFunction(t *testing.T) {
	code := `
package main

import "github.com/gin-gonic/gin"

func CreatePayment(c *gin.Context) {
	var payment map[string]interface{}
	c.ShouldBindJSON(&payment)
}

func GetPayment(c *gin.Context) {
	id := c.Param("id")
	// No JSON processing
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	results := map[string]string{}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name != nil {
			body := DetectBodyFromFunction(fn, fset)
			if body != "" {
				results[fn.Name.Name] = body
			}
		}
	}

	// CreatePayment should have a body
	if body, exists := results["CreatePayment"]; !exists {
		t.Error("Expected CreatePayment to have a body")
	} else {
		expected := `{"id":"string","name":"string","value":"string","timestamp":"2024-01-01T00:00:00Z"}`
		if body != expected {
			t.Errorf("CreatePayment body: expected %q, got %q", expected, body)
		}
	}

	// GetPayment should NOT have a body
	if body, exists := results["GetPayment"]; exists {
		t.Errorf("Expected GetPayment to NOT have a body, but got %q", body)
	}
}

// Helper function to parse a call expression from a string
func parseCallExpr(t *testing.T, code string) *ast.CallExpr {
	t.Helper()

	// Wrap in a function to make it valid Go code
	fullCode := "package main\nfunc test() {\n" + code + "\n}"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", fullCode, 0)
	if err != nil {
		t.Fatalf("Failed to parse code %q: %v", code, err)
	}

	// Find the call expression
	var callExpr *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok && callExpr == nil {
			callExpr = ce
			return false
		}
		return true
	})

	if callExpr == nil {
		t.Fatalf("No call expression found in %q", code)
	}

	return callExpr
}

func TestScanFunctionsForBodies(t *testing.T) {
	code := `
package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
)

func CreatePayment(c *gin.Context) {
	var payment map[string]interface{}
	c.ShouldBindJSON(&payment)
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook map[string]interface{}
	json.NewDecoder(r.Body).Decode(&webhook)
}

func GetPayment(c *gin.Context) {
	// No JSON processing
	id := c.Param("id")
}

func ProcessUser(data []byte) {
	var user map[string]interface{}
	json.Unmarshal(data, &user)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	results := scanFunctionsForBodies(file, fset)

	// Check that we found the right functions with bodies
	expectedFunctions := []string{"CreatePayment", "HandleWebhook", "ProcessUser"}

	for _, funcName := range expectedFunctions {
		if body, exists := results[funcName]; !exists {
			t.Errorf("Expected function %q to have a body detected", funcName)
		} else if body == "" {
			t.Errorf("Expected function %q to have a non-empty body", funcName)
		}
	}

	// Check that GetPayment doesn't have a body
	if body, exists := results["GetPayment"]; exists {
		t.Errorf("Expected function GetPayment to NOT have a body, but got %q", body)
	}

	// Verify specific body content
	if body, exists := results["CreatePayment"]; exists {
		if !strings.Contains(body, "id") || !strings.Contains(body, "name") {
			t.Errorf("CreatePayment body should be generic, got %q", body)
		}
	}

	if body, exists := results["ProcessUser"]; exists {
		if !strings.Contains(body, "name") || !strings.Contains(body, "email") {
			t.Errorf("ProcessUser body should be user-specific, got %q", body)
		}
	}
}
