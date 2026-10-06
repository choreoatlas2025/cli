// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/choreoatlas2025/cli/templates"
)

func initTestAssets(t *testing.T) fstest.MapFS {
	t.Helper()
	assets := fstest.MapFS{}
	if err := fs.WalkDir(templates.InitFS, "init", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(templates.InitFS, path)
		if err == nil {
			assets[path] = &fstest.MapFile{Data: data}
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return assets
}

func seedInitTargets(t *testing.T, files []generatedFile) map[string][]byte {
	t.Helper()
	originals := map[string][]byte{}
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.path), 0o755); err != nil {
			t.Fatal(err)
		}
		data := []byte("original " + file.path)
		if err := os.WriteFile(file.path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		originals[file.path] = data
	}
	return originals
}

func assertInitTargets(t *testing.T, originals map[string][]byte) {
	t.Helper()
	for path, before := range originals {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Errorf("original changed %s: %v", path, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("original mode changed %s", path)
		}
	}
}

func TestArchitectureInitQualificationBeforeWrite(t *testing.T) {
	for _, kind := range []string{"invalid-flow", "missing-reference", "invalid-service", "invalid-trace", "missing-trace-time", "invalid-workflow", "missing-workflow", "empty-title"} {
		t.Run(kind, func(t *testing.T) {
			opts := initOptions{TargetDir: t.TempDir(), Mode: "template", Title: "valid", CI: "minimal", Force: true}
			files, _, err := renderInitFiles(opts, templates.InitFS)
			if err != nil {
				t.Fatal(err)
			}
			originals := seedInitTargets(t, files)
			assets := initTestAssets(t)
			switch kind {
			case "invalid-flow":
				assets[templates.RootFlowSpecTemplate].Data = []byte("[broken YAML")
			case "missing-reference":
				assets[templates.RootFlowSpecTemplate].Data = bytes.ReplaceAll(assets[templates.RootFlowSpecTemplate].Data, []byte("order-service.servicespec.yaml"), []byte("absent.servicespec.yaml"))
				oldReference := filepath.Join(opts.TargetDir, "services", "absent.servicespec.yaml")
				if err := os.WriteFile(oldReference, assets[templates.OrderServiceTemplate].Data, 0o600); err != nil {
					t.Fatal(err)
				}
				originals[oldReference] = bytes.Clone(assets[templates.OrderServiceTemplate].Data)
			case "invalid-service":
				assets[templates.OrderServiceTemplate].Data = []byte("service: ''\noperations: []\n")
			case "invalid-trace":
				assets[templates.SuccessfulTraceTemplate].Data = []byte("invalid JSON")
			case "missing-trace-time":
				assets[templates.SuccessfulTraceTemplate].Data = []byte(`{"spans":[{"service":"orderService","name":"createOrder","endNanos":1}]}`)
			case "invalid-workflow":
				assets[templates.GithubWorkflowMinimalTemplate].Data = []byte("[broken YAML")
			case "missing-workflow":
				delete(assets, templates.GithubWorkflowMinimalTemplate)
			case "empty-title":
				opts.Title = ""
			}
			writes := 0
			_, _, err = initializeProject(opts, assets, func(a, b string) error { writes++; return os.Rename(a, b) })
			if err == nil || writes != 0 {
				t.Errorf("invalid plan started committing: %v, %d writes", err, writes)
			}
			assertInitTargets(t, originals)
		})
	}
}

func TestArchitectureInitRollbackCompleteSet(t *testing.T) {
	for _, mode := range []string{"template", "trace"} {
		for _, existing := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/new", true: "/replace"}[existing], func(t *testing.T) {
				dir := architectureFixture(t, "parallel-explicit-zero")
				opts := initOptions{TargetDir: filepath.Join(dir, "project"), Mode: mode, TracePath: filepath.Join(dir, "trace.json"), Title: "valid", CI: "combo", Force: existing, Examples: true}
				files, _, err := renderInitFiles(opts, templates.InitFS)
				if err != nil {
					t.Fatal(err)
				}
				var originals map[string][]byte
				if existing {
					originals = seedInitTargets(t, files)
				}
				writes := 0
				_, _, err = initializeProject(opts, templates.InitFS, func(a, b string) error {
					writes++
					if writes == len(files) {
						return errors.New("injected final workflow commit failure")
					}
					return os.Rename(a, b)
				})
				if err == nil {
					t.Fatal("fault did not reach commit")
				}
				if existing {
					assertInitTargets(t, originals)
				} else {
					for _, file := range files {
						if _, err := os.Lstat(file.path); !os.IsNotExist(err) {
							t.Errorf("partial output survived: %s", file.path)
						}
					}
				}
				if err := filepath.WalkDir(opts.TargetDir, func(path string, entry fs.DirEntry, err error) error {
					if err == nil && !entry.IsDir() && len(entry.Name()) >= 18 && entry.Name()[:18] == ".choreoatlas-stage-" {
						t.Errorf("staging file leaked: %s", path)
					}
					return err
				}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestArchitectureInitTraceCaptureContent(t *testing.T) {
	dir := architectureFixture(t, "parallel-explicit-zero")
	data, err := os.ReadFile(filepath.Join(dir, "trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	// --trace accepts native JSON by content, regardless of the input basename.
	source := filepath.Join(dir, "capture.servicespec.yaml")
	if err := os.WriteFile(source, data, 0o644); err != nil {
		t.Fatal(err)
	}
	opts := initOptions{TargetDir: filepath.Join(dir, "project"), Mode: "trace", TracePath: source, Title: "captured", CI: "none"}
	_, relative, err := initializeProject(opts, templates.InitFS, os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(opts.TargetDir, relative))
	if err != nil || !bytes.Equal(data, copied) {
		t.Fatalf("copied trace differs from captured input: %v", err)
	}
}
