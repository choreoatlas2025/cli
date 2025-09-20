# VSCode Schema Association Setup

## Overview

This guide explains how to configure VSCode to use the ChoreoAtlas FlowSpec schema for validation and auto-completion.

## Setup Instructions

### Method 1: Project-level Configuration (Recommended)

Add the following to your project's `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/choreoatlas2025/cli/main/schemas/flowspec.schema.json": [
      "*.flowspec.yaml",
      "*.flowspec.yml"
    ]
  }
}
```

### Method 2: User-level Configuration

Open VSCode Settings (Cmd/Ctrl + ,) and search for "yaml.schemas". Add the schema mapping in the settings.json:

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/choreoatlas2025/cli/main/schemas/flowspec.schema.json": [
      "*.flowspec.yaml",
      "*.flowspec.yml"
    ]
  }
}
```

### Method 3: In-file Schema Declaration

Add the following comment at the top of your FlowSpec file:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/choreoatlas2025/cli/main/schemas/flowspec.schema.json
info:
  title: "My Flow"
  # ... rest of your FlowSpec
```

## Required Extensions

Install the YAML Language Support extension by Red Hat:

```bash
code --install-extension redhat.vscode-yaml
```

## Features Enabled

Once configured, you'll get:

- **Validation**: Real-time validation against the FlowSpec schema
- **Auto-completion**: IntelliSense for FlowSpec properties
- **Hover Documentation**: Descriptions for properties on hover
- **Error Highlighting**: Visual indicators for schema violations
- **Format Detection**: Automatic detection of graph vs flow format

## Testing the Configuration

1. Create a new file with `.flowspec.yaml` extension
2. Start typing - you should see auto-completion suggestions
3. Try an invalid property - you should see error highlighting

Example test file:

```yaml
info:
  title: "Test Flow"
services:
  myService:
    spec: "./my-service.yaml"
graph:
  nodes:
    - id: "start"
      call: "myService.operation"
      # Type 'depends' here and see auto-completion
```

## Troubleshooting

### Schema not loading

1. Check internet connection (for remote schema)
2. Verify the YAML extension is installed and enabled
3. Reload VSCode window (Cmd/Ctrl + Shift + P -> "Reload Window")

### Auto-completion not working

1. Ensure file has `.flowspec.yaml` or `.flowspec.yml` extension
2. Check that yaml.schemas setting is properly configured
3. Look for errors in Output panel (View -> Output -> YAML)

### Using local schema

If you prefer using a local schema file:

```json
{
  "yaml.schemas": {
    "./schemas/flowspec.schema.json": [
      "*.flowspec.yaml"
    ]
  }
}
```

## Related Documentation

- [FlowSpec Schema Reference](../flowspec/schema.md)
- [Graph Format Guide](../flowspec/graph-format.md)
- [Migration from Flow to Graph](../migration/flow-to-graph.md)