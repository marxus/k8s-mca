# Proxy Server Tests

This document describes the test suite for the proxy server package (`pkg/proxy`).

## Overview

The proxy server package provides an HTTP reverse proxy that intercepts and forwards requests to Kubernetes API servers. It removes Authorization headers from incoming requests and forwards them through pre-configured reverse proxies. The tests validate dependency injection, request/response forwarding, and header manipulation.

## Test Coverage

- **TestNewServer** - validates that NewServer constructor properly initializes server with TLS certificate and reverse proxies map
- **TestServer_Handler_RemovesAuthorizationHeader** - validates that Authorization header is removed from requests before forwarding while preserving other headers
- **TestServer_Handler_ForwardsRequestToBackend** - validates that requests are forwarded to the backend with correct method, path, and response body
- **TestServer_Handler_ForwardsResponseStatusAndBody** - validates that backend response status code, body, and headers are correctly forwarded to the client
