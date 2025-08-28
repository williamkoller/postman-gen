//go:build go1.18

package scan

import (
	"go/parser"
	"go/token"
	"testing"
)

func FuzzAnnotations(f *testing.F) {
	seeds := []string{
		"// @route GET /x\n",
		"// @header X:A\n// @route POST /y\n",
		"// @body {\"a\":1}\n// @route PATCH /z\n",
		"// @tag a\n// @tag b\n// @route OPTIONS /k\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, comment string) {
		src := "package p\n\n" + "/*\n" + comment + "\n*/\n"
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fuzz.go", src, parser.ParseComments)
		if err != nil {
			return
		}
		_, _ = scanAnnotationsFromFile(file, "fuzz.go")
	})
}
