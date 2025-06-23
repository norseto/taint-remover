# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes controller called `taint-remover` that automatically removes specified taints from cluster nodes. It's built using the Kubebuilder framework and watches for both TaintRemover custom resources and Node events to maintain the desired taint state.

## Architecture

The project follows the standard Kubernetes controller pattern:

- **API Types** (`api/v1alpha1/`): Defines the TaintRemover Custom Resource Definition (CRD) with a simple spec containing the list of taints to remove
- **Controller** (`internal/controller/`): Contains the main reconciliation logic and event handlers for both TaintRemover CRDs and Node resources
- **Taints Utility** (`internal/taints/`): Kubernetes-derived utilities for taint manipulation (Apache 2.0 licensed)
- **Main Entry Point** (`cmd/main.go`): Sets up the controller manager with proper RBAC, metrics, and health checks

The controller operates at cluster scope and uses a dual-watch strategy:
1. Watches TaintRemover CRDs to understand which taints should be removed
2. Watches Node events to immediately process newly tainted nodes

## Development Commands

### Building and Testing
```bash
# Build the manager binary (includes manifests generation, formatting, and vetting)
make build

# Run tests with coverage
make test

# Run controller locally (for development)
make run

# Format and vet code
make fmt
make vet
```

### Code Generation
```bash
# Generate Kubernetes manifests (RBAC, CRDs, webhooks)
make manifests

# Generate DeepCopy methods
make generate
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Build multi-platform images
make docker-buildx

# Push image
make docker-push
```

### Deployment
```bash
# Install CRDs to cluster
make install

# Deploy controller to cluster
make deploy

# Create installation YAML
make build-installer

# Remove from cluster
make undeploy
make uninstall
```

### Running Single Tests
```bash
# Run specific test file
go test ./internal/controller -run TestTaintRemoverReconciler

# Run with verbose output
go test -v ./internal/taints
```

## Key Implementation Details

- The controller patches nodes using Strategic Merge Patch to remove matching taints
- Uses controller-runtime's predicate system to only process nodes with actual changes
- Implements error handling that continues processing all nodes even if some fail
- Maintains a unique taint registry to avoid duplicate processing across multiple TaintRemover resources
- Includes comprehensive RBAC permissions for cluster-wide node access

## Testing Setup

The project uses Ginkgo/Gomega for testing with envtest for integration tests. Test environment setup is handled by the `ENVTEST_K8S_VERSION` variable in the Makefile.