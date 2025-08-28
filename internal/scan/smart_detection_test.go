package scan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestSmartStructDetection_InlineStruct(t *testing.T) {
	code := `
package main

import "encoding/json"

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user struct {
		Name     string  ` + "`json:\"name\"`" + `
		Email    string  ` + "`json:\"email\"`" + `
		Age      int     ` + "`json:\"age\"`" + `
		Active   bool    ` + "`json:\"active\"`" + `
		Score    float64 ` + "`json:\"score\"`" + `
		Tags     []string ` + "`json:\"tags\"`" + `
		Metadata map[string]interface{} ` + "`json:\"metadata\"`" + `
	}
	json.NewDecoder(r.Body).Decode(&user)
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

	// Should generate JSON based on actual struct fields
	expectedFields := []string{
		`"name":"string"`,
		`"email":"string"`,
		`"age":0`,
		`"active":false`,
		`"score":0.0`,
		`"tags":["string"]`,
		`"metadata":{}`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(result.BodyExample, field) {
			t.Errorf("Expected body to contain %q, but got %q", field, result.BodyExample)
		}
	}
}

func TestSmartStructDetection_SimpleTypes(t *testing.T) {
	code := `
package main

import "encoding/json"

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var product struct {
		ID          int     ` + "`json:\"id\"`" + `
		Name        string  ` + "`json:\"name\"`" + `
		Price       float64 ` + "`json:\"price\"`" + `
		Available   bool    ` + "`json:\"available\"`" + `
		Categories  []string ` + "`json:\"categories\"`" + `
	}
	json.NewDecoder(r.Body).Decode(&product)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		if f, ok := decl.(*ast.FuncDecl); ok && f.Name.Name == "CreateProduct" {
			fn = f
			break
		}
	}

	if fn == nil {
		t.Fatal("CreateProduct function not found")
	}

	result := DetectJSONBody(fn, fset)

	if !result.HasBody {
		t.Error("Expected to detect JSON body, but didn't")
	}

	// Verify specific type mappings
	testCases := []struct {
		field    string
		expected string
	}{
		{"id", "0"},
		{"name", `"string"`},
		{"price", "0.0"},
		{"available", "false"},
		{"categories", `["string"]`},
	}

	for _, tc := range testCases {
		expectedField := `"` + tc.field + `":` + tc.expected
		if !strings.Contains(result.BodyExample, expectedField) {
			t.Errorf("Expected body to contain %q, but got %q", expectedField, result.BodyExample)
		}
	}
}

func TestSmartStructDetection_NoStruct_FallbackToVariableName(t *testing.T) {
	code := `
package main

import "encoding/json"

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user map[string]interface{}
	json.NewDecoder(r.Body).Decode(&user)
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

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

	// Should fall back to variable name analysis since no struct was found
	expected := `{"name":"string","email":"string","id":"string"}`
	if result.BodyExample != expected {
		t.Errorf("Expected fallback body %q, got %q", expected, result.BodyExample)
	}
}

func TestAnalyzeInlineStruct(t *testing.T) {
	code := `
struct {
	Name     string  ` + "`json:\"full_name\"`" + `
	Age      int     ` + "`json:\"age\"`" + `
	Active   bool    ` + "`json:\"is_active\"`" + `
	Tags     []string ` + "`json:\"tags,omitempty\"`" + `
	NoTag    string
}
`

	expr, err := parser.ParseExpr(code)
	if err != nil {
		t.Fatalf("Failed to parse struct: %v", err)
	}

	structType, ok := expr.(*ast.StructType)
	if !ok {
		t.Fatal("Expected struct type")
	}

	info := analyzeInlineStruct(structType, "TestStruct")

	expectedFields := map[string]struct {
		jsonTag string
		goType  string
	}{
		"Name":   {"full_name", "string"},
		"Age":    {"age", "int"},
		"Active": {"is_active", "bool"},
		"Tags":   {"tags", "[]string"},
		"NoTag":  {"notag", "string"}, // Should use lowercase field name
	}

	if len(info.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(info.Fields))
	}

	for _, field := range info.Fields {
		expected, exists := expectedFields[field.Name]
		if !exists {
			t.Errorf("Unexpected field: %s", field.Name)
			continue
		}

		if field.JSONTag != expected.jsonTag {
			t.Errorf("Field %s: expected JSON tag %q, got %q", field.Name, expected.jsonTag, field.JSONTag)
		}

		if field.Type != expected.goType {
			t.Errorf("Field %s: expected type %q, got %q", field.Name, expected.goType, field.Type)
		}
	}
}

func TestGenerateValueForType(t *testing.T) {
	testCases := []struct {
		goType   string
		expected string
	}{
		{"string", `"string"`},
		{"int", "0"},
		{"int32", "0"},
		{"int64", "0"},
		{"uint", "0"},
		{"float32", "0.0"},
		{"float64", "0.0"},
		{"bool", "false"},
		{"[]string", `["string"]`},
		{"[]int", "[0]"},
		{"map[string]interface{}", "{}"},
		{"interface{}", `"string"`},
		{"any", `"string"`},
		{"CustomType", `"string"`},
	}

	for _, tc := range testCases {
		t.Run(tc.goType, func(t *testing.T) {
			result := generateValueForType(tc.goType)
			if result != tc.expected {
				t.Errorf("For type %q, expected %q, got %q", tc.goType, tc.expected, result)
			}
		})
	}
}

func TestExtractJSONTag(t *testing.T) {
	testCases := []struct {
		tag      string
		expected string
	}{
		{`json:"name"`, "name"},
		{`json:"full_name"`, "full_name"},
		{`json:"name,omitempty"`, "name"},
		{`json:",omitempty"`, ""},
		{`json:"-"`, ""},
		{`json:"name" validate:"required"`, "name"},
		{`validate:"required" json:"name"`, "name"},
		{`xml:"name"`, ""},
		{``, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {
			result := extractJSONTag(tc.tag)
			if result != tc.expected {
				t.Errorf("For tag %q, expected %q, got %q", tc.tag, tc.expected, result)
			}
		})
	}
}
