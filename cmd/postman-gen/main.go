package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/williamkoller/postman-gen/internal/postman"
	"github.com/williamkoller/postman-gen/internal/scan"
)

func main() {
	dir := flag.String("dir", ".", "Root directory of the Go project to scan")
	name := flag.String("name", "Go API", "Name of the Postman Collection")
	baseURL := flag.String("base-url", "http://localhost:8080", "Initial value for the {{baseUrl}} variable")
	out := flag.String("out", "", "Collection output file (empty = stdout)")
	groupDepth := flag.Int("group-depth", 1, "Folder grouping depth (0 = no grouping)")
	groupByMethod := flag.Bool("group-by-method", false, "Create HTTP method subfolders within folders")
	tagFolders := flag.Bool("tag-folders", false, "Create an auxiliary 'By Tag' tree grouping by @tag")
	useTypes := flag.Bool("use-types", true, "Use go/packages analysis to increase precision")
	buildTags := flag.String("build-tags", "", "Build tags (e.g.: \"dev,integration\") for typed analysis")
	envOut := flag.String("env-out", "", "Postman Environment output file (optional)")
	envName := flag.String("env-name", "Local", "Name of the Postman Environment")
	flag.Parse()

	var endpoints []scan.Endpoint
	var err error

	if *useTypes {
		endpoints, _ = scan.ScanDirWithOpts(scan.ScanOptions{
			Dir:       *dir,
			UseTypes:  true,
			BuildTags: *buildTags,
		})
	}

	if len(endpoints) == 0 { // fallback (or -use-types=false)
		endpoints, err = scan.ScanDir(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scanning %s: %v\n", *dir, err)
			os.Exit(1)
		}
	}

	if len(endpoints) == 0 {
		fmt.Fprintln(os.Stderr, "No endpoints found. Tip: use @route for dynamic routes.")
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			if endpoints[i].Method == endpoints[j].Method {
				return endpoints[i].SourceFile < endpoints[j].SourceFile
			}
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	col := postman.BuildCollection(postman.BuildOpts{
		Name:          *name,
		BaseURL:       *baseURL,
		GroupDepth:    *groupDepth,
		GroupByMethod: *groupByMethod,
		TagFolders:    *tagFolders,
	}, endpoints)

	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serializing Collection: %v\n", err)
		os.Exit(1)
	}

	if *out == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(*out, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing Collection: %v\n", err)
			os.Exit(1)
		}
	}

	if *envOut != "" {
		env := postman.BuildEnvironment(*envName, *baseURL)
		edata, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error serializing Environment: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*envOut, edata, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing Environment: %v\n", err)
			os.Exit(1)
		}
	}
}
