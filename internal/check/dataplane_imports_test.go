package check

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var forbidden = []string{
	"internal/platform",
	"internal/reactor",
	"internal/buffer",
	"internal/protocol",
	"internal/proxy",
	"internal/routing",
	"internal/upstream",
	"internal/metrics",
}

func TestDataplaneNoNetImport(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	fset := token.NewFileSet()
	for _, rel := range forbidden {
		dir := filepath.Join(root, rel)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			af, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, im := range af.Imports {
				p := strings.Trim(im.Path.Value, `"`)
				if p == "net" || strings.HasPrefix(p, "net/") {
					t.Errorf("%s imports %s", path, p)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
