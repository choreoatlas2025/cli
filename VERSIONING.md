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

`make build`, GoReleaser and the standalone Dockerfile inject the complete
40-character Git commit and a build channel (`make`, `goreleaser`, `docker`).
Raw `go build` retains the `source` channel and unknown link metadata. The binary
hash remains the exact tool identity; the same commit can produce different
binaries across channels, environments or dirty development builds.

The standalone container requires `--build-arg GIT_COMMIT="$(git rev-parse HEAD)"`
and verifies that value against Git metadata included in its build context.
Supply `--build-arg VERSION=vX.Y.Z-ce` for a tagged build. `.dockerignore` excludes
local binaries, evidence and agent configuration from that context.

Link flags example:

```bash
LDFLAGS="-X github.com/choreoatlas2025/cli/internal/cli.Version=v0.8.0-ce \
        -X github.com/choreoatlas2025/cli/internal/cli.GitCommit=$(git rev-parse HEAD) \
        -X github.com/choreoatlas2025/cli/internal/cli.BuildTime=$(date -u +%FT%TZ) \
        -X github.com/choreoatlas2025/cli/internal/cli.BuildEdition=ce"
```

The `choreoatlas version` command displays one leading `v` and preserves an
existing `-ce` identifier, including release candidates such as
`v0.8.1-ce.rc.1`. Versions without the CE identifier receive `-ce` before any
`+build` metadata. Git-describe distance and dirty markers are preserved.

For example, both `0.8.0` and `v0.8.0-ce` display as `v0.8.0-ce`, while
`v0.8.0-ce-3-gabcdef-dirty` keeps its development metadata unchanged.

## Release Checklist

1. Ensure the exact release commit passes the reusable CI workflow, including
   lint, tests, race, vet, native CLI, browser, example and manifest rejection tests.
   Success on another commit cannot qualify this one.
2. Update documentation, examples, and changelog entries.
3. Tag the release: `git tag vX.Y.Z-ce && git push origin vX.Y.Z-ce`.
4. GitHub Actions (`release.yml`) first calls the same CI workflow for the tag's
   exact SHA. `release` depends on that workflow, checks out its qualified SHA and
   verifies its tree/tag before GoReleaser. Manual dispatch requires a CE tag.
   No release job runs if qualification fails. GoReleaser then:
   - Produces multi-arch archives and `SHA256SUMS.txt`
   - Embeds each platform's `release-manifest.json` and uploads the platform manifests
   - Rejects non-snapshot artifacts without matching clean-source qualification
   - Publishes Docker Hub and GHCR images (`vX.Y.Z-ce`, `latest`)
   - Updates the Homebrew tap (`homebrew-choreoatlas`)
5. Validate artifacts:
   - `brew install choreoatlas2025/homebrew-choreoatlas/choreoatlas`
   - `./scripts/install.sh --version vX.Y.Z-ce`
   - `docker run --rm choreoatlas/cli:vX.Y.Z-ce version`
6. Announce the release (GitHub Release notes, docs updates).

## Repository Scope

This repository builds and distributes the independent Community Edition. The `-ce` suffix identifies these releases; it does not select a different runtime feature set. See [DECISIONS.md](DECISIONS.md) for the repository's scope and maintenance rules.

## Artifact verification

CI raw binaries carry `build-manifest.json`; release archives contain
`release-manifest.json`. These records bind artifact size/hash, compiled full
commit, version/channel, Git tree and the exact source-check record (including
GitHub run ID/attempt when available). Build hooks inspect compiled Go metadata
rather than trusting supplied manifest fields. Qualified artifacts require a
clean build with matching VCS metadata. There is no cryptographic signature:
trust the repository's workflow/artifact origin, and verify the archive against
its published `SHA256SUMS.txt` before inspecting its embedded manifest.

After extracting a trusted archive, verify its binary against the expected full
SHA from the release tag:

```bash
python3 scripts/build-manifest.py --verify --manifest release-manifest.json \
  --artifact-dir . --commit FULL_40_CHARACTER_SHA
./choreoatlas version
```

Verification rejects a different commit, modified bytes, inconsistent compiled
identity fields, altered qualification hash, or an unqualified snapshot.
Snapshots deliberately record `qualified=false` when no qualification is
provided. `--snapshot=true` permits verifying such development bytes without
turning them into release evidence. Their Git tree is the HEAD tree, and
`sourceDirty` identifies working-tree changes; a snapshot is not a proof of a
clean revision. Actual registry publishing and installation of a tagged release
remain part of the maintainer's release checklist, not claims made by local
snapshot checks.
