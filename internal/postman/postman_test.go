package postman

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/williamkoller/postman-gen/internal/scan"
)

func TestBuildCollection_Golden(t *testing.T) {
	eps := []scan.Endpoint{
		{Method: "GET", Path: "/v1/users", SourceFile: "a.go"},
		{Method: "POST", Path: "/v1/users", SourceFile: "a.go", Headers: map[string]string{"X-Req": "1"}, BodyRaw: `{"a":1}`},
		{Method: "GET", Path: "/v1/orders/{id}", SourceFile: "b.go", Desc: "Get order"},
	}
	col := BuildCollection(BuildOpts{
		Name:          "Teste API",
		BaseURL:       "http://localhost:8080",
		GroupDepth:    2,
		GroupByMethod: true,
		TagFolders:    true,
	}, eps)

	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if info, ok := m["info"].(map[string]any); ok {
		info["_postman_id"] = "00000000-0000-0000-0000-000000000000"
	}
	norm, _ := json.MarshalIndent(m, "", "  ")

	golden := filepath.Join("testdata", "collection_v1.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, norm, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(norm)) {
		t.Errorf("collection differs.\n--- got:\n%s\n--- want:\n%s", string(norm), string(want))
	}
}
