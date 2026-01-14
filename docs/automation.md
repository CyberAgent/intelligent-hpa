# Automation

This document describes the automated build and deployment workflows for Intelligent HPA.

## Overview

The project uses GitHub Actions to automate the building and publishing of container images and Kubernetes manifests.

## Container Images

Container images are built and published to GitHub Container Registry (ghcr.io) automatically.

### Controller Image

- **Registry**: `ghcr.io/cyberagent/intelligent-hpa/intelligent-hpa-controller`
- **Source**: `ihpa-controller/Dockerfile`
- **Build Context**: `./ihpa-controller`

### FittingJob Image

- **Registry**: `ghcr.io/cyberagent/intelligent-hpa/intelligent-hpa-fittingjob`
- **Source**: `fittingjob/Dockerfile`
- **Build Context**: `./fittingjob`

## Workflow Triggers

The automation workflow is triggered by:

1. **Push to main/master branch**: Builds and pushes images with the `latest` tag and updates manifests
2. **Push tags (v*)**: Builds and pushes images with version tags (e.g., `v1.0.0`, `1.0`)
3. **Pull requests**: Builds images but does not push them (validation only)

## Image Tags

The following tags are automatically generated:

- `latest`: Latest build from the default branch (main/master)
- `<branch-name>`: Branch name for branch builds
- `<branch-name>-<sha>`: Branch name with commit SHA
- `<version>`: Semantic version from git tags (e.g., `1.0.0`)
- `<major>.<minor>`: Semantic version without patch (e.g., `1.0`)

## Manifest Generation

The Kubernetes manifest is automatically generated and updated in `manifests/intelligent-hpa.yaml` when:

1. Images are successfully built and pushed
2. The event is not a pull request

The manifest generation process:

1. Runs `make manifests` in the ihpa-controller directory to generate CRDs and RBAC
2. Updates the controller image reference in kustomization to use the appropriate tag
3. Builds the final manifest using kustomize
4. Commits and pushes the updated manifest back to the repository

## Manual Build

You can also build and push images manually using the Makefile:

```bash
# Build and push controller image
make controller

# Build and push fittingjob image
make fittingjob

# Generate manifest
make manifest
```

By default, the Makefile uses GitHub Container Registry. You can override the registry by setting the `REGISTRY` environment variable:

```bash
# Use a different registry
REGISTRY=docker.io/myorg make controller
```

## Permissions

The workflow requires the following permissions:

- `contents: read` - To checkout the repository
- `packages: write` - To push images to GitHub Container Registry

These permissions are automatically granted to the `GITHUB_TOKEN` in the workflow.

## Using Published Images

To use the published images in your Kubernetes cluster:

```bash
# Pull the latest controller image
docker pull ghcr.io/cyberagent/intelligent-hpa/intelligent-hpa-controller:latest

# Pull the latest fittingjob image
docker pull ghcr.io/cyberagent/intelligent-hpa/intelligent-hpa-fittingjob:latest
```

Note: GitHub Container Registry images are public by default for public repositories. For private repositories, you'll need to authenticate:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```
