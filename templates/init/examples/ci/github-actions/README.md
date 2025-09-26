# GitHub Actions Workflow Examples

This directory contains example GitHub Actions workflows for ChoreoAtlas CLI integration.

## Files

- `minimal.yml` - Basic validation workflow
- `combo.yml` - Complete CI/CD workflow with all features
- `discover.yml` - Contract discovery from traces (manual trigger)

## Usage

1. Copy the desired workflow file to your repository's `.github/workflows/` directory
2. Adjust paths and parameters as needed
3. Commit and push to activate

## Important Notes

- These are example files and will NOT run automatically in this repository
- They are designed to be copied to your own projects
- Adjust the paths to match your contract and trace file locations
- The Docker image `choreoatlas/cli:latest` should be available when these examples are used

## Quick Start

For a minimal setup, copy `minimal.yml` to your repo:

```bash
mkdir -p .github/workflows
cp minimal.yml .github/workflows/choreoatlas.yml
git add .github/workflows/choreoatlas.yml
git commit -m "Add ChoreoAtlas validation workflow"
git push
```

## Customization

Each workflow file contains comments explaining the configuration options and how to customize them for your specific needs.