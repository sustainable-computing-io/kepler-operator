# 🚀 Kepler Operator Release Process

This document outlines the automated release workflow for Kepler Operator.

## 📋 Release Workflow Overview

The release process for Kepler Operator is automated through GitHub Actions. When triggered, it handles version updates, image building, and publishing of artifacts.

Before packaging the Helm chart, the release workflow updates its `version` and `appVersion` and sets `operator.image` in `values.yaml` to the operator image built for that release. It runs `make helm-validate` to check the chart, including that the default operator image tag matches the chart version. The existing version-update PR includes these Helm changes.

## ⚠️ Important Notes

- Ensure you have proper permissions to trigger the release workflow
- Double-check the new version and which old operator version to be replaced before triggering the release
- Images are pushed to Quay.io registry, ensure credentials are properly configured in GitHub Secrets
- After release, verify that:
  - Images are available in the registry
  - GitHub Release is created with proper tags
  - Pull Request for version and bundle updates is created and merged

![Release Workflow Diagram](assets/release.png)
