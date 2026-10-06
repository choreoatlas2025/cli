// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package fileio

// Windows does not provide directory fsync through os.File. The staged file is
// synced before rename; no crash-durable directory entry guarantee is claimed.
func SyncDir(string) error { return nil }
