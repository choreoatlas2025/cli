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
