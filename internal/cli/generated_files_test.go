// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedFilesRollBackPartialCommit(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "service.yaml"), filepath.Join(dir, "flow.yaml")
	if err := os.WriteFile(first, []byte("original service"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("original flow"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := commitGeneratedFiles([]generatedFile{{path: first, data: []byte("new service")}, {path: second, data: []byte("new flow")}}, func(src, dst string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated commit I/O failure")
		}
		return os.Rename(src, dst)
	})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	for path, content := range map[string]string{first: "original service", second: "original flow"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != content {
			t.Fatalf("rollback changed %s: %s", path, b)
		}
	}
	info, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal("rollback changed original mode")
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, ".choreoatlas-stage-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left: %v", leftovers)
	}
}

func TestGeneratedFilesRollBackNewFiles(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "service.yaml"), filepath.Join(dir, "flow.yaml")
	count := 0
	err := commitGeneratedFiles([]generatedFile{{path: first, data: []byte("service")}, {path: second, data: []byte("flow")}}, func(src, dst string) error {
		count++
		if count == 2 {
			return errors.New("commit failed")
		}
		return os.Rename(src, dst)
	})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	for _, path := range []string{first, second} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed transaction left %s", path)
		}
	}
}
