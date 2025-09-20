# Installation Guide

## Overview

ChoreoAtlas CLI Community Edition (CE) can be installed through multiple channels. This is the **zero-telemetry** edition with no data collection, no network calls, and complete offline capability.

## Installation Methods

### Method 1: Docker (Recommended)

```bash
# Pull from Docker Hub
docker pull choreoatlas/cli:latest

# Run directly
docker run --rm -v $(pwd):/workspace choreoatlas/cli:latest lint --flow /workspace/flow.yaml

# Create an alias for convenience
alias choreoatlas='docker run --rm -v $(pwd):/workspace choreoatlas/cli:latest'
```

### Method 2: Homebrew (macOS/Linux)

```bash
# Add the ChoreoAtlas tap
brew tap choreoatlas2025/tap

# Install ChoreoAtlas CLI
brew install choreoatlas

# Verify installation
choreoatlas version
```

### Method 3: Direct Download

Download the appropriate binary for your platform from the [releases page](https://github.com/choreoatlas2025/cli/releases).

#### Linux
```bash
# Download (replace VERSION with actual version number)
curl -L https://github.com/choreoatlas2025/cli/releases/download/vVERSION/choreoatlas-linux-amd64 -o choreoatlas

# Make executable
chmod +x choreoatlas

# Move to PATH
sudo mv choreoatlas /usr/local/bin/

# Verify
choreoatlas version
```

#### macOS
```bash
# Download (replace VERSION with actual version number)
curl -L https://github.com/choreoatlas2025/cli/releases/download/vVERSION/choreoatlas-darwin-amd64 -o choreoatlas

# Make executable
chmod +x choreoatlas

# Move to PATH
sudo mv choreoatlas /usr/local/bin/

# Verify
choreoatlas version
```

#### Windows
1. Download `choreoatlas-windows-amd64.exe` from the [releases page](https://github.com/choreoatlas2025/cli/releases)
2. Rename to `choreoatlas.exe`
3. Add to your PATH environment variable
4. Open a new terminal and verify with `choreoatlas version`

### Method 4: Build from Source

```bash
# Clone the repository
git clone https://github.com/choreoatlas2025/cli.git
cd cli

# Build
go build -o bin/choreoatlas ./cmd/choreoatlas/

# Install to PATH
sudo cp bin/choreoatlas /usr/local/bin/

# Verify
choreoatlas version
```

## Verify Installation

After installation, verify the CLI is working correctly:

```bash
# Check version (should show vX.Y.Z-ce)
choreoatlas version

# Expected output:
# choreoatlas v0.7.0-ce
# Edition: Community Edition (CE)
# Git Commit: xxxxxxx
# Build Time: 2024-01-20T10:00:00Z
# Go Version: go1.21.0
# Platform: darwin/amd64

# Test with example
choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
```

## Command Aliases

For convenience, `ca` is available as a shorter alias:

```bash
# These are equivalent
choreoatlas lint --flow flow.yaml
ca lint --flow flow.yaml
```

## Privacy & Zero Telemetry

**ChoreoAtlas CE is a zero-telemetry edition:**

- ✅ **No data collection**: The CE edition collects absolutely no usage data
- ✅ **No network calls**: Operates completely offline, no external API calls
- ✅ **No tracking**: No user identification, no analytics, no metrics
- ✅ **Fully private**: All processing happens locally on your machine
- ✅ **Air-gap ready**: Can be used in isolated environments without internet

### Verification

You can verify the binary has no telemetry dependencies:

```bash
# Check for telemetry-related symbols (should return nothing)
nm choreoatlas | grep -i telemetry
nm choreoatlas | grep -i analytics
nm choreoatlas | grep -i track

# Check for network-related imports (only standard library)
go version -m choreoatlas | grep -E "sentry|datadog|newrelic|elastic"
```

## System Requirements

- **Operating Systems**: Linux, macOS, Windows
- **Architecture**: amd64, arm64
- **Memory**: 256MB minimum
- **Disk Space**: 50MB for binary
- **Runtime**: No external dependencies required

## Configuration

ChoreoAtlas CE requires no configuration to start using. All features work out-of-the-box:

```bash
# No config needed - just run
choreoatlas lint --flow myflow.yaml
choreoatlas validate --flow myflow.yaml --trace trace.json
```

## Getting Started

1. Install using one of the methods above
2. Create or use existing FlowSpec files
3. Run validation:
   ```bash
   choreoatlas lint --flow examples/flows/order-fulfillment.flowspec.yaml
   ```

## Troubleshooting

### "Command not found" after installation
- Ensure the binary is in your PATH
- Try using the full path to the binary
- On macOS, you may need to allow the binary in Security settings

### "Permission denied" on Linux/macOS
- Make the binary executable: `chmod +x choreoatlas`
- If installing to `/usr/local/bin`, use `sudo`

### Docker volume mounting issues
- Use absolute paths for volume mounts
- Ensure the local directory exists and has proper permissions

## Support

- **Documentation**: [GitHub Wiki](https://github.com/choreoatlas2025/cli/wiki)
- **Issues**: [GitHub Issues](https://github.com/choreoatlas2025/cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/choreoatlas2025/cli/discussions)

## License

Apache 2.0 - See [LICENSE](https://github.com/choreoatlas2025/cli/blob/main/LICENSE) for details.