// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package input

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotReadsOnceAndOwnsBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input.json")
	reads := 0
	data := []byte("original")
	snapshot := NewSnapshot(func(string) ([]byte, error) { reads++; return data, nil })
	first, err := snapshot.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'X'
	copy := first.Bytes()
	copy[0] = 'Y'
	second, err := snapshot.Read(filepath.Join(filepath.Dir(path), ".", filepath.Base(path)))
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || string(second.Bytes()) != "original" || second.Hash() != HashBytes([]byte("original")) {
		t.Fatalf("capture changed or reread: reads=%d bytes=%s", reads, second.Bytes())
	}
}

func TestSnapshotRetainsReadError(t *testing.T) {
	reads := 0
	snapshot := NewSnapshot(func(string) ([]byte, error) { reads++; return nil, os.ErrNotExist })
	path := filepath.Join(t.TempDir(), "missing")
	for i := 0; i < 2; i++ {
		if _, err := snapshot.Read(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	if reads != 1 {
		t.Fatalf("failed input retried %d times", reads)
	}
}

func TestSnapshotInputBudgets(t *testing.T) {
	dir := t.TempDir()
	s := NewSnapshotWithLimit(func(string) ([]byte, error) { return []byte("123456789"), nil }, 10)
	for _, name := range []string{"one", "two"} {
		if _, err := s.Read(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Read(filepath.Join(dir, "three")); err == nil {
		t.Fatal("aggregate cap ignored")
	}
	if _, err := s.HashExecutable(filepath.Join(dir, "tool")); err != nil {
		t.Fatal("tool charged to customer input budget", err)
	}
	s = NewSnapshotWithLimit(func(string) ([]byte, error) { return []byte("12345678901"), nil }, 10)
	if _, err := s.Read(filepath.Join(dir, "big")); err == nil {
		t.Fatal("file cap ignored")
	}
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("12345678901"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileLimited(path, 10); err == nil {
		t.Fatal("native reader ignored cap")
	}
}

func TestExecutableHashStreamAndCapture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte("original executable"), 0600); err != nil {
		t.Fatal(err)
	}
	s := NewSnapshotWithLimit(nil, 1)
	hash, err := s.HashExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	if hash != HashBytes([]byte("original executable")) {
		t.Fatal("stream digest differs")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	again, err := s.HashExecutable(path)
	if err != nil || again != hash {
		t.Fatal("tool identity reread", err)
	}
	if len(s.files) != 0 || s.totalBytes != 0 {
		t.Fatal("executable consumed customer input budget")
	}
}
