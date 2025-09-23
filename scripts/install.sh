#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

REPO="choreoatlas2025/cli"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_DOWNLOAD="https://github.com/${REPO}/releases/download"
REQUESTED_VERSION=""
FORCE_SYMLINK=0
CREATE_SYMLINK=1
TMPDIR=""

usage() {
  cat <<'USAGE'
ChoreoAtlas CLI Installer (Community Edition)

Usage: scripts/install.sh [--version vX.Y.Z-ce] [--force] [--no-symlink]

Options:
  --version <tag>   Install the specified version (default: latest release)
  --force           Overwrite existing ca symlink if present
  --no-symlink      Do not create the ca convenience symlink
  -h, --help        Show this help message

Example:
  scripts/install.sh --version v0.8.0-ce
USAGE
}

cleanup() {
  if [[ -n "$TMPDIR" && -d "$TMPDIR" ]]; then
    rm -rf "$TMPDIR"
  fi
}

fail() {
  echo "[ERROR] $1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "Missing required command: $1"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version)
        shift
        [[ $# -gt 0 ]] || fail "--version requires a value"
        REQUESTED_VERSION="$1"
        ;;
      --force)
        FORCE_SYMLINK=1
        ;;
      --no-symlink)
        CREATE_SYMLINK=0
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        fail "Unknown option: $1"
        ;;
    esac
    shift || true
  done
}

normalize_version() {
  local version="$1"
  if [[ -z "$version" ]]; then
    version="$(curl -fsSL -H 'Accept: application/vnd.github+json' "${GITHUB_API}/releases/latest" | awk -F'"' '/"tag_name"/ {print $4; exit}')"
    [[ -n "$version" ]] || fail "Unable to determine latest release tag"
  fi
  if [[ $version != v* ]]; then
    version="v${version}"
  fi
  echo "$version"
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$os" in
    linux)
      os="linux"
      ;;
    darwin)
      os="darwin"
      ;;
    *)
      fail "Unsupported OS: $os"
      ;;
  esac

  case "$arch" in
    x86_64|amd64)
      arch="amd64"
      ;;
    arm64|aarch64)
      arch="arm64"
      ;;
    *)
      fail "Unsupported architecture: $arch"
      ;;
  esac

  echo "$os" "$arch"
}

select_install_dir() {
  local os="$1"
  local arch="$2"
  local candidates=()

  if [[ "$os" == "darwin" && "$arch" == "arm64" ]]; then
    candidates+=("/opt/homebrew/bin")
  fi
  candidates+=("/usr/local/bin")
  candidates+=("${HOME}/.local/bin")

  for dir in "${candidates[@]}"; do
    if [[ "$dir" == "${HOME}/.local/bin" && ! -d "$dir" ]]; then
      mkdir -p "$dir"
    fi
    if [[ -d "$dir" && -w "$dir" ]]; then
      echo "$dir"
      return 0
    fi
  done

  fail "No writable install directory found. Try re-running with sudo or ensure ${HOME}/.local/bin exists and is writable."
}

verify_checksum() {
  local archive_path="$1"
  local sums_file="$2"
  local archive_name
  archive_name="$(basename "$archive_path")"

  local line
  line="$(grep "  ${archive_name}$" "$sums_file" || true)"
  [[ -n "$line" ]] || fail "Checksum for ${archive_name} not found in $(basename "$sums_file")"

  if command -v shasum >/dev/null 2>&1; then
    echo "$line" >"${TMPDIR}/checksum.txt"
    (cd "$TMPDIR" && shasum -a 256 -c checksum.txt >/dev/null)
  elif command -v sha256sum >/dev/null 2>&1; then
    echo "$line" | sha256sum --check --status
  else
    fail "Neither shasum nor sha256sum is available for checksum verification"
  fi
}

create_symlink() {
  local install_dir="$1"
  local target="${install_dir}/choreoatlas"
  local link_path="${install_dir}/ca"

  [[ $CREATE_SYMLINK -eq 1 ]] || return 0

  if [[ -e "$link_path" || -L "$link_path" ]]; then
    if [[ $FORCE_SYMLINK -eq 1 ]]; then
      rm -f "$link_path"
    else
      if [[ -L "$link_path" ]]; then
        local current
        current="$(readlink "$link_path")"
        if [[ "$current" == "choreoatlas" || "$current" == "$target" ]]; then
          rm -f "$link_path"
        else
          echo "[INFO] Skipping ca symlink: ${link_path} already points to ${current}. Use --force to overwrite or --no-symlink to skip." >&2
          return 0
        fi
      else
        echo "[INFO] Skipping ca symlink: ${link_path} already exists. Use --force to overwrite or --no-symlink to skip." >&2
        return 0
      fi
    fi
  fi

  ln -s "choreoatlas" "$link_path"
}

main() {
  trap cleanup EXIT
  parse_args "$@"

  need_cmd curl
  need_cmd tar
  need_cmd uname

  local version
  version="$(normalize_version "$REQUESTED_VERSION")"
  local os arch
  read -r os arch < <(detect_platform)

  TMPDIR="$(mktemp -d)"
  local archive="choreoatlas_${version}_${os}_${arch}.tar.gz"
  local download_url="${GITHUB_DOWNLOAD}/${version}/${archive}"
  local sums_url="${GITHUB_DOWNLOAD}/${version}/SHA256SUMS.txt"

  echo "[INFO] Installing ChoreoAtlas ${version} for ${os}/${arch}"

  echo "[INFO] Downloading ${archive}"
  curl -fsSL "${download_url}" -o "${TMPDIR}/${archive}"

  echo "[INFO] Downloading checksums"
  curl -fsSL "${sums_url}" -o "${TMPDIR}/SHA256SUMS.txt"

  echo "[INFO] Verifying checksum"
  verify_checksum "${TMPDIR}/${archive}" "${TMPDIR}/SHA256SUMS.txt"

  echo "[INFO] Extracting archive"
  tar -xf "${TMPDIR}/${archive}" -C "${TMPDIR}"

  local extracted_bin="${TMPDIR}/choreoatlas"
  [[ -f "$extracted_bin" ]] || fail "Extracted binary not found"
  chmod +x "$extracted_bin"

  local install_dir
  install_dir="$(select_install_dir "$os" "$arch")"
  echo "[INFO] Installing to ${install_dir}"
  install -m 0755 "$extracted_bin" "${install_dir}/choreoatlas"

  create_symlink "$install_dir"

  echo "[INFO] Installation complete"
  echo "       Binary : ${install_dir}/choreoatlas"
  if [[ $CREATE_SYMLINK -eq 1 && -L "${install_dir}/ca" ]]; then
    echo "       Symlink: ${install_dir}/ca"
  else
    echo "       Symlink: skipped"
  fi
  echo "[INFO] Run 'choreoatlas version' to verify installation"
}

main "$@"
