package scan

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Endpoint struct {
	Method     string            // HTTP method: GET, POST, etc.
	Path       string            // Path: /v1/users/{id}
	SourceFile string            // Source file where it was detected
	Handler    string            // Handler name when available
	Desc       string            // Optional description (from @route)
	Headers    map[string]string // @header Key: Value
	BodyRaw    string            // @body {...} (raw JSON - single line)
	Tags       []string          // @tag users
	Type       string            // "REST", "GraphQL", "RPC"
	GraphQL    *GraphQLInfo      // GraphQL specific information
}

type GraphQLInfo struct {
	Operation string // "query", "mutation", "subscription"
	Schema    string // GraphQL schema definition
	Query     string // GraphQL query example
	Variables string // Variables example (JSON)
}

var verbSet = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "DELETE": {}, "PATCH": {}, "HEAD": {}, "OPTIONS": {},
}

var (
	routeRe     = regexp.MustCompile(`(?i)@route\s+([A-Z]+)\s+(\S+)(?:\s+(.*))?$`)
	headerRe    = regexp.MustCompile(`(?i)@header\s+([^:]+):\s*(.+)$`)
	bodyRe      = regexp.MustCompile(`(?i)@body\s+(.+)$`)
	tagRe       = regexp.MustCompile(`(?i)@tag\s+([A-Za-z0-9_.\-\/]+)$`)
	graphqlRe   = regexp.MustCompile(`(?i)@graphql\s+(query|mutation|subscription)\s+(\S+)(?:\s+(.*))?$`)
	schemaRe    = regexp.MustCompile(`(?i)@schema\s+(.+)$`)
	queryRe     = regexp.MustCompile(`(?i)@query\s+(.+)$`)
	variablesRe = regexp.MustCompile(`(?i)@variables\s+(.+)$`)
	restRe      = regexp.MustCompile(`(?i)@rest\s+([A-Z]+)\s+(\S+)(?:\s+(.*))?$`)
)

// ScanDir: heuristic scanning (without type-checking)
func ScanDir(root string) ([]Endpoint, error) {
	fset := token.NewFileSet()
	var endpoints []Endpoint
	seen := make(map[string]struct{})

	add := func(e Endpoint) {
		if e.Method == "" {
			e.Method = "ANY"
		}
		if !strings.HasPrefix(e.Path, "/") {
			e.Path = "/" + e.Path
		}
		// Set default type if not specified
		if e.Type == "" {
			e.Type = "REST"
		}
		key := strings.ToUpper(e.Method) + " " + e.Path + " " + e.SourceFile + " " + strings.Join(e.Tags, ",")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		e.Method = strings.ToUpper(e.Method)
		endpoints = append(endpoints, e)
	}

	// First, analyze the entire project to understand its structure
	projectAnalysis, projectErr := AnalyzeProject(root)
	if projectErr != nil {
		// If project analysis fails, continue with the old method
		projectAnalysis = nil
	} else {
		// Set global project analysis for use in body detection
		globalProjectAnalysis = projectAnalysis
	}

	// Global function bodies map to store all detected bodies across files
	globalFunctionBodies := make(map[string]string)

	// First pass: collect all function bodies
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}

		// Collect function bodies from this file
		fileFunctionBodies := scanFunctionsForBodies(file, fset)
		for funcName, body := range fileFunctionBodies {
			globalFunctionBodies[funcName] = body
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: scan for endpoints and use global function bodies
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, perr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}

		anns, _ := scanAnnotationsFromFile(file, path)
		for _, a := range anns {
			add(a)
		}

		// Use global function bodies (already collected in first pass)

		// calls
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				sel := fun.Sel.Name

				// Special case: *.Methods("GET", "POST") chained from HandleFunc
				if sel == "Methods" && len(call.Args) >= 1 {
					if selExpr, ok := call.Fun.(*ast.SelectorExpr); ok {
						if innerCall, ok := selExpr.X.(*ast.CallExpr); ok {
							if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok {
								if (innerSel.Sel.Name == "HandleFunc" || innerSel.Sel.Name == "Handle") && len(innerCall.Args) >= 1 {
									if pathLit, ok := innerCall.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
										if p, err := strconv.Unquote(pathLit.Value); err == nil && isValidEndpointPath(p) {
											methods := stringArgs(call.Args)
											for _, m := range methods {
												add(Endpoint{Method: m, Path: p, SourceFile: fset.Position(call.Pos()).Filename, Handler: guessHandlerName(innerCall), Headers: map[string]string{}, Type: "REST"})
											}
										}
									}
								}
							}
						}
					}
				}

				// chi-like: r.Get("/path", handler)
				if isVerb(sel) && len(call.Args) >= 1 {
					if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
						if p, err := strconv.Unquote(pathLit.Value); err == nil && isValidEndpointPath(p) {
							handler := guessHandlerName(call)
							body := ""
							if handler != "" && globalFunctionBodies[handler] != "" {
								body = globalFunctionBodies[handler]
							}
							add(Endpoint{
								Method:     strings.ToUpper(sel),
								Path:       p,
								SourceFile: fset.Position(call.Pos()).Filename,
								Handler:    handler,
								Headers:    map[string]string{},
								BodyRaw:    body,
								Type:       "REST",
							})
						}
					}
				}

				// GraphQL endpoints detection (only for POST method)
				if sel == "POST" && len(call.Args) >= 1 {
					if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
						if p, err := strconv.Unquote(pathLit.Value); err == nil && isValidEndpointPath(p) {
							// Common GraphQL endpoint patterns
							if strings.Contains(strings.ToLower(p), "graphql") ||
								strings.Contains(strings.ToLower(p), "graph") ||
								strings.HasSuffix(strings.ToLower(p), "/query") {
								add(Endpoint{
									Method:     "POST",
									Path:       p,
									SourceFile: fset.Position(call.Pos()).Filename,
									Handler:    guessHandlerName(call),
									Headers:    map[string]string{},
									Type:       "GraphQL",
									GraphQL: &GraphQLInfo{
										Operation: "query", // Default to query
									},
								})
							} else {
								handler := guessHandlerName(call)
								body := ""
								if handler != "" && globalFunctionBodies[handler] != "" {
									body = globalFunctionBodies[handler]
								}
								add(Endpoint{
									Method:     "POST",
									Path:       p,
									SourceFile: fset.Position(call.Pos()).Filename,
									Handler:    handler,
									Headers:    map[string]string{},
									BodyRaw:    body,
									Type:       "REST",
								})
							}
						}
					}
				}

				// net/http & gorilla: *.HandleFunc("/path", h)
				if sel == "HandleFunc" && len(call.Args) >= 1 {
					if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
						if p, err := strconv.Unquote(pathLit.Value); err == nil && isValidEndpointPath(p) {
							methods := findChainedMethods(n)
							handler := guessHandlerName(call)
							body := ""
							if handler != "" && globalFunctionBodies[handler] != "" {
								body = globalFunctionBodies[handler]
							}
							if len(methods) == 0 {
								add(Endpoint{Method: "ANY", Path: p, SourceFile: fset.Position(call.Pos()).Filename, Handler: handler, Headers: map[string]string{}, BodyRaw: body, Type: "REST"})
							} else {
								for _, m := range methods {
									// Only add body for methods that typically use them
									methodBody := ""
									if (m == "POST" || m == "PUT" || m == "PATCH") && body != "" {
										methodBody = body
									}
									add(Endpoint{Method: m, Path: p, SourceFile: fset.Position(call.Pos()).Filename, Handler: handler, Headers: map[string]string{}, BodyRaw: methodBody, Type: "REST"})
								}
							}
						}
					}
				}

				// *.Handle("/path", h)
				if sel == "Handle" && len(call.Args) >= 1 {
					if pathLit, ok := call.Args[0].(*ast.BasicLit); ok && pathLit.Kind == token.STRING {
						if p, err := strconv.Unquote(pathLit.Value); err == nil && isValidEndpointPath(p) {
							methods := findChainedMethods(n)
							handler := guessHandlerName(call)
							body := ""
							if handler != "" && globalFunctionBodies[handler] != "" {
								body = globalFunctionBodies[handler]
							}
							if len(methods) == 0 {
								add(Endpoint{Method: "ANY", Path: p, SourceFile: fset.Position(call.Pos()).Filename, Handler: handler, Headers: map[string]string{}, BodyRaw: body, Type: "REST"})
							} else {
								for _, m := range methods {
									// Only add body for methods that typically use them
									methodBody := ""
									if (m == "POST" || m == "PUT" || m == "PATCH") && body != "" {
										methodBody = body
									}
									add(Endpoint{Method: m, Path: p, SourceFile: fset.Position(call.Pos()).Filename, Handler: handler, Headers: map[string]string{}, BodyRaw: methodBody, Type: "REST"})
								}
							}
						}
					}
				}
			}
			return true
		})

		return nil
	})
	return endpoints, err
}

// reading annotations
func scanAnnotationsFromFile(file *ast.File, sourcePath string) ([]Endpoint, error) {
	var res []Endpoint

	for _, cg := range file.Comments {
		// First pass: collect all annotations
		var annotations []string
		lines := strings.Split(cg.Text(), "\n")
		for _, raw := range lines {
			line := strings.TrimSpace(raw)
			if line == "" {
				continue
			}
			// Check if line matches any annotation pattern
			if headerRe.MatchString(line) || bodyRe.MatchString(line) || tagRe.MatchString(line) ||
				schemaRe.MatchString(line) || queryRe.MatchString(line) || variablesRe.MatchString(line) ||
				graphqlRe.MatchString(line) || restRe.MatchString(line) || routeRe.MatchString(line) {
				annotations = append(annotations, line)
			}
		}

		// Second pass: process annotations and find route definitions
		var routes []struct {
			method, path, desc, routeType string
			operation                     string // for GraphQL
		}

		// Find all route definitions first
		for _, line := range annotations {
			if m := graphqlRe.FindStringSubmatch(line); len(m) > 0 {
				operation := strings.ToLower(m[1])
				path := m[2]
				desc := ""
				if len(m) >= 4 {
					desc = strings.TrimSpace(m[3])
				}
				routes = append(routes, struct {
					method, path, desc, routeType string
					operation                     string
				}{"POST", path, desc, "GraphQL", operation})
			}
			if m := restRe.FindStringSubmatch(line); len(m) > 0 {
				method := strings.ToUpper(m[1])
				path := m[2]
				desc := ""
				if len(m) >= 4 {
					desc = strings.TrimSpace(m[3])
				}
				if _, ok := verbSet[method]; !ok {
					method = "ANY"
				}
				routes = append(routes, struct {
					method, path, desc, routeType string
					operation                     string
				}{method, path, desc, "REST", ""})
			}
			if m := routeRe.FindStringSubmatch(line); len(m) > 0 {
				method := strings.ToUpper(m[1])
				path := m[2]
				desc := ""
				if len(m) >= 4 {
					desc = strings.TrimSpace(m[3])
				}
				if _, ok := verbSet[method]; !ok {
					method = "ANY"
				}
				routes = append(routes, struct {
					method, path, desc, routeType string
					operation                     string
				}{method, path, desc, "REST", ""})
			}
		}

		// Now collect other annotations for all routes
		accHeaders := map[string]string{}
		accBody := ""
		var accTags []string
		var accGraphQL *GraphQLInfo

		for _, line := range annotations {
			// Headers
			if m := headerRe.FindStringSubmatch(line); len(m) > 0 {
				k := strings.TrimSpace(m[1])
				v := strings.TrimSpace(m[2])
				if k != "" {
					accHeaders[k] = v
				}
				continue
			}

			// Body
			if m := bodyRe.FindStringSubmatch(line); len(m) > 0 {
				accBody = strings.TrimSpace(m[1])
				continue
			}

			// Tags
			if m := tagRe.FindStringSubmatch(line); len(m) > 0 {
				tag := strings.TrimSpace(m[1])
				if tag != "" && !contains(accTags, tag) {
					accTags = append(accTags, tag)
				}
				continue
			}

			// GraphQL Schema
			if m := schemaRe.FindStringSubmatch(line); len(m) > 0 {
				if accGraphQL == nil {
					accGraphQL = &GraphQLInfo{}
				}
				accGraphQL.Schema = strings.TrimSpace(m[1])
				continue
			}

			// GraphQL Query
			if m := queryRe.FindStringSubmatch(line); len(m) > 0 {
				if accGraphQL == nil {
					accGraphQL = &GraphQLInfo{}
				}
				accGraphQL.Query = strings.TrimSpace(m[1])
				continue
			}

			// GraphQL Variables
			if m := variablesRe.FindStringSubmatch(line); len(m) > 0 {
				if accGraphQL == nil {
					accGraphQL = &GraphQLInfo{}
				}
				accGraphQL.Variables = strings.TrimSpace(m[1])
				continue
			}
		}

		// Create endpoints for all routes with collected annotations
		for _, route := range routes {
			hcopy := map[string]string{}
			for k, v := range accHeaders {
				hcopy[k] = v
			}
			tcopy := append([]string(nil), accTags...)

			if route.routeType == "GraphQL" {
				if accGraphQL == nil {
					accGraphQL = &GraphQLInfo{}
				}
				if accGraphQL.Operation == "" {
					accGraphQL.Operation = route.operation
				}

				res = append(res, Endpoint{
					Method:     route.method,
					Path:       route.path,
					SourceFile: sourcePath,
					Desc:       route.desc,
					Headers:    hcopy,
					BodyRaw:    accBody,
					Tags:       tcopy,
					Type:       "GraphQL",
					GraphQL:    accGraphQL,
				})
			} else {
				res = append(res, Endpoint{
					Method:     route.method,
					Path:       route.path,
					SourceFile: sourcePath,
					Desc:       route.desc,
					Headers:    hcopy,
					BodyRaw:    accBody,
					Tags:       tcopy,
					Type:       "REST",
					GraphQL:    nil,
				})
			}
		}
	}
	return res, nil
}

func isVerb(s string) bool {
	_, ok := verbSet[strings.ToUpper(s)]
	return ok
}

func guessHandlerName(call *ast.CallExpr) string {
	if len(call.Args) >= 2 {
		switch a := call.Args[1].(type) {
		case *ast.Ident:
			return a.Name
		case *ast.SelectorExpr:
			return a.Sel.Name
		}
	}
	return ""
}

func findChainedMethods(n ast.Node) []string {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	if sel.Sel.Name == "Methods" {
		return stringArgs(call.Args)
	}
	if inner, ok := sel.X.(*ast.CallExpr); ok {
		if s2, ok := inner.Fun.(*ast.SelectorExpr); ok && s2.Sel.Name == "Methods" {
			return stringArgs(inner.Args)
		}
	}
	return nil
}

func stringArgs(args []ast.Expr) []string {
	var out []string
	for _, a := range args {
		if bl, ok := a.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			if s, err := strconv.Unquote(bl.Value); err == nil {
				s = strings.ToUpper(strings.TrimSpace(s))
				if _, ok := verbSet[s]; ok {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// isValidEndpointPath validates if a string is a valid HTTP endpoint path
func isValidEndpointPath(path string) bool {
	if path == "" {
		return false
	}

	// Must start with /
	if !strings.HasPrefix(path, "/") {
		return false
	}

	// Filter out common non-endpoint patterns
	invalidPatterns := []string{
		"/X-Request-ID",  // Headers mistaken as paths
		"/Content-Type",  // Headers mistaken as paths
		"/Authorization", // Headers mistaken as paths
		"/Accept",        // Headers mistaken as paths
		"/User-Agent",    // Headers mistaken as paths
	}

	for _, invalid := range invalidPatterns {
		if strings.EqualFold(path, invalid) {
			return false
		}
	}

	// Filter out paths that look like headers (contain uppercase words with hyphens)
	if strings.Count(path, "-") > 0 && strings.Count(path, "/") == 1 {
		// Path like "/X-Request-ID" - likely a header mistaken as path
		pathPart := strings.TrimPrefix(path, "/")
		if strings.Contains(pathPart, "-") && strings.Title(pathPart) == pathPart {
			return false
		}
	}

	// Must not be just a single character or very short
	if len(strings.TrimPrefix(path, "/")) < 2 {
		return false
	}

	return true
}

// scanFunctionsForBodies analyzes all functions in a file to detect JSON body usage
func scanFunctionsForBodies(file *ast.File, fset *token.FileSet) map[string]string {
	functionBodies := make(map[string]string)

	// Iterate through all function declarations
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name != nil {
				funcName := fn.Name.Name
				detectedBody := DetectBodyFromFunction(fn, fset)
				if detectedBody != "" {
					functionBodies[funcName] = detectedBody
				}
			}
		}
	}

	return functionBodies
}
