package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryOnlyMatchesFullCatalogue(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":                 "module example.com/root\n\ngo 1.26.1\n",
		"main.go":                "package main\n//go:generate echo generated\nfunc main() {}\n",
		"main_test.go":           "package main\nimport \"testing\"\nfunc TestOne(t *testing.T) {}\nfunc FuzzTwo(f *testing.F) {}\nfunc ExampleThree() {}\n",
		"other/other.go":         "package other\nfunc Value() int { return 1 }\n",
		"broken/broken.go":       "package broken\nfunc (\n",
		"nested/go.mod":          "module example.com/nested\n\ngo 1.26.1\n",
		"nested/nested_test.go":  "package nested\nimport \"testing\"\nfunc TestNested(t *testing.T) {}\n",
		"empty/go.mod":           "module example.com/empty\n\ngo 1.26.1\n",
		"invalid/go.mod":         "not a Go module\n",
		"invalid/invalid.go":     "package invalid\n",
		"templates/body.go/a.in": "an asset in a directory named like a Go file\n",
	})
	full, thin := t.TempDir(), t.TempDir()
	if err := runAll([]string{"--root", root, "--output-dir", full}); err != nil {
		t.Fatal(err)
	}
	if err := runAll([]string{"--discovery-only", "--root", root, "--output-dir", thin}); err != nil {
		t.Fatal(err)
	}
	checked := 0
	err := filepath.WalkDir(thin, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(thin, p)
		if err != nil {
			return err
		}
		if rel != "_modules_" && rel != testNamesFile && !strings.HasPrefix(rel, "_discovery_") {
			t.Errorf("source-policy output in discovery result: %s", rel)
		}
		got, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		want, err := os.ReadFile(filepath.Join(full, rel))
		if err != nil {
			return err
		}
		if !bytes.Equal(want, got) {
			t.Errorf("catalogue %s differs from full scan\nwant %s\ngot %s", rel, want, got)
		}
		checked++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 5 {
		t.Fatalf("checked only %d catalogue files", checked)
	}
	// Full source-policy operations still report invalid Go bodies and retain
	// their header pack. Discovery has the same metadata despite that error.
	if data, err := os.ReadFile(filepath.Join(full, moduleOutputFile(".", ".err"))); err != nil || len(data) == 0 {
		t.Fatalf("full scan no longer reports the syntax error: %s, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(full, headerPackFile)); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoveryOnlyRejectsSourcePolicyModes(t *testing.T) {
	for _, mode := range []string{"--test", "--generate"} {
		if err := runAll([]string{"--discovery-only", mode, "--root", t.TempDir(), "--output-dir", t.TempDir()}); err == nil {
			t.Fatalf("discovery silently accepted %s", mode)
		}
	}
}
