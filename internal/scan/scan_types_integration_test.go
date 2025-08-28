//go:build integration

package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMod(dir string) error {
	mod := "module example.com/tmp\n\ngo 1.21\n\nrequire (\n\tgithub.com/go-chi/chi/v5 v5.0.10\n\tgithub.com/gorilla/mux v1.8.1\n\tgithub.com/labstack/echo/v4 v4.11.4\n\tgithub.com/gofiber/fiber/v2 v2.52.4\n)\n"
	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644)
}

func TestScanDirWithTypes_MixedRoutersAndPrefixes(t *testing.T) {
	dir := t.TempDir()
	if err := writeMod(dir); err != nil {
		t.Fatal(err)
	}

	code := `
package main

import (
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
	"github.com/gofiber/fiber/v2"
)

func handler(w http.ResponseWriter, r *http.Request){}

func main() {
	// net/http
	muxx := http.NewServeMux()
	muxx.HandleFunc("/v1/ping", handler)

	// gorilla mux
	r := mux.NewRouter()
	sub := r.PathPrefix("/api").Subrouter()
	sub.HandleFunc("/users", handler).Methods("GET","POST")
	sub.Handle("/orders", http.HandlerFunc(handler)).Methods("GET")

	// chi com Route
	c := chi.NewRouter()
	c.Route("/v1", func(sr chi.Router) {
		sr.Get("/books", handler)
		sr.Post("/books", handler)
	})

	// Echo com Group
	e := echo.New()
	g := e.Group("/v2")
	g.GET("/status", func(c echo.Context) error { return nil })

	// Fiber com Group
	app := fiber.New()
	grp := app.Group("/v3")
	grp.Post("/login", func(c *fiber.Ctx) error { return nil })
}
`
	fp := filepath.Join(dir, "main.go")
	if err := os.WriteFile(fp, []byte(code), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	eps, err := ScanDirWithOpts(ScanOptions{Dir: dir, UseTypes: true})
	if err != nil {
		t.Fatalf("ScanDirWithOpts err: %v", err)
	}

	want := map[string]bool{
		"ANY /v1/ping":   true,
		"GET /api/users": true, "POST /api/users": true, "GET /api/orders": true,
		"GET /v1/books": true, "POST /v1/books": true,
		"GET /v2/status": true,
		"POST /v3/login": true,
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
