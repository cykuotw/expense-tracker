package engine

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
	"expense-tracker/agent/skills"

	"google.golang.org/adk/v2/tool"
)

func bundledFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(skills.Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := fs.ReadFile(skills.Files, path)
			if err != nil {
				return err
			}
			files[path] = &fstest.MapFile{Data: data}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func testCatalog(t *testing.T, client getbalance.Client) map[string]tool.Tool {
	t.Helper()
	catalog, err := tools.NewCatalog(client)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
