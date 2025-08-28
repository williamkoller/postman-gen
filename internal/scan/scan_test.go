package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDir_AnnotationsHeadersBodyTags(t *testing.T) {
	dir := t.TempDir()

	code := `package main

// @header Authorization: Bearer {{token}}
// @header X-Correlation-ID: abc123
// @body {"name":"alice","age":30}
// @tag users
// @tag v1
// @route POST /v1/users Criar usuário

// @route GET /v1/users Listar usuários
// @tag users
// @tag v1

func main() {}
`
	fp := filepath.Join(dir, "ann.go")
	if err := os.WriteFile(fp, []byte(code), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	eps, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir err: %v", err)
	}

	var post, get *Endpoint
	for i := range eps {
		if eps[i].Method == "POST" {
			post = &eps[i]
		}
		if eps[i].Method == "GET" {
			get = &eps[i]
		}
	}
	if post == nil || get == nil {
		t.Fatalf("expected POST and GET endpoints")
	}
	if post.BodyRaw == "" || post.Headers["Authorization"] == "" {
		t.Errorf("expected headers/body on POST")
	}
	if len(get.Tags) == 0 {
		t.Errorf("expected tags on GET")
	}
}

func TestScanDir_MixedRouters_Heuristic(t *testing.T) {
	dir := t.TempDir()
	code := `
package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request){}

func main() {
	http.HandleFunc("/v1/ping", handler)
	x := something()
	x.HandleFunc("/v1/users", handler).Methods("GET","POST")
	var c Router
	c.Delete("/v1/orders/{id}", handler)
}
`
	fp := filepath.Join(dir, "main.go")
	if err := os.WriteFile(fp, []byte(code), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	eps, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("ScanDir err: %v", err)
	}
	want := map[string]bool{
		"ANY /v1/ping":           true,
		"GET /v1/users":          true,
		"POST /v1/users":         true,
		"DELETE /v1/orders/{id}": true,
	}
	got := make(map[string]bool)
	for _, e := range eps {
		got[strings.ToUpper(e.Method)+" "+e.Path] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing %s", k)
		}
	}
}
