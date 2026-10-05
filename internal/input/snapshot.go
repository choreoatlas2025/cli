// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package input captures invocation-local file bytes for parsing and identity.
package input

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

type File struct {
	data []byte
	hash string
}

func (f *File) Bytes() []byte { return bytes.Clone(f.data) }
func (f *File) Hash() string  { return f.hash }

func HashBytes(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }

type captured struct {
	file *File
	err  error
}

// Snapshot reads each absolute, cleaned path once, including failed reads.
// Returned bytes cannot mutate the capture. Separate files are not captured
// atomically together; every parser and digest uses its own captured bytes.
type Snapshot struct {
	readFile         func(string) ([]byte, error)
	files            map[string]captured
	maxBytes         int64
	totalBytes       int64
	executableHashes map[string]capturedHash
}

func NewSnapshot(readFile func(string) ([]byte, error)) *Snapshot {
	return NewSnapshotWithLimit(readFile, DefaultMaxBytes)
}

func NewSnapshotWithLimit(readFile func(string) ([]byte, error), maxBytes int64) *Snapshot {
	if maxBytes <= 0 || maxBytes > math.MaxInt64/2 {
		maxBytes = DefaultMaxBytes
	}
	return &Snapshot{readFile: readFile, files: map[string]captured{}, maxBytes: maxBytes}
}

func (s *Snapshot) Read(path string) (*File, error) {
	return s.read(path)
}

type capturedHash struct {
	hash string
	err  error
}

// HashExecutable streams tool bytes, retaining the identity/error once per path.
// It does not consume the customer byte/file budget or retain executable bytes.
func (s *Snapshot) HashExecutable(path string) (string, error) {
	key, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if previous, ok := s.executableHashes[key]; ok {
		return previous.hash, previous.err
	}
	var hash string
	if s.readFile != nil {
		var data []byte
		data, err = s.readFile(key)
		if err == nil {
			hash = HashBytes(data)
		}
	} else {
		var f *os.File
		f, err = os.Open(key)
		if err == nil {
			h := sha256.New()
			_, err = io.Copy(h, f)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				hash = fmt.Sprintf("sha256:%x", h.Sum(nil))
			}
		}
	}
	if s.executableHashes == nil {
		s.executableHashes = map[string]capturedHash{}
	}
	s.executableHashes[key] = capturedHash{hash: hash, err: err}
	return hash, err
}

func (s *Snapshot) read(path string) (*File, error) {
	key, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if previous, ok := s.files[key]; ok {
		return previous.file, previous.err
	}
	if len(s.files) >= MaxInputFiles {
		return nil, fmt.Errorf("invalid input: too many input files (maximum %d)", MaxInputFiles)
	}
	var data []byte
	if s.readFile == nil {
		data, err = ReadFileLimited(key, s.maxBytes)
	} else {
		data, err = s.readFile(key)
		if err == nil {
			data = bytes.Clone(data)
		}
	}
	if err == nil && (int64(len(data)) > s.maxBytes || int64(len(data)) > 2*s.maxBytes-s.totalBytes) {
		err = fmt.Errorf("invalid input: captured bytes exceed file or invocation limit")
	}
	var file *File
	if err == nil {
		file = &File{data: data, hash: HashBytes(data)}
		s.totalBytes += int64(len(data))
	}
	s.files[key] = captured{file: file, err: err}
	return file, err
}
