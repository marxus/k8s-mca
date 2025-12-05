# Serve Package Tests

This document describes the test suite for the serve package (`pkg/serve`).

## Overview

The serve package contains high-level functions for starting the MCA proxy and webhook servers. It handles file operations for service account credentials, certificate generation, Kubernetes client creation, and webhook configuration patching. The tests validate file I/O operations, JSON patch generation, and webhook patching logic using fake Kubernetes clients.

## Test Coverage

### proxy_test.go - File Operations

- **TestWriteCACertificate** - validates that CA certificate PEM is written to the correct serviceaccount path
- **TestWriteCACertificate_Error** - validates that write errors are properly handled and returned
- **TestWriteNamespaceFile** - validates that namespace file is copied from serviceaccount to mca-serviceaccount directory
- **TestWriteNamespaceFile_SourceNotFound** - validates that missing source namespace file returns appropriate error
- **TestWriteTokenFile** - validates that placeholder token file is written to mca-serviceaccount directory
- **TestWriteTokenFile_Error** - validates that write errors are properly handled and returned

### webhook_test.go - Webhook Configuration

- **TestBuildWebhookPatch** - validates that JSON patch is correctly formatted with base64-encoded CA certificate
- **TestBuildWebhookPatch_EmptyCert** - validates that empty certificate data is handled correctly in patch generation
- **TestPatchMutatingConfig** - validates that webhook patch is called with correct webhook name, patch type, and patch data using fake clientset
- **TestPatchMutatingConfig_PatchError** - validates that Kubernetes API errors during patching are properly handled and returned
- **TestStartWebhook_NamespaceFileNotFound** - validates that missing namespace file returns appropriate error during webhook startup
