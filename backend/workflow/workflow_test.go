package workflow

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestShippedCatalog(t *testing.T) {
	c, err := LoadFile(filepath.Join(repoRoot(t), "resources", "workflow", "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.GroupIDs(), []string{"verify", "release"}) {
		t.Fatal(c.GroupIDs())
	}
	if len(c.EmptyGroups()) != 0 {
		t.Fatal(c.EmptyGroups())
	}
	test, ok := c.ByID("test")
	if !ok || test.Group != "verify" || test.Argv[0] != "go" {
		t.Fatalf("%+v %v", test, ok)
	}
}

func TestFixtureClassification(t *testing.T) {
	root := filepath.Join(repoRoot(t), "testdata", "workflow")
	if _, err := LoadFile(filepath.Join(root, "ok.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(filepath.Join(root, "bad-group.json")); err == nil {
		t.Fatal("unknown group was accepted")
	}
}

func TestExecuteRejectsEmptyDir(t *testing.T) {
	res := Execute(context.Background(), Task{ID: "t", Argv: []string{"go", "version"}}, "")
	if res.OK || res.Err != "no module root" {
		t.Fatalf("%+v", res)
	}
}

func TestExecuteGoVersion(t *testing.T) {
	if _, err := os.Stat("/usr/bin/true"); err != nil && !commandExists("true") {
		t.Skip("no true")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := Execute(ctx, Task{ID: "t", Argv: []string{"go", "version"}}, t.TempDir())
	if !res.OK {
		t.Fatalf("%+v", res)
	}
	if res.Output == "" {
		t.Fatal("no output")
	}
}

func commandExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found")
	return ""
}
