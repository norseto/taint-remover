# Codebase Overview

This document summarizes the main components of the project for new contributors.

## Repository Structure

- `api/v1alpha1/` – Defines the **TaintRemover** custom resource type.
- `internal/controller/` – Contains the reconciliation logic that watches both TaintRemover CRs and Node events.
- `internal/taints/` – Utilities for handling Kubernetes taints, derived from upstream code.
- `cmd/main.go` – Entry point that configures the controller manager and health checks.
- `config/` – Kustomize templates for CRDs, RBAC, and sample manifests.
- `dist/` – Pre-generated install manifest for direct deployment.
- `hack/` – Helper scripts for versioning and release management.

## Key Features

- Removes spot instance taints set by cloud providers.
- Operates at cluster scope and reacts to changes in both custom resources and nodes.
- Includes a Makefile with tasks for building, testing, and deploying the controller.

Refer to `README.md` for deployment instructions and `CLAUDE.md` for more development details.
