# MCA API & Usage Documentation

**Last Updated**: 2025-12-03
**Version**: 1.0 (Phase 1)

## Table of Contents

- [Command Line Interface](#command-line-interface)
- [Pod Injection API](#pod-injection-api)
- [Webhook API](#webhook-api)
- [Configuration API](#configuration-api)
- [Package APIs](#package-apis)
- [Usage Examples](#usage-examples)

## Command Line Interface

### Binary Modes

MCA supports three operational modes via command-line flags:

```bash
# Proxy mode (default)
./mca --proxy

# Injection mode
./mca --inject < pod.yaml > injected-pod.yaml

# Webhook mode
./mca --webhook
```

### Mode Descriptions

| Mode | Flag | Description | Use Case |
|------|------|-------------|----------|
| Proxy | `--proxy` | Runs HTTPS reverse proxy server | Production pods with injected sidecar |
| Inject | `--inject` | Reads YAML from stdin, outputs mutated YAML | Development, CI/CD pipelines |
| Webhook | `--webhook` | Runs admission webhook HTTP server | Production cluster-wide injection |

## Pod Injection API

### Function Signatures

#### CLI Injection

```go
package inject

// ViaCLI processes pod YAML from bytes and returns mutated YAML
func ViaCLI(podYAML []byte) ([]byte, error)
```

**Parameters**:
- `podYAML`: Raw YAML bytes of a Kubernetes Pod manifest

**Returns**:
- `[]byte`: Mutated pod YAML with MCA sidecar injected
- `error`: Non-nil if unmarshaling, injection, or marshaling fails

**Example**:
```go
import "github.com/marxus/k8s-mca/pkg/inject"

podYAML := []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  containers:
  - name: app
    image: nginx
`)

mutatedYAML, err := inject.ViaCLI(podYAML)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(mutatedYAML))
```

#### Webhook Injection

```go
package inject

import corev1 "k8s.io/api/core/v1"

// ViaWebhook processes a pod object and returns mutated pod
func ViaWebhook(pod corev1.Pod) (corev1.Pod, error)
```

**Parameters**:
- `pod`: Kubernetes Pod object (from `k8s.io/api/core/v1`)

**Returns**:
- `corev1.Pod`: Mutated pod object with MCA sidecar injected
- `error`: Non-nil if injection logic fails

**Example**:
```go
import (
    "github.com/marxus/k8s-mca/pkg/inject"
    corev1 "k8s.io/api/core/v1"
)

var pod corev1.Pod
// ... populate pod from admission request

mutatedPod, err := inject.ViaWebhook(pod)
if err != nil {
    return err
}

// ... generate patch and return admission response
```

### Injection Behavior

#### What Gets Injected

1. **Init Container**: `mca-proxy` prepended as first init container
   ```yaml
   initContainers:
   - name: mca-proxy
     image: <from conf.ProxyImage>
     restartPolicy: Always
     args: [--proxy]
     securityContext:
       runAsNonRoot: true
       runAsUser: 999
     volumeMounts:
     - name: kube-api-access-sa
       mountPath: /var/run/secrets/kubernetes.io/serviceaccount
       readOnly: true
     - name: kube-api-access-mca-sa
       mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
   ```

2. **Environment Variables** (added to ALL non-MCA containers):
   ```yaml
   env:
   - name: KUBERNETES_SERVICE_HOST
     value: "127.0.0.1"
   - name: KUBERNETES_SERVICE_PORT
     value: "6443"
   ```

3. **Volume Mounts** (added to ALL non-MCA containers):
   ```yaml
   volumeMounts:
   - name: kube-api-access-mca-sa
     mountPath: /var/run/secrets/kubernetes.io/serviceaccount
     readOnly: true
   ```

4. **Volumes** (ensured to exist):
   ```yaml
   volumes:
   - name: kube-api-access-sa
     projected:
       sources:
       - serviceAccountToken:
           path: token
           expirationSeconds: 3607
       - configMap:
           name: kube-root-ca.crt
           items:
           - key: ca.crt
             path: ca.crt
       - downwardAPI:
           items:
           - path: namespace
             fieldRef:
               fieldPath: metadata.namespace
   - name: kube-api-access-mca-sa
     emptyDir: {}
   ```

5. **Pod Spec Changes**:
   ```yaml
   spec:
     automountServiceAccountToken: false
   ```

#### Idempotency

The injection is **idempotent** - running it multiple times on the same manifest produces the same result:

1. Existing `mca-proxy` init containers are removed before adding new one
2. Environment variables are updated if they exist, added if they don't
3. Volume mounts are filtered by mount path to avoid duplicates
4. Volumes are filtered by name to avoid duplicates

## Webhook API

### HTTP Endpoints

#### Health Check

```
GET /health
```

**Response**:
```json
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

#### Mutate Pods

```
POST /mutate
Content-Type: application/json
```

**Request Body** (Kubernetes `AdmissionReview`):
```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": {
    "uid": "12345-67890",
    "kind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "resource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "object": {
      "metadata": {
        "name": "test-pod",
        "labels": {
          "mca.k8s.io/inject": "true"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "app",
            "image": "nginx"
          }
        ]
      }
    }
  }
}
```

**Response Body** (Kubernetes `AdmissionReview`):
```json
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "response": {
    "uid": "12345-67890",
    "allowed": true,
    "patchType": "JSONPatch",
    "patch": "<base64-encoded JSON patch>"
  }
}
```

**JSON Patch Content** (decoded):
```json
[
  {
    "op": "replace",
    "path": "/spec",
    "value": {
      "automountServiceAccountToken": false,
      "initContainers": [
        {
          "name": "mca-proxy",
          "image": "mca:latest",
          "restartPolicy": "Always",
          ...
        }
      ],
      "containers": [
        {
          "name": "app",
          "image": "nginx",
          "env": [
            {"name": "KUBERNETES_SERVICE_HOST", "value": "127.0.0.1"},
            {"name": "KUBERNETES_SERVICE_PORT", "value": "6443"}
          ],
          ...
        }
      ],
      ...
    }
  }
]
```

### Webhook Configuration

The webhook is configured via `MutatingWebhookConfiguration`:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mca-webhook
webhooks:
  - name: mca-webhook.k8s.io
    clientConfig:
      service:
        name: mca-webhook
        namespace: default
        path: /mutate
      caBundle: <base64-encoded CA certificate>  # Auto-patched by webhook at startup
    rules:
      - operations: [CREATE]
        apiGroups: [""]
        apiVersions: [v1]
        resources: [pods]
    objectSelector:
      matchLabels:
        mca.k8s.io/inject: "true"  # Only pods with this label
    admissionReviewVersions: [v1, v1beta1]
    sideEffects: None
    failurePolicy: Fail
    reinvocationPolicy: IfNeeded
```

**Key Features**:
- **Label Selector**: Only triggers on pods labeled `mca.k8s.io/inject=true`
- **Failure Policy**: `Fail` - pod creation blocked if webhook fails
- **Reinvocation Policy**: `IfNeeded` - allows multiple mutations if needed
- **Auto-Patching**: Webhook patches its own `caBundle` at startup with generated CA

## Configuration API

### Build-Time Configuration

Configuration is selected at **compile time** via build tags:

```bash
# Production build
go build -tags release -o mca cmd/mca/main.go

# Development build (default)
go build -o mca cmd/mca/main.go

# Testing build (automatic in tests)
go test ./...
```

### Configuration Variables

#### Release Configuration

```go
package conf

// Filesystem access (real OS filesystem)
var FS afero.Fs = afero.NewOsFs()

// Kubernetes authentication (in-cluster config)
var InClusterConfig = rest.InClusterConfig

// Proxy server binding address
var ProxyServerAddr = "127.0.0.1:6443"

// Certificate IP addresses for SAN
var ProxyCertIPAddresses = []net.IP{
    net.IPv4(127, 0, 0, 1),  // 127.0.0.1
    net.IPv6loopback,         // ::1
}

// Proxy image (from environment)
var ProxyImage = os.Getenv("MCA_PROXY_IMAGE")

// Webhook name (from environment)
var WebhookName = os.Getenv("MCA_WEBHOOK_NAME")
```

#### Development Configuration

```go
package conf

// Filesystem access (sandboxed to ./tmp/)
var FS afero.Fs  // Set in initDevelop() to afero.NewBasePathFs(osfs, "./tmp")

// Kubernetes authentication (local kubeconfig)
var InClusterConfig = clientcmd.ClientConfig  // Uses MCA_K8S_CTX env var or "mca-k8s-ctx"

// Proxy server binding address (external access)
var ProxyServerAddr = "0.0.0.0:6443"

// Certificate IP addresses for SAN (custom dev IP)
var ProxyCertIPAddresses = []net.IP{
    net.IPv4(192, 168, 5, 2),
}

// Proxy image (hardcoded)
var ProxyImage = "mca:latest"

// Webhook name (hardcoded)
var WebhookName = "mca-webhook"
```

### Environment Variables

#### Production (Release Build)

| Variable | Purpose | Example |
|----------|---------|---------|
| `MCA_PROXY_IMAGE` | Image to inject as proxy sidecar | `ghcr.io/marxus/k8s-mca:v0.1.0` |
| `MCA_WEBHOOK_NAME` | Name of webhook configuration | `mca-webhook` |

#### Development

| Variable | Purpose | Example |
|----------|---------|---------|
| `MCA_K8S_CTX` | Kubeconfig context to use | `minikube`, `kind-cluster` |

## Package APIs

### pkg/certs

```go
package certs

// GenerateCAAndTLSCert generates a CA certificate and a TLS server certificate
// signed by that CA with the specified DNS names and IP addresses in SANs.
//
// Returns:
//   - tls.Certificate: Server certificate usable with net/http TLS listener
//   - []byte: PEM-encoded CA certificate for client trust
//   - error: Non-nil if generation fails
func GenerateCAAndTLSCert(dnsNames []string, ipAddresses []net.IP) (tls.Certificate, []byte, error)
```

**Example**:
```go
import "github.com/marxus/k8s-mca/pkg/certs"

dnsNames := []string{"localhost", "mca-webhook", "mca-webhook.default.svc"}
ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}

serverCert, caCertPEM, err := certs.GenerateCAAndTLSCert(dnsNames, ipAddresses)
if err != nil {
    log.Fatal(err)
}

// Use serverCert for TLS listener
server := &http.Server{
    Addr: ":8443",
    TLSConfig: &tls.Config{
        Certificates: []tls.Certificate{serverCert},
    },
}

// Write caCertPEM to file for client trust
err = os.WriteFile("/tmp/ca.crt", caCertPEM, 0644)
```

### pkg/proxy

```go
package proxy

// Server represents an HTTPS reverse proxy server
type Server struct {
    tlsCert tls.Certificate
}

// NewServer creates a new proxy server with the given TLS certificate
func NewServer(tlsCert tls.Certificate) *Server

// Start starts the proxy server and blocks until error or shutdown.
// Discovers Kubernetes API endpoint and forwards all requests.
func (s *Server) Start() error
```

**Example**:
```go
import (
    "github.com/marxus/k8s-mca/pkg/certs"
    "github.com/marxus/k8s-mca/pkg/proxy"
    "github.com/marxus/k8s-mca/conf"
)

// Generate certificates
dnsNames := []string{"localhost"}
serverCert, _, err := certs.GenerateCAAndTLSCert(dnsNames, conf.ProxyCertIPAddresses)
if err != nil {
    log.Fatal(err)
}

// Create and start proxy
proxyServer := proxy.NewServer(serverCert)
log.Fatal(proxyServer.Start())  // Blocks
```

### pkg/webhook

```go
package webhook

// Server represents an admission webhook HTTP server
type Server struct {
    tlsCert tls.Certificate
}

// NewServer creates a new webhook server with the given TLS certificate
func NewServer(tlsCert tls.Certificate) *Server

// Start starts the webhook server on :8443 and blocks
func (s *Server) Start() error
```

**Example**:
```go
import (
    "github.com/marxus/k8s-mca/pkg/certs"
    "github.com/marxus/k8s-mca/pkg/webhook"
)

// Generate certificates
dnsNames := []string{"mca-webhook", "mca-webhook.default.svc"}
serverCert, _, err := certs.GenerateCAAndTLSCert(dnsNames, nil)
if err != nil {
    log.Fatal(err)
}

// Create and start webhook
webhookServer := webhook.NewServer(serverCert)
log.Fatal(webhookServer.Start())  // Blocks on :8443
```

### pkg/serve

```go
package serve

// StartProxy orchestrates proxy startup:
// 1. Generates TLS certificates
// 2. Writes CA and namespace files to volumes
// 3. Starts proxy server
func StartProxy() error

// StartWebhook orchestrates webhook startup:
// 1. Generates TLS certificates
// 2. Patches MutatingWebhookConfiguration with CA bundle
// 3. Starts webhook server
func StartWebhook() error
```

**Example**:
```go
import "github.com/marxus/k8s-mca/pkg/serve"

// In production pod with --proxy flag
if err := serve.StartProxy(); err != nil {
    log.Fatal(err)
}

// In webhook pod with --webhook flag
if err := serve.StartWebhook(); err != nil {
    log.Fatal(err)
}
```

## Usage Examples

### Example 1: Manual CLI Injection

```bash
# Original pod manifest
cat > pod.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: nginx-test
spec:
  containers:
  - name: nginx
    image: nginx:latest
EOF

# Inject MCA sidecar
./mca --inject < pod.yaml > pod-with-mca.yaml

# Apply to cluster
kubectl apply -f pod-with-mca.yaml

# Verify injection
kubectl get pod nginx-test -o jsonpath='{.spec.initContainers[0].name}'
# Output: mca-proxy
```

### Example 2: Webhook Injection

```bash
# Deploy webhook
helm install mca ./charts/mca \
  --set image.repository=ghcr.io/marxus/k8s-mca \
  --set image.tag=v0.1.0

# Create pod with label
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: auto-inject-test
  labels:
    mca.k8s.io/inject: "true"
spec:
  containers:
  - name: nginx
    image: nginx:latest
EOF

# Verify automatic injection
kubectl get pod auto-inject-test -o jsonpath='{.spec.initContainers[0].name}'
# Output: mca-proxy
```

### Example 3: Local Development with Proxy

```bash
# Terminal 1: Run proxy
export MCA_K8S_CTX=minikube
go run cmd/mca/main.go --proxy

# Terminal 2: Test with kubectl through proxy
./kubectl.sh get pods -A

# kubectl.sh uses volume mounts and environment overrides:
# - Mounts ./tmp/var/run/secrets/kubernetes.io/mca-serviceaccount
# - Sets KUBERNETES_SERVICE_HOST=192.168.5.2
# - Sets KUBERNETES_SERVICE_PORT=6443
```

### Example 4: Programmatic Injection

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/marxus/k8s-mca/pkg/inject"
)

func main() {
    // Read pod YAML from file
    podYAML, err := os.ReadFile("pod.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Inject MCA sidecar
    mutatedYAML, err := inject.ViaCLI(podYAML)
    if err != nil {
        log.Fatal(err)
    }

    // Write to stdout or file
    fmt.Println(string(mutatedYAML))
}
```

### Example 5: Testing with Sandboxed Filesystem

```go
package mypackage

import (
    "testing"

    "github.com/marxus/k8s-mca/conf"
    "github.com/spf13/afero"
)

func TestWithMockFS(t *testing.T) {
    // In test builds, conf.FS is automatically an in-memory filesystem

    // Write mock files
    afero.WriteFile(conf.FS, "/tmp/test.txt", []byte("content"), 0644)

    // Read back
    data, err := afero.ReadFile(conf.FS, "/tmp/test.txt")
    if err != nil {
        t.Fatal(err)
    }

    if string(data) != "content" {
        t.Errorf("Expected 'content', got %q", data)
    }

    // No side effects on real filesystem!
}
```

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to unmarshal pod` | Invalid YAML input | Check YAML syntax |
| `failed to create MCA container` | Internal injection error | Check logs, report bug |
| `failed to generate certificates` | Crypto library error | Check system crypto support |
| `failed to get Kubernetes config` | Missing kubeconfig or in-cluster config | Set MCA_K8S_CTX or deploy in-cluster |
| `failed to apply mutating webhook` | RBAC permissions | Check ServiceAccount has patch permissions |

### Error Response Handling

```go
mutatedYAML, err := inject.ViaCLI(podYAML)
if err != nil {
    if strings.Contains(err.Error(), "unmarshal") {
        log.Println("Invalid YAML input")
    } else if strings.Contains(err.Error(), "marshal") {
        log.Println("Failed to generate output YAML")
    } else {
        log.Printf("Injection error: %v", err)
    }
    return err
}
```

## Performance Considerations

### Injection Performance

- **CLI Mode**: ~5ms per manifest (parsing + mutation + serialization)
- **Webhook Mode**: ~10-20ms per request (includes network + admission overhead)

### Memory Usage

- **Proxy Container**: ~50MB resident memory
- **Webhook Pod**: ~30MB resident memory

### Throughput

- **Webhook**: >500 requests/second (single replica)
- **Proxy**: >10,000 API requests/second (localhost communication)

## Best Practices

1. **Use Webhook in Production**: Automatic, consistent, policy-enforced
2. **Use CLI in Development**: Fast iteration, no cluster dependencies
3. **Label Pods Explicitly**: Don't rely on namespace-wide injection
4. **Test Injection Idempotency**: Run injection multiple times to verify
5. **Monitor Webhook Health**: Use `/health` endpoint for liveness/readiness
6. **Version Your Images**: Always use specific image tags, not `latest`
7. **Configure RBAC Carefully**: Webhook needs patch permission for MutatingWebhookConfiguration

## Troubleshooting

### Pod Creation Fails

```bash
# Check webhook logs
kubectl logs -l app=mca-webhook

# Check webhook configuration
kubectl get mutatingwebhookconfiguration mca-webhook -o yaml

# Check CA bundle is set
kubectl get mutatingwebhookconfiguration mca-webhook \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d
```

### Proxy Not Working

```bash
# Check proxy logs in pod
kubectl logs <pod-name> -c mca-proxy

# Verify environment variables
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].env[?(@.name=="KUBERNETES_SERVICE_HOST")]}'

# Verify volumes
kubectl get pod <pod-name> -o jsonpath='{.spec.volumes}'
```

### CLI Injection Issues

```bash
# Validate YAML before injection
kubectl --dry-run=client -f pod.yaml

# Test injection
./mca --inject < pod.yaml | kubectl --dry-run=client -f -

# Check for multiple init containers
./mca --inject < pod.yaml | yq '.spec.initContainers[] | .name'
```

---

For more information, see:
- [Architecture Documentation](architecture.md)
- [Deployment Guide](DEPLOYMENT.md)
- [GitHub Repository](https://github.com/marxus/k8s-mca)
