// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package fileio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAtomicOutputFailures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	fault := errors.New("injected disk full")
	failingStage := func(path string, b []byte, mode os.FileMode) (string, error) {
		return stageFile(path, b, mode, func(f *os.File, b []byte) error {
			if _, err := f.Write(b[:1]); err != nil {
				return err
			}
			return fault
		})
	}
	for _, stage := range []func(string, []byte, os.FileMode) (string, error){failingStage, Stage} {
		err := writeFile(path, []byte("new"), 0644, stage, func(string, string) error { return fault }, SyncDir)
		if !errors.Is(err, fault) {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "old" {
			t.Fatal("previous output truncated", err)
		}
		matches, err := filepath.Glob(filepath.Join(dir, ".choreoatlas-stage-*"))
		if err != nil || len(matches) != 0 {
			t.Fatalf("stage leaked: %v %v", matches, err)
		}
	}
	if err := WriteFile(path, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("private permissions lost", err)
	}
	err = writeFile(path, []byte("committed"), 0644, Stage, os.Rename, func(string) error { return fault })
	var commitErr *CommitError
	if !errors.As(err, &commitErr) {
		t.Fatal("post-commit error not distinguished", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "committed" {
		t.Fatal("commit state misstated", err)
	}
	symlink := filepath.Join(dir, "link")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(symlink, []byte("attack"), 0644); err == nil {
		t.Fatal("symlink followed")
	}
	if err := WriteFile(dir, []byte("attack"), 0644); err == nil {
		t.Fatal("directory replaced")
	}
}

func TestAtomicOutputProcessHelper(t *testing.T) {
	if os.Getenv("CHOREOATLAS_STAGE_PROCESS") != "1" {
		return
	}
	name, err := Stage(os.Getenv("CHOREOATLAS_STAGE_TARGET"), []byte("new"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(name)
	// Parent kills this process after the synced stage has been created.
	_, _ = io.Copy(io.Discard, os.Stdin)
}
func TestAtomicOutputInterruptedBeforeRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAtomicOutputProcessHelper$")
	cmd.Env = append(os.Environ(), "CHOREOATLAS_STAGE_PROCESS=1", "CHOREOATLAS_STAGE_TARGET="+path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close() }()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line == "" {
			t.Fatal("child stage failed")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stage timeout")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "old" {
		t.Fatal("interrupted stage damaged old output", err)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".choreoatlas-stage-*"))
	if len(matches) != 1 {
		t.Fatal("interrupted stage state unexplained", matches)
	}
}
