# Privacy Policy - ChoreoAtlas CE

## Zero Usage Telemetry Commitment

ChoreoAtlas Community Edition is an independent, local CLI. It does not collect
or upload usage statistics, analytics, crash reports, installation counts, or
user identifiers. It does not check for updates or fetch remote configuration.

All contract validation, trace analysis, baseline comparison, and report
generation run locally. Contracts, traces, reports, and execution output remain
under your control.

OpenTelemetry attributes in traces are business-system inputs supplied by you.
Processing those attributes locally is separate from collecting product usage
telemetry. Reports can contain data from your traces; control their storage and
sharing as you would the original inputs.

## Network Isolation

The CLI's local workflows require no outbound connections, license validation,
or remote feature configuration. Downloading releases, installing through a
package manager, and obtaining build dependencies are distribution steps; they
are separate from running local validation.

## Verification Methods

### 1. Source and Build Audit

Build the default CE executable without edition tags:

```bash
go build -o choreoatlas ./cmd/choreoatlas
./choreoatlas version
go version -m choreoatlas

# Review runtime code for network clients and outbound calls.
rg -n 'net/http|http\.(Get|Post|NewRequest)|net\.Dial' \
  cmd internal --glob '*.go' --glob '!*_test.go'
```

The repository is public at https://github.com/choreoatlas2025/cli. Review the
source and dependency graph together. A binary string search for words such as
`telemetry`, `tracking`, or HTTP URLs does not establish whether the executable
sends data: those strings can occur in trace fields, schemas, or library code.

### 2. Runtime Verification

Run representative discovery, validation, baseline, and report workflows in an
environment with outbound networking disabled. They should continue to work
from local inputs. Network monitoring can supplement that check; observing only
connections to one hostname or port does not cover all possible destinations.

### 3. Release Verification

Download artifacts from the official release channel, verify their published
checksums, and review changes between versions. Source builds use the same CE
scope as distributed executables; an edition build tag is not needed to enable it.

## Your Rights

You can audit the source, build from source, control your local inputs and
outputs, and modify or redistribute the software under Apache-2.0.

## Contact

For privacy-related questions:

- Open an issue: https://github.com/choreoatlas2025/cli/issues
- Email: choreoatlas@gmail.com (public inbox)

## Commitment

Zero usage telemetry and local execution are part of this repository's scope.
See [DECISIONS.md](../DECISIONS.md) for its maintenance rules.

*Last updated: 2026-10-04*

*Policy version: 1.1.0*
