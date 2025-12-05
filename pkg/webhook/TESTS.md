# Webhook Server Tests

This document describes the test suite for the webhook server package (`pkg/webhook`).

## Overview

The webhook server package implements a Kubernetes mutating admission webhook that intercepts pod creation requests and injects the MCA proxy sidecar. The tests validate HTTP endpoint handling, admission review request/response processing, pod mutation logic, and JSON patch generation.

## Test Coverage

- **TestNewServer** - validates that NewServer constructor properly initializes server with TLS certificate
- **TestServer_HandleHealth** - validates that /health endpoint returns HTTP 200 with "OK" body
- **TestServer_HandleMutate_ValidRequest** - validates that valid admission review request is processed successfully with allowed response and JSON patch
- **TestServer_HandleMutate_InvalidJSON** - validates that invalid JSON in request body returns HTTP 400 error
- **TestServer_Mutate_Success** - validates that mutate function returns allowed admission response with correct UID, patch type, and non-empty patch
- **TestServer_Mutate_InvalidPod** - validates that invalid pod JSON in admission request returns denied response with error message
- **TestServer_GenerateJSONPatch** - validates that JSON patch is generated with replace operation for /spec path
