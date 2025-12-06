# k8s-mca (Multi Cluster Adapter)

MCA is a router for Kubernetes API requests, implemented via a sidecar that intercepts requests using a MITM reverse proxy. The purpose of this project is to transparently enable multi-cluster mode for Kubernetes-native apps that don't include multi-cluster support, by routing API requests based on analysis of request body labels, annotations, or other criteria.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Pod Creation (label: mca.k8s.io/inject: "true")                │
└────────────────────────────┬────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────────┐
│  Webhook Intercepts → Injects mca-proxy Init Container          │
└────────────────────────────┬────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────────┐
│  App Container                                                  │
│    ↓ KUBERNETES_SERVICE_HOST=127.0.0.1:6443                     │
│    ↓                                                            │
│  MCA Proxy (analyzes request, routes to destination cluster)    │
│    ↓ (applies proper auth for target cluster)                   │
│  Kubernetes API Server                                          │
└─────────────────────────────────────────────────────────────────┘
```

**⚠️ Experimental / Prototype Project**

This is a proof-of-concept demonstrating MITM proxy patterns for Kubernetes API requests. Not intended for production use.

## Build

```bash
# Clone the repository
git clone https://github.com/marxus/k8s-mca.git
cd k8s-mca

# Build the Docker image
docker build -t ghcr.io/marxus/k8s-mca:latest .

# Push to registry
docker push ghcr.io/marxus/k8s-mca:latest
```

## Installation

```bash
# Install or upgrade in the mca namespace
helm upgrade --install mca ./charts/mca \
  --namespace mca \
  --create-namespace \
  --set image.repository=ghcr.io/marxus/k8s-mca \
  --set image.tag=latest
```

This installs:
- Webhook server with MutatingWebhookConfiguration
- Required RBAC and ServiceAccount

## Project Progress

### Implemented

- ✅ TLS certificate generation (`pkg/certs`)
- ✅ Pod mutation and sidecar injection (`pkg/inject`)
- ✅ HTTP reverse proxy with Authorization header removal (`pkg/proxy`)
- ✅ Kubernetes webhook server (`pkg/webhook`)
- ✅ CLI with inject/proxy/webhook modes (`cmd/mca`)
- ✅ Helm chart with webhook configuration (`charts/mca`)
- ✅ Multi-arch Docker build via CI
- ✅ Comprehensive test coverage (41 tests across 5 packages)

### Future Exploration

- Multi-cluster routing logic
- Multi-cluster configuration supporting everything the Go Kubernetes SDK supports
- Production hardening and security review

## Local Development

### Prerequisites

- Go 1.24.3
- Access to a Kubernetes cluster
- kubectl configured with appropriate context

### How to Run Inject Locally

The inject mode reads pod YAML from stdin and outputs mutated YAML to stdout:

```bash
# Basic usage
cat pod.yaml | go run ./cmd/mca --inject > mutated-pod.yaml

# Or with kubectl
kubectl get pod my-pod -o yaml | go run ./cmd/mca --inject | kubectl apply -f -
```

**What it does:**
- Adds `mca-proxy` init container as first init container
- Modifies all containers to redirect Kubernetes API calls to `127.0.0.1:6443`
- Adds volume mount at `/var/run/secrets/kubernetes.io/serviceaccount`
- Sets env vars: `KUBERNETES_SERVICE_HOST=127.0.0.1`, `KUBERNETES_SERVICE_PORT=6443`

### How to Run Webhook Locally

```bash
# Set the Kubernetes context (optional, defaults to "mca-k8s-ctx")
export MCA_K8S_CTX=my-cluster-context

# Run the webhook server
go run ./cmd/mca --webhook
```

**How it works:**
- Listens on port `:8443`
- **Automatically patches existing `mca-webhook` MutatingWebhookConfiguration** with generated CA certificate
- Uses kubeconfig context specified by `MCA_K8S_CTX` environment variable

**Endpoints:**
- `/mutate` - Webhook admission endpoint
- `/health` - Health check endpoint

**⚠️ Troubleshooting:**
- Requires cluster to have existing `mca-webhook` resource - see [Installation](#installation) section
- This will patch the cluster's `mca-webhook` with the updated CA certificate, but it won't actually receive any traffic unless using tools like `mirrord`
- If you want to test the injection logic, just use the CLI's `--inject` mode

### How to Run Proxy Locally

```bash
# Set the Kubernetes context (optional, defaults to "mca-k8s-ctx")
export MCA_K8S_CTX=my-cluster-context

# Run the proxy server
go run ./cmd/mca --proxy
```

**What it does:**
1. Generates TLS certificates for localhost/127.0.0.1
2. Writes credential files to `./tmp/var/run/secrets/kubernetes.io/mca-serviceaccount/`:
   - `ca.crt` - Generated CA certificate
   - `namespace` - Pod namespace (defaults to "default")
   - `token` - Placeholder token file (contains "-")
3. Creates reverse proxy to real Kubernetes API (using kubeconfig context from `MCA_K8S_CTX`)
4. Removes Authorization headers and applies in-cluster auth config
5. Listens on `127.0.0.1:6443` with HTTPS

**Testing with kubectl:**

Use the provided `kubectl.sh` helper script to test API calls through the proxy:

```bash
./kubectl.sh get pods -A
./kubectl.sh get nodes
```

This creates a temporary kubeconfig pointing to `127.0.0.1:6443` with the MCA CA certificate.

## Environment Variables

**Development mode** (default):
- `MCA_K8S_CTX` - Kubernetes context (default: "mca-k8s-ctx")

## Package Structure

```
pkg/
├── certs/       - TLS certificate generation
├── inject/      - Pod mutation and sidecar injection logic
├── proxy/       - HTTP reverse proxy server
├── webhook/     - Kubernetes webhook server
└── serve/       - High-level functions to start proxy and webhook

cmd/mca/         - Main CLI entry point

charts/mca/      - Helm chart for deployment
```

## Development

**Testing:**
```bash
go test ./...
```

All tests: 41 tests across 5 packages

**Build:**
```bash
# Development build
go build -o mca ./cmd/mca

# Release build (static binary)
go build -tags=release -o mca ./cmd/mca
```

**Docker:**
```bash
docker build -t mca:latest .
```

Multi-stage build with support for prebuilt binaries.

## CLI Usage

```
Usage: mca [--inject|--proxy|--webhook]
  --inject   Inject MCA sidecar into Pod manifest (stdin/stdout)
  --proxy    Start MCA proxy server
  --webhook  Start MCA webhook server
```

## License

MIT
