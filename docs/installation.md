# Installation Guide

## Overview

ChoreoAtlas CLI Community Edition (CE) ships as a **single, zero-telemetry channel**. Every artifact advertises the same `-ce` suffix so you can verify exactly which build you are running.

- Git tags and GitHub Releases: `vX.Y.Z-ce`
- Homebrew formula: `choreoatlas2025/homebrew-choreoatlas/choreoatlas`
- Installers: `scripts/install.sh` (macOS/Linux) and `scripts/install.ps1` (Windows)
- Containers: `choreoatlas/cli` and `ghcr.io/choreoatlas2025/cli` multi-arch manifests tagged `vX.Y.Z-ce` and `latest`
- Checksums: `SHA256SUMS.txt` accompanies every release

The CE build is permanently offline and telemetry-free. See [docs/privacy.md](privacy.md) for verification steps.

## Method 1: Homebrew (macOS & Linux)

```bash
brew install choreoatlas2025/homebrew-choreoatlas/choreoatlas
# Update later
brew upgrade choreoatlas2025/homebrew-choreoatlas/choreoatlas
```

What you get:
- `choreoatlas` binary in your Homebrew prefix
- `ca` helper symlink (skips creation if `ca` already exists and is not pointing to ChoreoAtlas)
- Automatic updates via `brew upgrade`

## Method 2: Shell Installer (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/choreoatlas2025/cli/main/scripts/install.sh -o choreoatlas-install.sh
chmod +x choreoatlas-install.sh
./choreoatlas-install.sh                # auto-picks /opt/homebrew/bin → /usr/local/bin → $HOME/.local/bin
```

Flags:
- `--version vX.Y.Z-ce` – pin to a specific release (default: latest)
- `--force` – overwrite an existing `ca` helper if it points elsewhere
- `--no-symlink` – skip creating the `ca` helper entirely

The script downloads the matching archive, verifies it with `SHA256SUMS.txt`, and installs to the first writable directory in the priority list `/opt/homebrew/bin` (Apple Silicon), `/usr/local/bin`, then `$HOME/.local/bin`.

## Method 3: PowerShell Installer (Windows)

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/choreoatlas2025/cli/main/scripts/install.ps1 -OutFile choreoatlas-install.ps1
pwsh -ExecutionPolicy Bypass -File choreoatlas-install.ps1
```

- Default install paths: `%ProgramFiles%\ChoreoAtlas` (if writable) falling back to `%LOCALAPPDATA%\ChoreoAtlas\bin`
- Helper creation: attempts `ca.exe` symbolic link, falls back to `ca.cmd` if symlinks are not permitted
- Flags: `-Version vX.Y.Z-ce`, `-Force`, `-NoSymlink`

## Method 4: Containers (Docker Hub & GHCR)

```bash
# Docker Hub
docker run --rm choreoatlas/cli:latest version

# GitHub Container Registry
docker run --rm ghcr.io/choreoatlas2025/cli:latest version
```

Both registries publish multi-arch (`linux/amd64`, `linux/arm64`) images with matching `vX.Y.Z-ce` manifests. Mount local files to `/workspace` for linting/validation workflows.

## Method 5: Manual Download

1. Open the [GitHub Releases page](https://github.com/choreoatlas2025/cli/releases).
2. Download the archive matching your OS/architecture (`choreoatlas_vX.Y.Z-ce_<os>_<arch>.tar.gz` or `.zip`).
3. Download `SHA256SUMS.txt` from the same release.
4. Verify integrity:
   ```bash
   grep "choreoatlas_vX.Y.Z-ce_darwin_arm64.tar.gz" SHA256SUMS.txt > checksum.txt
   shasum -a 256 -c checksum.txt
   ```
5. Extract the archive and move `choreoatlas` to a directory on your `PATH` (`/opt/homebrew/bin`, `/usr/local/bin`, or `%LOCALAPPDATA%\ChoreoAtlas\bin`).
6. Optionally create a helper symlink or wrapper (`ln -s choreoatlas /usr/local/bin/ca`).

## Verify Installation

```bash
choreoatlas version

# Expected fields
# choreoatlas vX.Y.Z-ce
# Edition: Community Edition (CE)
# Git Commit: <hash>
# Build Time: <timestamp>
# Go Version: go1.24.x
# Platform: darwin/arm64
```

For Windows PowerShell:
```powershell
choreoatlas version
```

## Helper Command (`ca`)

- Homebrew and the installers create a `ca` helper that mirrors `choreoatlas` when it does not overwrite an existing binary.
- Use `--no-symlink`/`-NoSymlink` to skip creation, or `--force`/`-Force` to replace an existing helper.
- Manual installs can add the helper later with `ln -s choreoatlas /usr/local/bin/ca` (macOS/Linux) or by creating a `ca.cmd` wrapper on Windows.

## Privacy & Zero Telemetry

ChoreoAtlas CE never makes outbound network calls, collects usage data, or embeds telemetry SDKs. For verification steps (binary inspection, network sniffing, source audit), see [docs/privacy.md](privacy.md).

## Troubleshooting

- **Permission denied / cannot write to install directory**: rerun with `sudo ./choreoatlas-install.sh` or let the script fall back to `$HOME/.local/bin`.
- **Existing `ca` command**: rerun with `--force` or `--no-symlink` to control helper creation.
- **Checksum mismatch**: redownload both the archive and `SHA256SUMS.txt`; ensure proxies do not rewrite downloads.
- **Command not found after install**: confirm the target directory is on your `PATH`. For `$HOME/.local/bin`, add `export PATH="$HOME/.local/bin:$PATH"` to your shell profile.
- **PowerShell execution policy**: use `-ExecutionPolicy Bypass` or run from an elevated PowerShell session.
- **Docker volume issues**: mount absolute paths, e.g. `-v "$(pwd)":/workspace` on macOS/Linux.

## Support

- **Documentation**: https://github.com/choreoatlas2025/cli/docs
- **Issues**: https://github.com/choreoatlas2025/cli/issues
- **Discussions**: https://github.com/choreoatlas2025/cli/discussions

## License

Apache 2.0 – see [LICENSE](https://github.com/choreoatlas2025/cli/blob/main/LICENSE) for details.
