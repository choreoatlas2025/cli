// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestArchitectureAtomicReportAndBaseline(t *testing.T) {
	dir, _, _ := correctnessFixture(t, []string{"A"}, "true")
	for _, format := range []string{"json", "junit", "html"} {
		name := "report." + format
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("old report"), 0600); err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 0, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", format, "--report-out", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
			t.Fatal("private report permissions lost")
		}
		link := filepath.Join(dir, "link."+format)
		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		correctnessCommand(t, dir, 2, "validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", format, "--report-out", filepath.Base(link))
		data, err := os.ReadFile(path)
		if err != nil || string(data) != string(original) {
			t.Fatal("rejected report target damaged original", err)
		}
	}
	path := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(path, []byte("old baseline"), 0600); err != nil {
		t.Fatal(err)
	}
	correctnessCommand(t, dir, 0, "baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "baseline.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("private baseline permissions lost")
	}
	if runtime.GOOS == "windows" {
		return
	}
	blocked := filepath.Join(dir, "readonly")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(blocked, "result")
	if err := os.WriteFile(target, []byte("retain me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(blocked, 0700) }()
	for _, args := range [][]string{
		{"validate", "--flow", "flow.yaml", "--trace", "trace.json", "--report-format", "json", "--report-out", "readonly/result"},
		{"baseline", "record", "--flow", "flow.yaml", "--trace", "trace.json", "--out", "readonly/result"},
	} {
		correctnessCommand(t, dir, 1, args...)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "retain me" {
		t.Fatal("permission failure truncated existing result", err)
	}
}
