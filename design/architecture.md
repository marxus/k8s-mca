# MCA Architecture Overview

## System Architecture

MCA (Multi Cluster Adapter) is designed as a Kubernetes sidecar proxy that intercepts and manages all API calls from application containers. The architecture follows a phased approach, building from basic MITM proxy capabilities to advanced multi-cluster routing and production deployment automation.

## Core Components

### Sidecar Proxy Pattern
- **Network Namespace Sharing**: MCA and application containers share the same network namespace
- **Environment Variable Override**: Applications connect to `127.0.0.1:6443` instead of real API server
- **True Sidecar**: Uses Kubernetes `initContainers` with `restartPolicy: Always`

### Certificate Management
- **Custom CA Generation**: MCA generates its own Certificate Authority at startup
- **Server Certificates**: Creates TLS certificates for localhost with proper SAN extensions
- **Trust Chain**: Applications trust MCA's custom CA for TLS verification

### ServiceAccount Isolation
- **Bypass Auto-mount**: Uses `automountServiceAccountToken: false` to control SA access
- **Dual SA Access**: MCA has real SA access, applications get fake SA with custom CA
- **Token Management**: MCA handles real authentication, applications use empty tokens

## Phase Architecture Evolution

### Phase 1: MITM Proxy Foundation
```
┌─────────────────────────────────────────┐
│                 Pod                     │
├─────────────────┬───────────────────────┤
│   Application   │        MCA            │
│   Container     │    (init sidecar)     │
│                 │                       │
│  ┌─────────────┐│  ┌─────────────────┐  │
│  │ K8s Client  ││  │  HTTPS Proxy    │  │
│  │             ││  │                 │  │
│  │ 127.0.0.1:  ││  │ Listen: :6443   │  │
│  │ 6443        ││  │ Forward: Real   │  │
│  │             ││  │ API Server      │  │
│  └─────────────┘│  └─────────────────┘  │
│                 │                       │
│  Fake SA        │   Real SA             │
│  + Custom CA    │   + Real Token        │
└─────────────────┴───────────────────────┘
                  │
                  ▼
         ┌─────────────────┐
         │ Kubernetes API  │
         │    Server       │
         └─────────────────┘
```

### Phase 2: Multi-Cluster Routing
- **Cluster Registry**: Configuration management for multiple target clusters
- **Routing Rules**: Namespace, resource-type, and label-based routing logic
- **Load Balancing**: Distribution across multiple clusters
- **Credential Management**: Per-cluster authentication handling

### Phase 3: Observability & Advanced Features
- **Metrics Collection**: Prometheus metrics for API usage
- **Request Tracing**: Distributed tracing for multi-cluster calls
- **Health Monitoring**: Target cluster availability checking
- **Performance Optimization**: Connection pooling and caching

### Phase 4: Production Deployment
- **Admission Webhook**: Automatic sidecar injection
- **Operator Pattern**: Advanced lifecycle management
- **Helm Charts**: Production deployment packaging
- **CI/CD Integration**: Automated testing and deployment

## Data Flow

### Request Flow (Phase 1)
1. **Application** makes K8s API call to `127.0.0.1:6443`
2. **MCA Proxy** receives HTTPS request with fake token
3. **Certificate Verification** succeeds using custom CA
4. **Token Replacement** replaces fake token with real SA token
5. **Request Forwarding** to real API server via `rest.InClusterConfig()`
6. **Response Proxying** back to application with inspection/logging

### Volume Architecture
```
Pod Volumes:
├── kube-api-access-sa (Projected)
│   ├── token (Real ServiceAccount token)
│   ├── ca.crt (Real cluster CA)
│   └── namespace
│   └── → Mounted to: MCA container:/var/run/secrets/kubernetes.io/serviceaccount/
│
└── kube-api-access-mca-sa (emptyDir)
    ├── ca.crt (Custom MCA CA)
    ├── token (Empty file)
    └── namespace (Copy from real SA)
    └── → Mounted to: Application container:/var/run/secrets/kubernetes.io/serviceaccount/
```

## Security Model

### Phase 1 Security Boundaries
- **Application Isolation**: Applications cannot access real cluster credentials
- **Certificate Trust**: Only MCA-generated certificates are trusted by applications
- **Token Separation**: Real tokens never exposed to application containers
- **Network Segmentation**: All external API calls flow through MCA proxy

### Authentication Flow
1. **MCA Authentication**: Uses real ServiceAccount token for upstream calls
2. **Application Authentication**: Provides empty/fake tokens (ignored by MCA)
3. **Token Forwarding**: MCA strips incoming tokens and adds real ones
4. **Authorization**: Real RBAC policies apply at the cluster level

## Deployment Model

### Container Configuration
```yaml
spec:
  automountServiceAccountToken: false
  
  initContainers:
  - name: mca
    image: mca:latest
    restartPolicy: Always
    # Real SA access + Certificate generation
    
  containers:
  - name: application
    # Fake SA access + Environment overrides
    env:
    - name: KUBERNETES_SERVICE_HOST
      value: "127.0.0.1"
    - name: KUBERNETES_SERVICE_PORT
      value: "6443"
```

### Network Architecture
- **Shared Network Namespace**: All containers in pod share network stack
- **Localhost Communication**: Application → MCA via 127.0.0.1:6443
- **External Communication**: MCA → Real API via cluster networking
- **Port Conventions**: 6443 follows Kubernetes API server standards

## Scalability Considerations

### Phase 1 Limitations
- **Single Cluster**: Only supports current cluster API calls
- **No Load Balancing**: Direct proxy to single API server
- **Limited Caching**: No request/response caching

### Future Scalability (Phase 2+)
- **Multi-Cluster Support**: Route to multiple target clusters
- **Connection Pooling**: Efficient connection reuse
- **Request Caching**: Reduce API server load
- **Horizontal Scaling**: Multiple MCA instances per application