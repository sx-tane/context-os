# Docker

Container and local infrastructure definitions for running ContextOS services in a reproducible environment.

## Files

- `Dockerfile.api`: Go API container image.
- `Dockerfile.frontend`: frontend development/runtime container image.
- `Dockerfile.worker`: AI worker container image.

## Usage

- Use these images for local containerized workflows and CI-compatible environment setup.
- Keep Dockerfile dependencies aligned with the local setup scripts.
- Update this README when new service images or shared base-image expectations are added.

## Maintenance Checklist

- Keep build steps deterministic and minimal.
- Document new runtime ports, environment variables, or mounted volumes.
- Recheck local startup docs when Docker workflows change.
