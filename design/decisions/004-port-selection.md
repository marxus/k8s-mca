# ADR-004: Port Selection - 6443 for Kubernetes API Convention

**Date**: 2025-11-19  
**Status**: Accepted  
**Context**: Phase 1 MITM Proxy Implementation

## Problem

MCA proxy needs to listen on a specific port within the pod's network namespace that applications will connect to via `KUBERNETES_SERVICE_PORT`. The port selection affects:

1. **Compatibility** - Applications expect certain port conventions for K8s API servers
2. **Conflict Avoidance** - Must not interfere with other services in the pod
3. **Security** - Some ports have special meanings or security implications
4. **Debugging** - Should be intuitive for developers and operators to understand

## Decision  

MCA will listen on port **6443** (`127.0.0.1:6443`) and applications will be configured with `KUBERNETES_SERVICE_PORT=6443`.

## Rationale

### Kubernetes API Server Port Conventions
- **6443**: Standard Kubernetes API server port (HTTPS)
- **443**: Standard HTTPS port  
- **8443**: Alternative K8s API server port (often used in development)

Port 6443 is the de facto standard for Kubernetes API servers and will be immediately recognizable to developers and operators.

### Alternatives Considered:

#### Option 1: Port 443 (Rejected)
```bash
KUBERNETES_SERVICE_HOST=127.0.0.1
KUBERNETES_SERVICE_PORT=443
```
**Advantages:**
- Standard HTTPS port
- Matches real world API server deployments

**Disadvantages:**
- **Conflict Risk**: Port 443 might be used by other services in the pod
- **Common Usage**: Many application frameworks use 443 for their own HTTPS endpoints
- **Privilege Issues**: Port 443 may require privileged containers in some environments

#### Option 2: Port 8443 (Considered)  
```bash
KUBERNETES_SERVICE_HOST=127.0.0.1
KUBERNETES_SERVICE_PORT=8443
```
**Advantages:**
- Alternative K8s convention
- Less likely to conflict than 443

**Disadvantages:**
- Less standard than 6443
- Often associated with development/testing environments
- Not the primary K8s API server convention

#### Option 3: Random High Port like 9443 (Rejected)
```bash
KUBERNETES_SERVICE_HOST=127.0.0.1  
KUBERNETES_SERVICE_PORT=9443
```
**Advantages:**
- Very unlikely to conflict
- Clearly custom/proxy port

**Disadvantages:**
- **Not Intuitive**: Developers won't recognize it as K8s API related
- **Debugging Confusion**: May confuse operators during troubleshooting
- **Lost Convention**: Loses connection to Kubernetes API server standards

#### Option 4: Port 6443 (Selected)
```bash
KUBERNETES_SERVICE_HOST=127.0.0.1
KUBERNETES_SERVICE_PORT=6443
```
**Advantages:**
- **Standard Convention**: Primary Kubernetes API server port
- **Developer Recognition**: Immediately identifiable as K8s API
- **Debugging Friendly**: Operators understand 6443 = K8s API
- **Conflict Avoidance**: Less likely to be used by application services
- **Future Proof**: Aligns with multi-cluster API server conventions

## Implementation Details

### Environment Variable Configuration
```yaml
containers:
- name: application
  env:
  - name: KUBERNETES_SERVICE_HOST
    value: "127.0.0.1"
  - name: KUBERNETES_SERVICE_PORT
    value: "6443"
```

### MCA Proxy Server Configuration
```go
func (p *ProxyServer) Start() error {
    listener, err := tls.Listen("tcp", "127.0.0.1:6443", &tls.Config{
        Certificates: []tls.Certificate{p.serverCert},
    })
    if err != nil {
        return fmt.Errorf("failed to listen on port 6443: %w", err)
    }
    
    return p.server.Serve(listener)
}
```

### Client Connection Flow
```
Application K8s Client
         │
         ▼ (Connect to 127.0.0.1:6443)
    MCA Proxy Server  
         │
         ▼ (Forward to real API server)
    Real K8s API Server
    (cluster.local:6443)
```

## Port Conflict Analysis

### Likelihood of Port 6443 Usage by Applications:
- **Very Low**: Most applications don't run Kubernetes API servers
- **Database Ports**: Typically use different ranges (3306, 5432, 27017, etc.)
- **Web Applications**: Usually use 80, 443, 8080, 3000, etc.
- **Microservices**: Often use 8000-9000 range or random high ports

### Conflict Resolution Strategy:
If port 6443 is occupied by another service in the pod:
1. **Detection**: MCA startup will fail with "port already in use" error
2. **Configuration**: Allow port override via environment variable
3. **Documentation**: Clear error messages and troubleshooting guide
4. **Future**: Automatic port detection in advanced deployments

```go
func (p *ProxyServer) Start() error {
    port := os.Getenv("MCA_LISTEN_PORT")
    if port == "" {
        port = "6443"  // Default
    }
    
    addr := fmt.Sprintf("127.0.0.1:%s", port)
    // ... rest of implementation
}
```

## Security Considerations

### Port 6443 Security Profile:
- **Non-Privileged**: Ports > 1024 don't require root privileges
- **Standard Practice**: Following established K8s conventions
- **Network Isolation**: Only accessible within pod network namespace
- **TLS Required**: Always encrypted communication

### Attack Surface:
- **Pod-Scoped**: Only accessible from containers within the same pod
- **Authenticated**: Requires valid K8s API authentication (handled by MCA)
- **Authorized**: Subject to RBAC policies on the real API server

## Monitoring & Observability

### Port 6443 Benefits for Operations:
- **Log Recognition**: Operators immediately understand 6443 = K8s API
- **Network Monitoring**: Standard tools recognize 6443 as K8s API traffic
- **Alerting Rules**: Existing K8s monitoring can be extended to cover proxy
- **Debugging**: Network traces show familiar K8s API port patterns

### Metrics Labeling:
```go
var requestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "mca_requests_total",
        Help: "Total number of proxied requests",
    },
    []string{"port", "cluster", "resource"},
)

// Usage:
requestsTotal.WithLabelValues("6443", "current", "pods").Inc()
```

## Documentation & Communication

### Developer Experience:
```yaml
# Clear documentation example:
# MCA listens on port 6443 (standard Kubernetes API server port)
# Applications connect to 127.0.0.1:6443 instead of the real API server
env:
- name: KUBERNETES_SERVICE_HOST
  value: "127.0.0.1"
- name: KUBERNETES_SERVICE_PORT  
  value: "6443"  # Standard K8s API server port
```

### Troubleshooting Guides:
- Port conflicts: Check for other services using 6443
- Connection issues: Verify MCA is listening on 127.0.0.1:6443
- Certificate errors: Validate TLS setup on port 6443

## Future Considerations

### Phase 2 Multi-Cluster:
- Port 6443 maintains consistency across multiple target clusters
- Routing decisions can include port information for debugging
- Multiple clusters still funnel through single 6443 proxy port

### Phase 3 Observability:
- Standard port makes metrics correlation easier
- 6443 appears in dashboards as recognizable K8s API traffic
- Alerting rules can leverage port-based filtering

### Phase 4 Production:
- Service monitors can target port 6443 for metrics collection
- Network policies can reference standard K8s API port
- Load balancers and ingress can use familiar port conventions

## Consequences

### Positive:
- **Intuitive**: Developers immediately recognize 6443 as Kubernetes API
- **Standard**: Follows established Kubernetes port conventions  
- **Debuggable**: Network traces and logs are immediately understandable
- **Compatible**: Works with existing K8s tooling and monitoring

### Negative:
- **Potential Conflicts**: Rare possibility of port conflicts in complex pods
- **Fixed Convention**: Less flexibility than dynamic port assignment

### Mitigation:
- Clear error messages for port conflicts
- Environment variable override capability
- Comprehensive documentation for conflict resolution

This port selection provides the best balance of convention adherence, conflict avoidance, and operational clarity for a Kubernetes-focused proxy system.