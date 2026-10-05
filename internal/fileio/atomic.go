// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

// Package fileio stages single-file outputs before atomically replacing them.
// Callers must serialize writers and prevent concurrent destination changes.
package fileio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CommitError means rename completed but directory durability could not be confirmed.
// The new destination is already visible; consumers must inspect it before retrying.
type CommitError struct {
	Path string
	Err  error
}

func (e *CommitError) Error() string {
	return fmt.Sprintf("output %s committed; directory sync failed, durability unconfirmed: %v", e.Path, e.Err)
}
func (e *CommitError) Unwrap() error { return e.Err }

func WriteFile(path string, data []byte, mode os.FileMode) error {
	return writeFile(path, data, mode, Stage, os.Rename, SyncDir)
}

func writeFile(path string, data []byte, mode os.FileMode, stage func(string, []byte, os.FileMode) (string, error), rename func(string, string) error, syncDir func(string) error) error {
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("invalid output target: %s is not a regular file", path)
		}
		mode = info.Mode().Perm()
	}
	name, err := stage(path, data, mode)
	if err != nil {
		return fmt.Errorf("failed to stage output %s: %w", path, err)
	}
	defer func() { _ = os.Remove(name) }()
	if err := rename(name, path); err != nil {
		return fmt.Errorf("failed to replace output %s; previous output retained: %w", path, err)
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return &CommitError{Path: path, Err: err}
	}
	return nil
}

// Stage writes and syncs a temporary file beside path. The caller owns cleanup.
func Stage(path string, data []byte, mode os.FileMode) (string, error) {
	return stageFile(path, data, mode, func(f *os.File, b []byte) error {
		n, err := f.Write(b)
		if err == nil && n != len(b) {
			err = io.ErrShortWrite
		}
		return err
	})
}
func stageFile(path string, data []byte, mode os.FileMode, write func(*os.File, []byte) error) (name string, err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".choreoatlas-stage-*")
	if err != nil {
		return "", err
	}
	name = f.Name()
	staged := name
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = os.Remove(staged)
		}
	}()
	if err = write(f, data); err != nil {
		return "", err
	}
	if err = f.Chmod(mode); err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return name, nil
}
