// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package input captures invocation-local file bytes for parsing and identity.
package input

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type File struct {
	data []byte
}

func (f *File) Bytes() []byte { return bytes.Clone(f.data) }
func (f *File) Hash() string  { return HashBytes(f.data) }

func HashBytes(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }

type captured struct {
	file *File
	err  error
}

// Snapshot reads each absolute, cleaned path once, including failed reads.
// Returned bytes cannot mutate the capture. Separate files are not captured
// atomically together; every parser and digest uses its own captured bytes.
type Snapshot struct {
	readFile func(string) ([]byte, error)
	files    map[string]captured
}

func NewSnapshot(readFile func(string) ([]byte, error)) *Snapshot {
	if readFile == nil {
		readFile = os.ReadFile
	}
	return &Snapshot{readFile: readFile, files: map[string]captured{}}
}

func (s *Snapshot) Read(path string) (*File, error) {
	key, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if previous, ok := s.files[key]; ok {
		return previous.file, previous.err
	}
	data, err := s.readFile(key)
	var file *File
	if err == nil {
		file = &File{data: bytes.Clone(data)}
	}
	s.files[key] = captured{file: file, err: err}
	return file, err
}
