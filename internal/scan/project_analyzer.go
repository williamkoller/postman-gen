package scan

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ProjectAnalysis contains comprehensive analysis of the entire Go project
type ProjectAnalysis struct {
	Structs     map[string]*StructDefinition
	Interfaces  map[string]*InterfaceDefinition
	Functions   map[string]*FunctionInfo
	Types       map[string]*TypeDefinition
	Packages    map[string]*PackageInfo
	ModuleName  string
	ArchPattern ArchitecturePattern
}

// StructDefinition contains detailed information about a struct
type StructDefinition struct {
	Name       string
	Fields     []StructFieldInfo
	Package    string
	File       string
	IsExported bool
	Comments   []string
	Tags       map[string]string
}

// InterfaceDefinition contains information about interfaces
type InterfaceDefinition struct {
	Name       string
	Methods    []MethodInfo
	Package    string
	File       string
	IsExported bool
}

// FunctionInfo contains information about functions
type FunctionInfo struct {
	Name       string
	Package    string
	File       string
	Params     []ParamInfo
	Returns    []ParamInfo
	IsExported bool
	Comments   []string
	IsMethod   bool
	Receiver   *ParamInfo
}

// TypeDefinition contains information about custom types
type TypeDefinition struct {
	Name         string
	UnderlyingType string
	Package      string
	File         string
	IsExported   bool
}

// PackageInfo contains information about a package
type PackageInfo struct {
	Name      string
	Path      string
	Files     []string
	Imports   []string
	IsMain    bool
	HasTests  bool
}

// MethodInfo contains information about interface methods
type MethodInfo struct {
	Name    string
	Params  []ParamInfo
	Returns []ParamInfo
}

// ParamInfo contains information about parameters
type ParamInfo struct {
	Name string
	Type string
}

// ArchitecturePattern represents the detected architecture pattern
type ArchitecturePattern struct {
	Type        string   // "clean", "mvc", "layered", "microservice", "monolith"
	Layers      []string // detected layers/packages
	DTOPatterns []string // detected DTO/model patterns
	Confidence  float64  // confidence in detection (0-1)
}

// AnalyzeProject performs comprehensive analysis of the entire Go project
func AnalyzeProject(rootDir string) (*ProjectAnalysis, error) {
	analysis := &ProjectAnalysis{
		Structs:    make(map[string]*StructDefinition),
		Interfaces: make(map[string]*InterfaceDefinition),
		Functions:  make(map[string]*FunctionInfo),
		Types:      make(map[string]*TypeDefinition),
		Packages:   make(map[string]*PackageInfo),
	}

	fset := token.NewFileSet()

	// First pass: collect all Go files and basic package info
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Skip vendor, .git, and other common directories
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return analyzeFile(path, fset, analysis)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to analyze project: %w", err)
	}

	// Detect module name from go.mod
	analysis.ModuleName = detectModuleName(rootDir)

	// Detect architecture pattern
	analysis.ArchPattern = detectArchitecturePattern(analysis)

	// Resolve type references across packages
	resolveTypeReferences(analysis)

	return analysis, nil
}

// analyzeFile analyzes a single Go file
func analyzeFile(filePath string, fset *token.FileSet, analysis *ProjectAnalysis) error {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	packageName := file.Name.Name
	relPath, _ := filepath.Rel(filepath.Dir(filePath), filePath)

	// Initialize package info if not exists
	if analysis.Packages[packageName] == nil {
		analysis.Packages[packageName] = &PackageInfo{
			Name:    packageName,
			Path:    filepath.Dir(filePath),
			Files:   []string{},
			Imports: []string{},
			IsMain:  packageName == "main",
		}
	}

	pkg := analysis.Packages[packageName]
	pkg.Files = append(pkg.Files, relPath)

	// Collect imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		pkg.Imports = append(pkg.Imports, importPath)
	}

	// Analyze declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			analyzeGenDecl(d, packageName, filePath, analysis)
		case *ast.FuncDecl:
			analyzeFuncDecl(d, packageName, filePath, analysis)
		}
	}

	return nil
}

// analyzeGenDecl analyzes general declarations (types, vars, consts)
func analyzeGenDecl(decl *ast.GenDecl, packageName, filePath string, analysis *ProjectAnalysis) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			analyzeTypeSpec(s, decl, packageName, filePath, analysis)
		}
	}
}

// analyzeTypeSpec analyzes type specifications
func analyzeTypeSpec(spec *ast.TypeSpec, decl *ast.GenDecl, packageName, filePath string, analysis *ProjectAnalysis) {
	typeName := spec.Name.Name
	isExported := ast.IsExported(typeName)
	qualifiedName := packageName + "." + typeName

	// Extract comments
	var comments []string
	if decl.Doc != nil {
		for _, comment := range decl.Doc.List {
			comments = append(comments, strings.TrimPrefix(comment.Text, "//"))
		}
	}

	switch t := spec.Type.(type) {
	case *ast.StructType:
		// Analyze struct
		structDef := &StructDefinition{
			Name:       typeName,
			Fields:     []StructFieldInfo{},
			Package:    packageName,
			File:       filePath,
			IsExported: isExported,
			Comments:   comments,
			Tags:       make(map[string]string),
		}

		// Analyze struct fields
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldInfo := analyzeStructField(field)
				structDef.Fields = append(structDef.Fields, fieldInfo...)
			}
		}

		analysis.Structs[qualifiedName] = structDef

	case *ast.InterfaceType:
		// Analyze interface
		interfaceDef := &InterfaceDefinition{
			Name:       typeName,
			Methods:    []MethodInfo{},
			Package:    packageName,
			File:       filePath,
			IsExported: isExported,
		}

		// Analyze interface methods
		if t.Methods != nil {
			for _, method := range t.Methods.List {
				methodInfo := analyzeInterfaceMethod(method)
				if methodInfo != nil {
					interfaceDef.Methods = append(interfaceDef.Methods, *methodInfo)
				}
			}
		}

		analysis.Interfaces[qualifiedName] = interfaceDef

	default:
		// Other type definitions
		analysis.Types[qualifiedName] = &TypeDefinition{
			Name:           typeName,
			UnderlyingType: getTypeString(t),
			Package:        packageName,
			File:           filePath,
			IsExported:     isExported,
		}
	}
}

// analyzeStructField analyzes struct fields
func analyzeStructField(field *ast.Field) []StructFieldInfo {
	var fields []StructFieldInfo

	fieldType := getTypeString(field.Type)

	// Handle embedded fields or multiple fields with same type
	if len(field.Names) == 0 {
		// Embedded field
		fields = append(fields, StructFieldInfo{
			Name:     getTypeString(field.Type), // Use type as name for embedded
			Type:     fieldType,
			JSONTag:  "",
			Required: true,
		})
	} else {
		// Named fields
		for _, name := range field.Names {
			fieldInfo := StructFieldInfo{
				Name:     name.Name,
				Type:     fieldType,
				Required: true,
			}

			// Extract JSON tag
			if field.Tag != nil {
				tag := strings.Trim(field.Tag.Value, "`")
				fieldInfo.JSONTag = extractJSONTag(tag)
				if fieldInfo.JSONTag == "" {
					fieldInfo.JSONTag = strings.ToLower(name.Name)
				}
			} else {
				fieldInfo.JSONTag = strings.ToLower(name.Name)
			}

			fields = append(fields, fieldInfo)
		}
	}

	return fields
}

// analyzeInterfaceMethod analyzes interface methods
func analyzeInterfaceMethod(method *ast.Field) *MethodInfo {
	if len(method.Names) == 0 {
		return nil // Embedded interface
	}

	methodName := method.Names[0].Name
	methodInfo := &MethodInfo{
		Name:    methodName,
		Params:  []ParamInfo{},
		Returns: []ParamInfo{},
	}

	if funcType, ok := method.Type.(*ast.FuncType); ok {
		// Analyze parameters
		if funcType.Params != nil {
			for _, param := range funcType.Params.List {
				paramType := getTypeString(param.Type)
				if len(param.Names) == 0 {
					methodInfo.Params = append(methodInfo.Params, ParamInfo{
						Name: "",
						Type: paramType,
					})
				} else {
					for _, name := range param.Names {
						methodInfo.Params = append(methodInfo.Params, ParamInfo{
							Name: name.Name,
							Type: paramType,
						})
					}
				}
			}
		}

		// Analyze returns
		if funcType.Results != nil {
			for _, result := range funcType.Results.List {
				resultType := getTypeString(result.Type)
				if len(result.Names) == 0 {
					methodInfo.Returns = append(methodInfo.Returns, ParamInfo{
						Name: "",
						Type: resultType,
					})
				} else {
					for _, name := range result.Names {
						methodInfo.Returns = append(methodInfo.Returns, ParamInfo{
							Name: name.Name,
							Type: resultType,
						})
					}
				}
			}
		}
	}

	return methodInfo
}

// analyzeFuncDecl analyzes function declarations
func analyzeFuncDecl(decl *ast.FuncDecl, packageName, filePath string, analysis *ProjectAnalysis) {
	funcName := decl.Name.Name
	qualifiedName := packageName + "." + funcName

	funcInfo := &FunctionInfo{
		Name:       funcName,
		Package:    packageName,
		File:       filePath,
		Params:     []ParamInfo{},
		Returns:    []ParamInfo{},
		IsExported: ast.IsExported(funcName),
		Comments:   []string{},
		IsMethod:   decl.Recv != nil,
	}

	// Extract comments
	if decl.Doc != nil {
		for _, comment := range decl.Doc.List {
			funcInfo.Comments = append(funcInfo.Comments, strings.TrimPrefix(comment.Text, "//"))
		}
	}

	// Analyze receiver (for methods)
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		receiverType := getTypeString(recv.Type)
		receiverName := ""
		if len(recv.Names) > 0 {
			receiverName = recv.Names[0].Name
		}
		funcInfo.Receiver = &ParamInfo{
			Name: receiverName,
			Type: receiverType,
		}
	}

	// Analyze parameters
	if decl.Type.Params != nil {
		for _, param := range decl.Type.Params.List {
			paramType := getTypeString(param.Type)
			if len(param.Names) == 0 {
				funcInfo.Params = append(funcInfo.Params, ParamInfo{
					Name: "",
					Type: paramType,
				})
			} else {
				for _, name := range param.Names {
					funcInfo.Params = append(funcInfo.Params, ParamInfo{
						Name: name.Name,
						Type: paramType,
					})
				}
			}
		}
	}

	// Analyze returns
	if decl.Type.Results != nil {
		for _, result := range decl.Type.Results.List {
			resultType := getTypeString(result.Type)
			if len(result.Names) == 0 {
				funcInfo.Returns = append(funcInfo.Returns, ParamInfo{
					Name: "",
					Type: resultType,
				})
			} else {
				for _, name := range result.Names {
					funcInfo.Returns = append(funcInfo.Returns, ParamInfo{
						Name: name.Name,
						Type: resultType,
					})
				}
			}
		}
	}

	analysis.Functions[qualifiedName] = funcInfo
}

// shouldSkipDir determines if a directory should be skipped
func shouldSkipDir(dirName string) bool {
	skipDirs := []string{
		"vendor", ".git", ".vscode", ".idea", "node_modules",
		"__pycache__", ".pytest_cache", "build", "dist",
	}

	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}

	return strings.HasPrefix(dirName, ".")
}

// detectModuleName detects the module name from go.mod
func detectModuleName(rootDir string) string {
	goModPath := filepath.Join(rootDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return ""
}

// detectArchitecturePattern detects the architecture pattern used in the project
func detectArchitecturePattern(analysis *ProjectAnalysis) ArchitecturePattern {
	pattern := ArchitecturePattern{
		Type:        "unknown",
		Layers:      []string{},
		DTOPatterns: []string{},
		Confidence:  0.0,
	}

	packageNames := make([]string, 0, len(analysis.Packages))
	for name := range analysis.Packages {
		if name != "main" {
			packageNames = append(packageNames, name)
		}
	}

	// Detect Clean Architecture
	cleanScore := detectCleanArchitecture(packageNames)
	
	// Detect MVC
	mvcScore := detectMVC(packageNames)
	
	// Detect Layered Architecture
	layeredScore := detectLayeredArchitecture(packageNames)

	// Detect microservice patterns
	microserviceScore := detectMicroservicePattern(analysis)

	// Choose the pattern with highest confidence
	maxScore := cleanScore
	pattern.Type = "clean"
	pattern.Confidence = cleanScore

	if mvcScore > maxScore {
		maxScore = mvcScore
		pattern.Type = "mvc"
		pattern.Confidence = mvcScore
	}

	if layeredScore > maxScore {
		maxScore = layeredScore
		pattern.Type = "layered"
		pattern.Confidence = layeredScore
	}

	if microserviceScore > maxScore {
		pattern.Type = "microservice"
		pattern.Confidence = microserviceScore
	}

	pattern.Layers = packageNames
	pattern.DTOPatterns = detectDTOPatterns(analysis)

	return pattern
}

// detectCleanArchitecture detects Clean Architecture patterns
func detectCleanArchitecture(packages []string) float64 {
	score := 0.0
	total := 0.0

	// Look for typical Clean Architecture packages
	cleanPatterns := []string{
		"domain", "entity", "entities",
		"usecase", "usecases", "application",
		"repository", "repositories", "infrastructure",
		"handler", "handlers", "delivery", "transport",
		"service", "services",
	}

	for _, pattern := range cleanPatterns {
		total += 1.0
		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg), pattern) {
				score += 1.0
				break
			}
		}
	}

	if total == 0 {
		return 0.0
	}

	return score / total
}

// detectMVC detects MVC patterns
func detectMVC(packages []string) float64 {
	score := 0.0
	total := 3.0 // model, view, controller

	mvcPatterns := []string{"model", "view", "controller"}

	for _, pattern := range mvcPatterns {
		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg), pattern) {
				score += 1.0
				break
			}
		}
	}

	return score / total
}

// detectLayeredArchitecture detects layered architecture patterns
func detectLayeredArchitecture(packages []string) float64 {
	score := 0.0
	total := 0.0

	layerPatterns := []string{
		"api", "web", "http",
		"business", "logic", "service",
		"data", "dal", "persistence",
		"common", "shared", "utils",
	}

	for _, pattern := range layerPatterns {
		total += 1.0
		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg), pattern) {
				score += 1.0
				break
			}
		}
	}

	if total == 0 {
		return 0.0
	}

	return score / total
}

// detectMicroservicePattern detects microservice patterns
func detectMicroservicePattern(analysis *ProjectAnalysis) float64 {
	score := 0.0

	// Look for gRPC/protobuf usage
	for _, pkg := range analysis.Packages {
		for _, imp := range pkg.Imports {
			if strings.Contains(imp, "grpc") || strings.Contains(imp, "protobuf") {
				score += 0.3
			}
		}
	}

	// Look for main package (microservice entry point)
	if analysis.Packages["main"] != nil {
		score += 0.2
	}

	// Look for config/environment patterns
	for pkgName := range analysis.Packages {
		if strings.Contains(strings.ToLower(pkgName), "config") ||
		   strings.Contains(strings.ToLower(pkgName), "env") {
			score += 0.2
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// detectDTOPatterns detects common DTO/model patterns
func detectDTOPatterns(analysis *ProjectAnalysis) []string {
	patterns := []string{}

	// Look for common DTO suffixes in struct names
	for structName, structDef := range analysis.Structs {
		lowerName := strings.ToLower(structName)
		
		if strings.HasSuffix(lowerName, "request") ||
		   strings.HasSuffix(lowerName, "req") ||
		   strings.HasSuffix(lowerName, "dto") ||
		   strings.HasSuffix(lowerName, "model") ||
		   strings.HasSuffix(lowerName, "entity") ||
		   strings.HasSuffix(lowerName, "response") ||
		   strings.HasSuffix(lowerName, "resp") {
			
			patterns = append(patterns, structDef.Name)
		}
	}

	return patterns
}

// resolveTypeReferences resolves type references across packages
func resolveTypeReferences(analysis *ProjectAnalysis) {
	// This would implement cross-package type resolution
	// For now, we'll keep it simple and focus on the current implementation
} 