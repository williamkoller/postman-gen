package scan

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// Global project analysis - set by ScanDir
var globalProjectAnalysis *ProjectAnalysis

// BodyDetectionResult contains information about detected JSON bodies
type BodyDetectionResult struct {
	HasBody     bool
	BodyExample string
	StructName  string
}

// StructFieldInfo represents information about a struct field
type StructFieldInfo struct {
	Name     string
	Type     string
	JSONTag  string
	Required bool
}

// StructInfo contains analyzed struct information
type StructInfo struct {
	Name   string
	Fields []StructFieldInfo
}

// DetectJSONBody analyzes a function to detect if it expects a JSON body
func DetectJSONBody(fn *ast.FuncDecl, fset *token.FileSet) BodyDetectionResult {
	result := BodyDetectionResult{}

	if fn.Body == nil {
		return result
	}

	// First, scan for struct information in the function
	structInfo := scanStructUsage(fn)

	// Look for common JSON unmarshaling patterns
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Check for ShouldBindJSON, BindJSON, etc.
			if checkGinJSONBinding(node) {
				result.HasBody = true
				result.BodyExample = generateSmartBodyExample(node, structInfo)
				return false
			}

			// Check for json.NewDecoder(r.Body).Decode
			if checkJSONDecoder(node) {
				result.HasBody = true
				result.BodyExample = generateSmartBodyExample(node, structInfo)
				return false
			}

			// Check for json.Unmarshal
			if checkJSONUnmarshal(node) {
				result.HasBody = true
				result.BodyExample = generateSmartBodyExample(node, structInfo)
				return false
			}

			// Check for io.ReadAll pattern (often followed by json.Unmarshal)
			if checkIOReadAll(node) {
				result.HasBody = true
				result.BodyExample = generateSmartBodyExample(node, structInfo)
				return false
			}
		}
		return true
	})

	return result
}

// checkGinJSONBinding detects Gin framework JSON binding calls
func checkGinJSONBinding(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		return methodName == "ShouldBindJSON" ||
			methodName == "BindJSON" ||
			methodName == "ShouldBind" ||
			methodName == "Bind"
	}
	return false
}

// checkJSONDecoder detects standard library json.NewDecoder pattern
func checkJSONDecoder(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "Decode" {
			// Check if it's called on json.NewDecoder result
			if innerSel, ok := sel.X.(*ast.CallExpr); ok {
				if innerSelExpr, ok := innerSel.Fun.(*ast.SelectorExpr); ok {
					return innerSelExpr.Sel.Name == "NewDecoder"
				}
			}
		}
	}
	return false
}

// checkJSONUnmarshal detects json.Unmarshal calls
func checkJSONUnmarshal(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "json" && sel.Sel.Name == "Unmarshal"
		}
	}
	return false
}

// checkIOReadAll detects io.ReadAll calls (often used before json.Unmarshal)
func checkIOReadAll(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "io" && sel.Sel.Name == "ReadAll"
		}
	}
	return false
}

// generateBodyByVariableName creates JSON based on variable name patterns
func generateBodyByVariableName(varName string) string {
	lowerName := strings.ToLower(varName)

	switch {
	case strings.Contains(lowerName, "user"):
		return `{"name":"string","email":"string","id":"string"}`
	case strings.Contains(lowerName, "create") || strings.Contains(lowerName, "post"):
		return `{"name":"string","value":"string","type":"string"}`
	case strings.Contains(lowerName, "update") || strings.Contains(lowerName, "put") || strings.Contains(lowerName, "patch"):
		return `{"id":"string","name":"string","value":"string"}`
	case strings.Contains(lowerName, "delete"):
		return `{"id":"string","reason":"string"}`
	case strings.Contains(lowerName, "request") || strings.Contains(lowerName, "req"):
		return `{"data":"string","parameters":{}}`
	default:
		return `{"id":"string","name":"string","value":"string","timestamp":"2024-01-01T00:00:00Z"}`
	}
}

// scanStructUsage analyzes the function to find struct types being used
func scanStructUsage(fn *ast.FuncDecl) *StructInfo {
	var structInfo *StructInfo

	// Look for variable declarations with struct types
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			// Look for var declarations
			for _, spec := range node.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					if structType, ok := valueSpec.Type.(*ast.StructType); ok {
						// Found an inline struct declaration
						structInfo = analyzeInlineStruct(structType, "InlineStruct")
						return false
					}
				}
			}
		case *ast.AssignStmt:
			// Look for := assignments with struct literals
			for _, rhs := range node.Rhs {
				if compLit, ok := rhs.(*ast.CompositeLit); ok {
					if structType, ok := compLit.Type.(*ast.StructType); ok {
						structInfo = analyzeInlineStruct(structType, "InlineStruct")
						return false
					}
				}
			}
		}
		return true
	})

	return structInfo
}

// analyzeInlineStruct analyzes an inline struct type
func analyzeInlineStruct(structType *ast.StructType, name string) *StructInfo {
	info := &StructInfo{
		Name:   name,
		Fields: []StructFieldInfo{},
	}

	for _, field := range structType.Fields.List {
		fieldInfo := StructFieldInfo{
			Required: true, // Default to required
		}

		// Get field names
		if len(field.Names) > 0 {
			fieldInfo.Name = field.Names[0].Name
		}

		// Get field type
		fieldInfo.Type = getTypeString(field.Type)

		// Get JSON tag if present
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			if strings.Contains(tag, "json:") {
				fieldInfo.JSONTag = extractJSONTag(tag)
			}
		}

		// If no JSON tag, use field name in lowercase
		if fieldInfo.JSONTag == "" {
			fieldInfo.JSONTag = strings.ToLower(fieldInfo.Name)
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return info
}

// getTypeString converts an ast.Expr type to a string representation
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", getTypeString(t.X), t.Sel.Name)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", getTypeString(t.Key), getTypeString(t.Value))
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	default:
		return "interface{}"
	}
}

// extractJSONTag extracts the JSON field name from a struct tag
func extractJSONTag(tag string) string {
	// Look for json:"fieldname"
	parts := strings.Split(tag, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "json:") {
			jsonPart := strings.TrimPrefix(part, "json:")
			jsonPart = strings.Trim(jsonPart, "\"")
			// Handle json:",omitempty" or json:"fieldname,omitempty"
			if idx := strings.Index(jsonPart, ","); idx != -1 {
				jsonPart = jsonPart[:idx]
			}
			if jsonPart != "-" && jsonPart != "" {
				return jsonPart
			}
		}
	}
	return ""
}

// generateSmartBodyExample creates JSON based on actual struct analysis
func generateSmartBodyExample(call *ast.CallExpr, structInfo *StructInfo) string {
	// Try to use project-wide analysis first
	if globalProjectAnalysis != nil {
		if body := generateBodyFromProjectAnalysis(call, globalProjectAnalysis); body != "" {
			return body
		}
	}

	// Fallback to local struct analysis
	if structInfo != nil && len(structInfo.Fields) > 0 {
		return generateJSONFromStruct(structInfo)
	}

	// Fallback to variable name analysis
	if len(call.Args) > 0 {
		if unary, ok := call.Args[0].(*ast.UnaryExpr); ok {
			if ident, ok := unary.X.(*ast.Ident); ok {
				return generateBodyByVariableName(ident.Name)
			}
		}
		// For json.Unmarshal, second argument is the target
		if len(call.Args) > 1 {
			if unary, ok := call.Args[1].(*ast.UnaryExpr); ok {
				if ident, ok := unary.X.(*ast.Ident); ok {
					return generateBodyByVariableName(ident.Name)
				}
			}
		}
	}

	return `{"data":"string","parameters":{}}`
}

// generateJSONFromStruct creates a JSON example from struct field information
func generateJSONFromStruct(structInfo *StructInfo) string {
	if len(structInfo.Fields) == 0 {
		return `{}`
	}

	var jsonPairs []string
	for _, field := range structInfo.Fields {
		value := generateValueForType(field.Type)
		jsonPairs = append(jsonPairs, fmt.Sprintf(`"%s":%s`, field.JSONTag, value))
	}

	return "{" + strings.Join(jsonPairs, ",") + "}"
}

// generateValueForType generates an appropriate JSON value based on Go type
func generateValueForType(goType string) string {
	switch {
	case goType == "string":
		return `"string"`
	case goType == "int" || goType == "int32" || goType == "int64" ||
		goType == "uint" || goType == "uint32" || goType == "uint64":
		return "0"
	case goType == "float32" || goType == "float64":
		return "0.0"
	case goType == "bool":
		return "false"
	case strings.HasPrefix(goType, "[]"):
		elemType := strings.TrimPrefix(goType, "[]")
		elemValue := generateValueForType(elemType)
		return "[" + elemValue + "]"
	case strings.HasPrefix(goType, "map["):
		return "{}"
	case goType == "interface{}" || goType == "any":
		return `"string"`
	default:
		// For custom types, assume string
		return `"string"`
	}
}

// generateBodyFromProjectAnalysis generates JSON body using project-wide analysis
func generateBodyFromProjectAnalysis(call *ast.CallExpr, analysis *ProjectAnalysis) string {
	// Try to extract the variable type being decoded to
	var targetTypeName string

	if len(call.Args) > 0 {
		if unary, ok := call.Args[0].(*ast.UnaryExpr); ok {
			if ident, ok := unary.X.(*ast.Ident); ok {
				// Look for struct definitions that match this variable name or pattern
				for _, structDef := range analysis.Structs {
					lowerStructName := strings.ToLower(structDef.Name)
					lowerVarName := strings.ToLower(ident.Name)

					// Match by variable name pattern
					if strings.Contains(lowerStructName, lowerVarName) ||
						strings.Contains(lowerVarName, lowerStructName) ||
						isStructNameMatch(lowerVarName, lowerStructName) {
						return generateJSONFromProjectStruct(structDef)
					}
				}
				targetTypeName = ident.Name
			}
		}

		// For json.Unmarshal, second argument is the target
		if len(call.Args) > 1 {
			if unary, ok := call.Args[1].(*ast.UnaryExpr); ok {
				if ident, ok := unary.X.(*ast.Ident); ok {
					for _, structDef := range analysis.Structs {
						lowerStructName := strings.ToLower(structDef.Name)
						lowerVarName := strings.ToLower(ident.Name)

						if strings.Contains(lowerStructName, lowerVarName) ||
							strings.Contains(lowerVarName, lowerStructName) ||
							isStructNameMatch(lowerVarName, lowerStructName) {
							return generateJSONFromProjectStruct(structDef)
						}
					}
					targetTypeName = ident.Name
				}
			}
		}
	}

	// Look for DTOs that match common patterns
	if targetTypeName != "" {
		for _, dtoPattern := range analysis.ArchPattern.DTOPatterns {
			if strings.Contains(strings.ToLower(dtoPattern), strings.ToLower(targetTypeName)) {
				if structDef, exists := analysis.Structs[dtoPattern]; exists {
					return generateJSONFromProjectStruct(structDef)
				}
			}
		}
	}

	return ""
}

// isStructNameMatch checks if variable name matches struct name patterns
func isStructNameMatch(varName, structName string) bool {
	// Remove common suffixes from struct names for matching
	cleanStructName := structName
	suffixes := []string{"request", "req", "dto", "model", "entity", "response", "resp"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(cleanStructName, suffix) {
			cleanStructName = strings.TrimSuffix(cleanStructName, suffix)
			break
		}
	}

	return strings.Contains(varName, cleanStructName) || strings.Contains(cleanStructName, varName)
}

// generateJSONFromProjectStruct generates JSON from project-analyzed struct
func generateJSONFromProjectStruct(structDef *StructDefinition) string {
	if len(structDef.Fields) == 0 {
		return `{}`
	}

	var jsonPairs []string
	for _, field := range structDef.Fields {
		if field.JSONTag == "-" {
			continue // Skip fields marked as ignored
		}

		value := generateValueForType(field.Type)
		jsonTag := field.JSONTag
		if jsonTag == "" {
			jsonTag = strings.ToLower(field.Name)
		}
		jsonPairs = append(jsonPairs, fmt.Sprintf(`"%s":%s`, jsonTag, value))
	}

	return "{" + strings.Join(jsonPairs, ",") + "}"
}

// DetectBodyFromFunction analyzes a function declaration and detects JSON body patterns
func DetectBodyFromFunction(fn *ast.FuncDecl, fset *token.FileSet) string {
	result := DetectJSONBody(fn, fset)
	if result.HasBody {
		return result.BodyExample
	}
	return ""
}
