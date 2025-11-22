# ADR-001: Sidecar Pattern with Network Namespace Sharing

**Date**: 2025-11-19  
**Status**: Accepted  
**Context**: Phase 1 MITM Proxy Implementation

## Problem

MCA needs to intercept all Kubernetes API calls from application containers. We need to choose an architectural pattern that allows transparent interception without modifying application code or requiring changes to existing Kubernetes clients.

## Decision

We will use the **Kubernetes sidecar pattern with network namespace sharing** and override environment variables to redirect API calls to localhost.

### Specific Implementation:
- Use `initContainers` with `restartPolicy: Always` (true sidecar pattern)
- MCA listens on `127.0.0.1:6443` within the shared pod network
- Override `KUBERNETES_SERVICE_HOST=127.0.0.1` and `KUBERNETES_SERVICE_PORT=6443` 
- Applications make API calls to localhost, unaware they're hitting a proxy

## Rationale

### Advantages:
1. **Zero Application Changes**: Applications continue using standard Kubernetes environment variables
2. **Language Agnostic**: Works with any Kubernetes client library (Go, Python, Java, etc.)
3. **Network Efficiency**: Localhost communication has minimal latency overhead
4. **Kubernetes Native**: Uses standard Kubernetes sidecar patterns and lifecycle management
5. **Transparent Operation**: Applications are unaware of the proxy layer

### Alternatives Considered:

#### 1. Network Proxy/iptables Redirection
- **Pros**: Completely transparent, no environment changes needed
- **Cons**: Requires privileged containers, complex iptables rules, harder to debug
- **Rejected**: Too complex and security concerns with privileged access

#### 2. Service Mesh Integration (Istio/Linkerd)
- **Pros**: Leverages existing infrastructure, advanced traffic management
- **Cons**: Requires service mesh deployment, adds complexity, not always available
- **Rejected**: Too heavy-weight for Phase 1, adds external dependencies

#### 3. Init Container + Volume Sharing
- **Pros**: Clean separation of concerns
- **Cons**: Applications would need to restart after MCA initialization
- **Rejected**: Breaks normal pod startup patterns, potential race conditions

#### 4. DaemonSet Proxy
- **Pros**: Centralized proxy per node
- **Cons**: Requires node-level networking, harder to isolate per-application
- **Rejected**: Less granular control, harder multi-tenancy

## Implementation Details

### True Sidecar Pattern
```yaml
spec:
  initContainers:
  - name: mca
    image: mca:latest
    restartPolicy: Always  # Key: makes init container run continuously
    # MCA proxy server configuration
    
  containers:
  - name: application
    env:
    - name: KUBERNETES_SERVICE_HOST
      value: "127.0.0.1"
    - name: KUBERNETES_SERVICE_PORT
      value: "6443"
```

### Network Architecture
- **Shared Network Namespace**: All containers in pod share same network stack
- **Localhost Communication**: Application → MCA via 127.0.0.1:6443
- **External Forwarding**: MCA → Real API Server via cluster networking

### Port Selection
- Port 6443 follows Kubernetes API server conventions
- Avoids conflicts with common application ports (80, 443, 8080, etc.)
- Familiar to Kubernetes developers and operators

## Consequences

### Positive:
- Simple implementation and debugging
- No privileged containers required
- Works with existing deployment patterns
- Easy to understand and troubleshoot

### Negative:
- Requires environment variable overrides in application containers
- Applications see modified Kubernetes service discovery
- Pod startup dependency (MCA must start first)

### Mitigations:
- Pod injection tool automates environment variable changes
- Health checks ensure MCA is ready before application starts
- Clear documentation for manual deployment scenarios

## Future Considerations

- Phase 2: Multi-cluster routing builds naturally on this foundation
- Phase 3: Observability can easily instrument the localhost proxy layer
- Phase 4: Admission webhook automates sidecar injection

This pattern provides a solid foundation that scales from simple MITM proxy to advanced multi-cluster routing without architectural changes.