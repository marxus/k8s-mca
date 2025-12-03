# MCA (Multi Cluster Adapter)

[![CI/CD](https://github.com/marxus/k8s-mca/workflows/release/badge.svg)](https://github.com/marxus/k8s-mca/actions)

MCA is a Kubernetes MITM (Man-in-the-Middle) proxy that intercepts API calls by injecting a sidecar proxy into pods. It redirects all Kubernetes API traffic through a local HTTPS proxy at `127.0.0.1:6443`, enabling transparent inspection, modification, and routing of API requests.

## Overview

### What is MCA?

MCA acts as a transparent reverse proxy for the Kubernetes API server. By overriding the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables, all application containers connect to the MCA proxy instead of directly to the API server. This enables:

- **Transparent Interception**: No application code changes required
- **Zero-Trust Security**: Applications never see real cluster credentials
- **Request Inspection**: Log and analyze all API calls
- **Future Multi-Cluster Routing**: Foundation for routing requests to multiple clusters (Phase 2)

### Current Status: Phase 1 Complete ✓

- ✅ **MITM Proxy**: Fully functional reverse proxy to current cluster
- ✅ **Certificate Management**: Auto-generates TLS certificates with proper SANs
- ✅ **ServiceAccount Isolation**: Bypass pattern with dual ServiceAccount volumes
- ✅ **Pod Injection**: CLI tool and Webhook for automatic sidecar injection
- ✅ **Admission Webhook**: Production-ready webhook server with automatic CA bundle management
- ✅ **Helm Charts**: Complete deployment package with RBAC

## Architecture

### How It Works

```
┌─────────────────────────────────────────┐
│                  Pod                    │
├──────────────────┬──────────────────────┤
│  Init Container  │  Application         │
│  (mca-proxy)     │  Container           │
│                  │                      │
│  ┌─────────────┐ │  ┌─────────────┐     │
│  │ Proxy Server│ │  │ K8s Client  │     │
│  │ :6443       │◄┼──┤ 127.0.0.1:  │     │
│  │             │ │  │ 6443        │     │
│  └──────┬──────┘ │  └─────────────┘     │
│         │        │                      │
│  Real SA Token   │  Fake SA + Custom CA │
└─────────┼────────┴──────────────────────┘
          │
          ▼
   ┌──────────────┐
   │ K8s API      │
   │ Server       │
   └──────────────┘
```

### Key Components

1. **Init Container with `restartPolicy: Always`**: Acts as a true sidecar despite being in initContainers
2. **Environment Variable Override**: `KUBERNETES_SERVICE_HOST=127.0.0.1` redirects traffic
3. **Dual ServiceAccount Volumes**:
   - `kube-api-access-sa`: Real token for MCA proxy
   - `kube-api-access-mca-sa`: Fake token with custom CA for applications
4. **Automatic Injection**: Webhook or CLI modifies pod specs transparently

## Quick Start

### Prerequisites

- Kubernetes cluster (1.16+)
- kubectl configured
- Helm 3 (for webhook deployment)

### Option 1: CLI Injection (Development)

```bash
# Install from release
go install github.com/marxus/k8s-mca/cmd/mca@latest

# Or build from source
git clone https://github.com/marxus/k8s-mca.git
cd k8s-mca
go build -o mca cmd/mca/main.go

# Inject MCA into a pod manifest
cat pod.yaml | ./mca --inject > pod-with-mca.yaml
kubectl apply -f pod-with-mca.yaml
```

### Option 2: Webhook (Production)

```bash
# Install webhook using Helm
helm install mca ./charts/mca \
  --set image.repository=your-registry/mca \
  --set image.tag=v0.1.0

# Label pods for automatic injection
kubectl label pod my-pod mca.k8s.io/inject=true
```

## Usage Examples

### Example 1: Simple Pod Injection

```bash
# Original pod
cat <<EOF | ./mca --inject | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: kubectl
    image: bitnami/kubectl:latest
    command: ["/bin/sh", "-c", "kubectl get pods && sleep 3600"]
EOF
```

### Example 2: Webhook Injection

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: auto-injected
  labels:
    mca.k8s.io/inject: "true"  # Webhook triggers on this label
spec:
  containers:
  - name: app
    image: my-app:latest
# MCA automatically injects proxy sidecar
```

### Example 3: Local Development

```bash
# Set Kubernetes context for development
export MCA_K8S_CTX=my-cluster-context

# Run proxy locally
go run cmd/mca/main.go --proxy

# In another terminal, test with kubectl using MCA proxy
./kubectl.sh get pods -A
```

## Configuration

### Build Configurations

MCA uses build tags to create optimized binaries for different environments:

| Build | Filesystem | K8s Auth | Proxy Bind | Use Case |
|-------|------------|----------|------------|----------|
| `release` | Real OS | In-cluster | 127.0.0.1:6443 | Production |
| `develop` | Sandboxed (./tmp) | Kubeconfig | 0.0.0.0:6443 | Local dev |
| `testing` | In-memory | Mock | 127.0.0.1:6443 | Unit tests |

```bash
# Build for production
go build -tags release -o mca-release cmd/mca/main.go

# Build for development (default)
go build -o mca-dev cmd/mca/main.go
```

### Environment Variables

**Webhook Deployment:**
```bash
MCA_PROXY_IMAGE=your-registry/mca:v0.1.0  # Proxy image to inject
MCA_WEBHOOK_NAME=mca-webhook               # Webhook service name
```

**Development:**
```bash
MCA_K8S_CTX=my-kube-context  # Kubernetes context to use
```

## Development

### Project Structure

```
k8s-mca/
├── cmd/mca/              # Main entry point
├── pkg/
│   ├── inject/          # Pod mutation logic (CLI & webhook)
│   ├── proxy/           # HTTPS reverse proxy server
│   ├── webhook/         # Admission webhook HTTP handler
│   ├── serve/           # Server lifecycle orchestration
│   └── certs/           # TLS certificate generation
├── conf/                # Build-time configurations
├── charts/mca/          # Helm deployment chart
└── design/              # Architecture & decision docs
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/inject/
```

### Local Development Workflow

```bash
# 1. Start proxy locally
export MCA_K8S_CTX=minikube
go run cmd/mca/main.go --proxy

# 2. Test with custom kubectl
./kubectl.sh get pods

# 3. Inject into test pod
cat test-pod.yaml | go run cmd/mca/main.go --inject | kubectl apply -f -
```

## Deployment

### Helm Chart

The Helm chart includes:
- **Deployment**: Webhook server with single replica
- **Service**: Routes admission requests to webhook
- **ServiceAccount**: Identity for webhook pod
- **RBAC**: Permissions to patch MutatingWebhookConfiguration
- **MutatingWebhookConfiguration**: Triggers on pods with `mca.k8s.io/inject=true`

```bash
# Install chart
helm install mca ./charts/mca

# Upgrade chart
helm upgrade mca ./charts/mca

# Uninstall
helm uninstall mca
```

### CI/CD Integration

The project includes GitHub Actions workflow for:
- Building multi-arch Docker images (amd64, arm64)
- Running automated tests
- Publishing releases on git tags
- Deploying Helm charts

```bash
# Trigger development build
git tag v0.0.0-develop
git push origin v0.0.0-develop

# Trigger production release
git tag v0.1.0
git push origin v0.1.0
```

## Security Considerations

### Design Principles

1. **ServiceAccount Isolation**: Applications never access real cluster credentials
2. **TLS Everywhere**: All communication encrypted with custom CA
3. **Authorization Preserved**: Real RBAC policies enforced at API server
4. **Non-Root Containers**: MCA runs as UID 999 with `runAsNonRoot: true`

### Threat Model

| Threat | Mitigation |
|--------|------------|
| App reads real SA token | Token mounted only to MCA, not application |
| App bypasses proxy | Environment variables force localhost connection |
| Certificate compromise | Certificates regenerated on every pod start |
| Privilege escalation | MCA runs non-root, no privileged capabilities |

## Roadmap

### Phase 2: Multi-Cluster Routing (Planned)
- Cluster registry configuration
- Label-based routing rules
- Multi-cluster load balancing
- Per-cluster credential management

### Phase 3: Observability (Planned)
- Prometheus metrics export
- Distributed tracing integration
- API usage analytics
- Health monitoring dashboard

### Phase 4: Production Hardening (Planned)
- Connection pooling and caching
- Rate limiting and throttling
- Advanced RBAC policies
- High-availability webhook deployment

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

## License

[MIT License](LICENSE)

## Acknowledgments

This project was developed with assistance from Claude AI (Anthropic). The design phase and implementation process are documented in exported conversation logs available in `.claude/exports/`.

---

**Note**: MCA is currently in Phase 1 (MITM Proxy). Multi-cluster routing capabilities are planned for Phase 2.
