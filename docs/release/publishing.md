# Release Infrastructure Checklist

Before tagging a `vX.Y.Z-ce` release, ensure the supporting infrastructure is ready.

## 1. Homebrew Tap Repository

- Create a public repository `choreoatlas2025/homebrew-choreoatlas`.
- Default branch: `main`.
- The GoReleaser workflow will push formula updates directly; no manual PR is created.
- Grant updaters write access (or use a machine account token, see below).

## 2. GitHub Secrets

Set the following secrets on `choreoatlas2025/cli`:

| Secret | Purpose | Notes |
|--------|---------|-------|
| `DOCKERHUB_USERNAME` | Docker Hub login | Should have push rights to `choreoatlas/cli` |
| `DOCKERHUB_TOKEN` | Docker Hub access token/password | Use a long-lived access token |
| `GORELEASER_GITHUB_TOKEN` | Push access to `homebrew-choreoatlas` tap | Personal access token or GitHub App with `repo` scope |

The built-in `GITHUB_TOKEN` already has permission to publish GitHub Releases and GHCR images; no extra configuration is required for those steps.

## 3. Docker Registries

- Ensure the `choreoatlas/cli` repository exists on Docker Hub with the `write` permission granted to the releasing account.
- GHCR images are published under the same organization automatically (`ghcr.io/choreoatlas2025/cli`). No additional setup is required beyond the default packages permission.

## 4. Local Validation

Before cutting an official tag, run a dry-release to confirm configuration:

```bash
# Clean environment
rm -rf dist

# Snapshot build (no publishing)
goreleaser release --clean --skip=publish --snapshot
```

Verify the following artifacts in `dist/`:
- Multi-arch archives and `SHA256SUMS.txt`
- Homebrew formula under `dist/homebrew`
- Docker image tarballs or metadata for `linux/amd64` and `linux/arm64`

## 5. Permissions & Access

- Confirm at least one maintainer has access to `docker.io/choreoatlas/cli`, `ghcr.io/choreoatlas2025`, and the Homebrew tap repository.
- Store the release tokens in a shared secret manager so they can be rotated without repository downtime.

## 6. After Each Release

- `brew update` should list the new formula version.
- `docker run choreoatlas/cli:vX.Y.Z-ce version` prints the expected version information.
- `./scripts/install.sh --version vX.Y.Z-ce` succeeds on a clean machine.

Keeping this checklist handy ensures GoReleaser can produce a consistent release across every distribution channel.
