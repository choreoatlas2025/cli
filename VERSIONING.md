# ChoreoAtlas CLI Versioning Strategy

## Version Format

The project follows Semantic Versioning with edition qualifiers:

```
v{MAJOR}.{MINOR}.{PATCH}[-{EDITION}][.{PRERELEASE}]
```

### Components

- **MAJOR**: Breaking API changes
- **MINOR**: New features, backwards compatible
- **PATCH**: Bug fixes, backwards compatible
- **EDITION**: Product edition identifier
  - `ce` - Community Edition
  - `pro` - Professional Edition (includes Pro-Free and Pro-Privacy)
  - `cloud` - Cloud Edition
- **PRERELEASE**: Optional pre-release identifier
  - `alpha.N` - Alpha releases
  - `beta.N` - Beta releases
  - `rc.N` - Release candidates

## Version History

### Legacy Versions (Pre-CE Split)
- v0.1.0 to v0.1.5: Original unified codebase

### Community Edition (CE)
- v0.2.0-ce.beta.1: First CE beta release (current)
  - Zero telemetry implementation
  - Complete discover functionality
  - GitHub Actions integration

### Planned Releases
- v0.2.0-ce.beta.2: User feedback incorporation
- v0.2.0-ce.rc.1: Release candidate
- v0.2.0-ce: First stable CE release
- v0.3.0-ce: Feature enhancements

## Branch Strategy

- `main`: Current development (CE-focused)
- `release/v*`: Release branches
- `feature/*`: Feature development
- Tags: `v*` for releases

## Version Comparison

| Edition | Version Range | Features | Telemetry |
|---------|--------------|----------|-----------|
| Legacy | v0.1.x | All features mixed | Optional |
| CE | v0.2.x-ce | Core features only | None |
| Pro | v0.2.x-pro | Advanced features | Optional |
| Cloud | v0.2.x-cloud | Cloud features | Required |

## Build Version Injection

Versions are injected at build time:

```bash
# Automatic version from git
make build

# Manual version override
VERSION=v0.2.0-ce make build
```

## Release Process

1. **Beta Phase** (current)
   - Tag: v0.2.0-ce.beta.N
   - Purpose: Early user testing
   - Duration: 2-4 weeks

2. **Release Candidate**
   - Tag: v0.2.0-ce.rc.N
   - Purpose: Final testing
   - Duration: 1 week

3. **Stable Release**
   - Tag: v0.2.0-ce
   - Purpose: Production use
   - Support: Bug fixes in v0.2.x-ce

## Migration Path

Users upgrading from legacy versions:

- v0.1.x → v0.2.0-ce: Clean install recommended
- Configuration changes required for edition-specific features
- No data migration needed (stateless tool)