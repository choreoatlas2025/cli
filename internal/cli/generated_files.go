// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"github.com/choreoatlas2025/cli/internal/fileio"
	"os"
	"path/filepath"
)

type generatedFile struct {
	path string
	data []byte
}

type preparedFile struct {
	path, staged, backup string
}

// Stage every destination before replacing any file. Rename within each
// destination directory, and roll back completed replacements on commit error.
// This handles ordinary write errors; it is not a cross-directory crash journal.
func commitGeneratedFiles(files []generatedFile, rename func(string, string) error) error {
	var prepared []preparedFile
	retainBackups := false
	defer func() {
		for _, file := range prepared {
			if file.staged != "" {
				_ = os.Remove(file.staged)
			}
			if file.backup != "" && !retainBackups {
				_ = os.Remove(file.backup)
			}
		}
	}()
	seen := map[string]bool{}
	for _, file := range files {
		path, err := filepath.Abs(file.path)
		if err != nil {
			return err
		}
		if seen[path] {
			return fmt.Errorf("invalid generation targets: duplicate destination %s", path)
		}
		seen[path] = true
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		entry := preparedFile{path: path}
		mode := os.FileMode(0o644)
		info, err := os.Lstat(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("invalid generation target: %s is not a regular file", path)
			}
			mode = info.Mode().Perm()
			original, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry.backup, err = stageGeneratedFile(path, original, mode)
			if err != nil {
				return err
			}
		}
		prepared = append(prepared, entry)
		prepared[len(prepared)-1].staged, err = stageGeneratedFile(path, file.data, mode)
		if err != nil {
			return err
		}
	}
	for i, file := range prepared {
		if err := rename(file.staged, file.path); err != nil {
			var rollbackErr error
			for j := i - 1; j >= 0; j-- {
				previous := prepared[j]
				var restoreErr error
				if previous.backup != "" {
					restoreErr = rename(previous.backup, previous.path)
				} else {
					restoreErr = os.Remove(previous.path)
				}
				rollbackErr = errors.Join(rollbackErr, restoreErr)
			}
			if rollbackErr != nil {
				retainBackups = true
				return errors.Join(fmt.Errorf("failed to commit %s: %w", file.path, err), fmt.Errorf("rollback failed; recovery files retained beside destinations: %w", rollbackErr))
			}
			return fmt.Errorf("failed to commit %s; previous files restored: %w", file.path, err)
		}
	}
	return nil
}

func stageGeneratedFile(path string, data []byte, mode os.FileMode) (string, error) {
	return fileio.Stage(path, data, mode)
}
