# Privacy Policy - ChoreoAtlas CE

## Zero Telemetry Commitment

ChoreoAtlas Community Edition (CE) is designed with **absolute privacy** as a core principle. We collect **ZERO** data from CE users.

## What We DON'T Collect

The CE edition does NOT:
- ❌ Collect usage statistics
- ❌ Track command executions
- ❌ Monitor feature usage
- ❌ Send crash reports
- ❌ Phone home for updates
- ❌ Transmit any data over network
- ❌ Store user identifiers
- ❌ Generate anonymous IDs
- ❌ Log IP addresses
- ❌ Track installation counts

## What Stays on Your Machine

Everything. All operations are 100% local:
- ✅ FlowSpec validation
- ✅ Trace analysis
- ✅ Report generation
- ✅ Error messages
- ✅ Configuration
- ✅ Execution logs

## Network Isolation

ChoreoAtlas CE can operate in completely air-gapped environments:
- No outbound connections required
- No update checks
- No license validation
- No feature flag fetching
- No remote configuration

## Verification Methods

### 1. Binary Inspection

You can verify our zero-telemetry claim:

```bash
# Search for telemetry-related symbols (should return empty)
strings choreoatlas | grep -i telemetry
strings choreoatlas | grep -i analytics
strings choreoatlas | grep -i tracking

# Check for common telemetry SDKs (should return empty)
strings choreoatlas | grep -E "segment|mixpanel|amplitude|posthog|sentry"

# Examine network-related imports
go version -m choreoatlas
```

### 2. Network Monitoring

Monitor network activity during execution:

```bash
# macOS
sudo tcpdump -i any host github.com or host api.github.com

# Linux
sudo tcpdump -i any port 443 or port 80

# Run ChoreoAtlas (should show no network activity)
choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
```

### 3. Source Code Audit

The source code is fully open:
- Repository: https://github.com/choreoatlas2025/cli
- No telemetry dependencies in `go.mod`
- No analytics code in source
- No network calls except for standard library

## Comparison with Other Editions

| Feature | CE (Community) | Pro-Free | Pro-Privacy | Cloud |
|---------|---------------|----------|-------------|-------|
| Telemetry | ❌ None | ✅ Optional | ❌ None | ✅ Required |
| Network Calls | ❌ None | ✅ Update checks | ❌ None | ✅ API calls |
| Data Collection | ❌ Zero | ✅ Anonymous | ❌ Zero | ✅ Account-based |
| Offline Mode | ✅ Always | ⚠️ Partial | ✅ Always | ❌ Requires connection |

## Build Verification

Our CI/CD pipeline ensures telemetry-free builds:

```yaml
# Build flags used for CE
go build -tags ce -ldflags "-X main.Version=vX.Y.Z"

# No telemetry modules included
# No analytics SDKs linked
# No tracking code compiled
```

## Your Rights

With ChoreoAtlas CE, you have the right to:
- Complete privacy of your validation workflows
- Full control over your data
- Audit the source code
- Build from source
- Modify and redistribute (per Apache 2.0 license)

## Security Considerations

While we don't collect data, we recommend:
- Download binaries only from official sources
- Verify checksums when available
- Build from source for maximum trust
- Review code changes between versions

## Contact

For privacy-related questions:
- Open an issue: https://github.com/choreoatlas2025/cli/issues
- Email: choreoatlas@gmail.com (public inbox)

## Commitment

This zero-telemetry commitment is permanent for the CE edition. Any future changes would require:
1. Major version bump
2. Clear changelog entry
3. Updated documentation
4. Separate edition/binary

We will NEVER silently add telemetry to the CE edition.

---

*Last updated: January 2025*
*Policy version: 1.0.0*