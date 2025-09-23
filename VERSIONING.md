# ChoreoAtlas CLI Versioning Strategy

## Version Format

ChoreoAtlas CE uses a single semantic versioning channel with an explicit edition suffix:

```
v{MAJOR}.{MINOR}.{PATCH}-ce[.rc.N]
```

- `MAJOR`, `MINOR`, `PATCH`: follow SemVer semantics.
- `ce`: denotes the Community Edition build (zero telemetry, offline).
- Optional `.rc.N`: release candidates prior to a stable cut. We currently avoid other prerelease labels; betas are issued as RCs when needed.

Examples:
- `v0.8.0-ce` – stable CE release
- `v0.8.1-ce.rc.1` – first release candidate for `v0.8.1-ce`

## Tagging & Distribution

A single Git tag (e.g. `v0.8.0-ce`) fans out to every distribution channel:

| Channel | Artifact | Notes |
|---------|----------|-------|
| GitHub Releases | `choreoatlas_v0.8.0-ce_<os>_<arch>.{tar.gz,zip}` + `SHA256SUMS.txt` | Uploads driven by GoReleaser |
| Homebrew Tap | `choreoatlas2025/homebrew-choreoatlas/choreoatlas` | Formula updates commit the same version number |
| Install scripts | `scripts/install.sh`, `scripts/install.ps1` | Default to `latest`; `--version/-Version` pins to any CE tag |
| Containers | `choreoatlas/cli` & `ghcr.io/choreoatlas2025/cli` | Multi-arch manifests tagged `v0.8.0-ce` and `latest` |

## Branch Strategy

- `main`: rolling development for CE.
- `release/v{MAJOR}.{MINOR}.x`: optional stabilization branches when coordinating large drops.
- Tags: `v*.*.*-ce[.rc.N]` created from `main` or a release branch.

## Build Metadata Injection

`make build` and GoReleaser inject version metadata at link time:

```bash
LDFLAGS="-X github.com/choreoatlas2025/cli/internal/cli.Version=v0.8.0-ce \
        -X github.com/choreoatlas2025/cli/internal/cli.GitCommit=$(git rev-parse --short HEAD) \
        -X github.com/choreoatlas2025/cli/internal/cli.BuildTime=$(date -u +%FT%TZ) \
        -X github.com/choreoatlas2025/cli/internal/cli.BuildEdition=ce"
```

The `choreoatlas version` command always shows the `-ce` suffix so operators can confirm edition provenance.

## Release Checklist

1. Ensure `main` (or the release branch) is green in CI.
2. Update documentation, examples, and changelog entries.
3. Tag the release: `git tag vX.Y.Z-ce && git push origin vX.Y.Z-ce`.
4. GitHub Actions (`release.yml`) triggers GoReleaser, which:
   - Produces multi-arch archives and `SHA256SUMS.txt`
   - Publishes Docker Hub and GHCR images (`vX.Y.Z-ce`, `latest`)
   - Updates the Homebrew tap (`homebrew-choreoatlas`)
5. Validate artifacts:
   - `brew install choreoatlas2025/homebrew-choreoatlas/choreoatlas`
   - `./scripts/install.sh --version vX.Y.Z-ce`
   - `docker run --rm choreoatlas/cli:vX.Y.Z-ce version`
6. Announce the release (GitHub Release notes, docs updates).

## Future Editions

The `-ce` suffix keeps the namespace open for future commercial or enterprise editions (e.g. `-pro`, `-cloud`). Any new edition must use a distinct suffix and distribution channel so CE users can continue to verify the zero-telemetry guarantee.
