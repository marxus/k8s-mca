# Phase 1: MITM Proxy Implementation

**Status**: FULLY DETAILED - Ready for Implementation

## Overview

Phase 1 establishes the foundation of MCA as a Man-in-the-Middle HTTPS proxy that intercepts all Kubernetes API calls from application containers. This phase focuses on proving the core sidecar pattern and certificate management approach.

## Goals

- ✅ Intercept all K8s API calls from applications using sidecar pattern
- ✅ Implement custom certificate generation and TLS management
- ✅ Create ServiceAccount bypass mechanism
- ✅ Build Pod manifest injection tool for testing
- ✅ Establish comprehensive testing framework
- ✅ Provide request inspection and logging capabilities

## Technical Specification

### 1. Binary Architecture

#### Single Binary with Mode Selection
```bash
# Default: Run HTTPS proxy server
./mca

# Pod manifest injection mode
./mca --inject < pod.yaml > pod-with-mca.yaml
```

#### Command Line Interface
```go
type Config struct {
    // Proxy mode settings
    ListenAddr    string // Default: "127.0.0.1:6443"
    LogLevel      string // Default: "info"
    
    // Injection mode settings
    InjectMode    bool   // --inject flag
}
```

### 2. Certificate Management

#### Requirements
- Generate custom Certificate Authority at startup
- Create server certificate with proper SAN extensions
- Store certificates in shared volume for application access

#### SAN Extensions Required
```
Subject Alternative Names:
- DNS: localhost
- IP: 127.0.0.1
- IP: ::1 (IPv6 localhost)
```

#### Implementation Details
```go
type CertificateManager struct {
    CACert     *x509.Certificate
    CAKey      *rsa.PrivateKey
    ServerCert tls.Certificate
}

func (cm *CertificateManager) GenerateCA() error
func (cm *CertificateManager) GenerateServerCert() error
func (cm *CertificateManager) SaveCertificates(outputDir string) error
```

### 3. HTTPS Proxy Server

#### Core Functionality
- Listen on `127.0.0.1:6443` with custom TLS certificates
- Discover real API endpoint using `rest.InClusterConfig()`
- Forward all requests to real Kubernetes API server
- Replace fake tokens with real ServiceAccount tokens

#### Request Processing Pipeline
```go
type ProxyServer struct {
    realAPIEndpoint string
    realToken       []byte
    serverCert      tls.Certificate
    middleware      []Middleware
}

type Middleware func(http.Handler) http.Handler

// Middleware functions:
- TokenReplacementMiddleware() // Replace fake tokens with real ones
- LoggingMiddleware()          // Log all API requests/responses
- InspectionMiddleware()       // Parse and analyze K8s API calls
```

#### API Endpoint Discovery
```go
func discoverKubernetesAPI() (string, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        return "", err
    }
    return config.Host, nil
}
```

### 4. ServiceAccount Management

#### ServiceAccount Bypass Strategy
```yaml
spec:
  automountServiceAccountToken: false  # Disable automatic mounting
  
  volumes:
  - name: kube-api-access-sa          # Real SA for MCA
    projected:
      sources:
      - serviceAccountToken:
          path: token
      - configMap:
          name: kube-root-ca.crt
          items: [...]
      - downwardAPI:
          items: [...]
  
  - name: kube-api-access-mca-sa      # Fake SA for application
    emptyDir: {}
```

#### Fake ServiceAccount Creation
```go
type ServiceAccountManager struct {
    realSAPath  string // "/var/run/secrets/kubernetes.io/serviceaccount"
    fakeSAPath  string // "/var/run/secrets/kubernetes.io/mca-serviceaccount"
}

func (sam *ServiceAccountManager) CreateFakeServiceAccount() error {
    // Create fake SA directory structure:
    // - ca.crt (custom CA certificate)
    // - token (empty file)
    // - namespace (copied from real SA)
}
```

### 5. Pod Manifest Injection

#### Injection Requirements
- **Idempotent**: Multiple runs produce same result
- **Container-Specific**: Different treatment for MCA vs application containers
- **Comprehensive**: Handle all containers and init containers

#### Pod Mutation Logic
```go
type PodInjector struct {
    mcaImage string // "mca:latest"
}

func (pi *PodInjector) InjectMCASidecar(pod *v1.Pod) error {
    // 1. Set automountServiceAccountToken: false
    // 2. Ensure MCA is first init container (remove duplicates)
    // 3. Add environment overrides to non-MCA containers
    // 4. Add volume mounts to non-MCA containers
    // 5. Ensure required volumes exist
}

func (pi *PodInjector) addEnvironmentOverrides(container *v1.Container) {
    envVars := []v1.EnvVar{
        {Name: "KUBERNETES_SERVICE_HOST", Value: "127.0.0.1"},
        {Name: "KUBERNETES_SERVICE_PORT", Value: "6443"},
    }
    // Update or add environment variables (idempotent)
}
```

#### MCA Init Container Specification
```yaml
initContainers:
- name: mca
  image: mca:latest
  restartPolicy: Always
  volumeMounts:
  - name: kube-api-access-sa
    mountPath: /var/run/secrets/kubernetes.io/serviceaccount
    readOnly: true
  - name: kube-api-access-mca-sa
    mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
```

### 6. Container Lifecycle Management

#### Startup Sequence
1. **MCA Init Container Starts**: 
   - Generates certificates and stores in shared volume
   - Creates fake ServiceAccount directory
   - Starts HTTPS proxy server on `127.0.0.1:6443`

2. **Application Container Starts**:
   - Reads custom CA from fake SA directory
   - Uses overridden environment variables
   - Makes API calls to `127.0.0.1:6443`

#### Volume Sharing
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

## Implementation Plan

### Package Structure
```
cmd/
└── mca/
    └── main.go              # Binary entry point with flag handling

pkg/
├── certs/
│   ├── generator.go         # Certificate generation
│   └── generator_test.go    # Certificate tests
├── inject/
│   ├── injector.go          # Pod manifest injection
│   └── injector_test.go     # Injection tests
├── proxy/
│   ├── server.go            # HTTPS reverse proxy
│   ├── middleware.go        # Request processing middleware
│   └── server_test.go       # Proxy tests
└── sa/
    ├── manager.go           # ServiceAccount management
    └── manager_test.go      # SA tests

tmp/                         # Testing workspace
├── certs/                   # Generated test certificates
├── serviceaccount/          # Mock SA files
├── test-pods/               # Sample Pod manifests
└── mca-serviceaccount/      # Fake SA directory
```

### Testing Strategy

#### Unit Tests
- **Certificate Generation**: Validate CA and server cert generation, SAN extensions
- **Pod Injection**: Test idempotent mutations, container-specific handling
- **Proxy Logic**: Mock HTTP requests/responses, token replacement
- **SA Management**: File system operations using `tmp/` directory

#### Integration Tests
- **End-to-End Flow**: Application → MCA → Mock API Server
- **Certificate Chain**: Validate full TLS handshake
- **Environment Isolation**: Multiple test scenarios in `tmp/` subdirectories

#### Test Execution
```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Test specific package
go test ./pkg/certs/

# Integration tests
go test -tags integration ./...
```

## Configuration

### Environment Variables (MCA Container)
```bash
# Proxy configuration
MCA_LISTEN_ADDR=127.0.0.1:6443
MCA_LOG_LEVEL=info

# Certificate paths
MCA_CERT_DIR=/var/run/secrets/kubernetes.io/mca-serviceaccount

# Real ServiceAccount paths (auto-detected)
MCA_REAL_SA_PATH=/var/run/secrets/kubernetes.io/serviceaccount
```

### Environment Variables (Application Containers)
```bash
# Injected by Pod mutation
KUBERNETES_SERVICE_HOST=127.0.0.1
KUBERNETES_SERVICE_PORT=6443
```

## Success Criteria

### Functional Requirements
- ✅ Applications can make K8s API calls through MCA proxy
- ✅ All API requests are logged and inspectable
- ✅ Real cluster credentials are isolated from applications
- ✅ Pod injection tool produces valid Kubernetes manifests
- ✅ Certificate chain validation succeeds for all clients

### Performance Requirements
- ✅ Proxy latency < 10ms for typical API calls
- ✅ Memory usage < 100MB for MCA container
- ✅ Certificate generation < 1 second at startup

### Security Requirements
- ✅ Applications cannot access real ServiceAccount tokens
- ✅ Custom CA certificates are properly validated
- ✅ TLS communication between application and MCA
- ✅ No credentials logged or exposed in debug output

## Deployment Example

### Complete Pod Manifest
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-app-with-mca
spec:
  automountServiceAccountToken: false
  
  initContainers:
  - name: mca
    image: mca:latest
    restartPolicy: Always
    volumeMounts:
    - name: kube-api-access-sa
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      readOnly: true
    - name: kube-api-access-mca-sa
      mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
      
  containers:
  - name: test-app
    image: test-app:latest
    env:
    - name: KUBERNETES_SERVICE_HOST
      value: "127.0.0.1"
    - name: KUBERNETES_SERVICE_PORT
      value: "6443"
    volumeMounts:
    - name: kube-api-access-mca-sa
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      readOnly: true
      
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

## Next Steps

Upon completion of Phase 1:
1. **Validate with Real Applications**: Test with actual Kubernetes clients
2. **Performance Benchmarking**: Measure proxy overhead and optimization opportunities
3. **Security Audit**: Review certificate management and token handling
4. **Phase 2 Planning**: Begin detailed design for multi-cluster routing capabilities