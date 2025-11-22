# ADR-003: ServiceAccount Bypass with Manual Volume Management

**Date**: 2025-11-19  
**Status**: Accepted  
**Context**: Phase 1 MITM Proxy Implementation

## Problem

Kubernetes automatically mounts ServiceAccount credentials to `/var/run/secrets/kubernetes.io/serviceaccount/` in all containers. This creates a conflict for our MITM proxy approach:

1. **Applications** need a custom CA certificate for trusting MCA's TLS proxy
2. **MCA** needs real ServiceAccount credentials for authenticating with the actual API server  
3. **Kubernetes clients** expect ServiceAccount files at standard paths
4. **Security** requires isolation between real and fake credentials

## Decision

We will use **ServiceAccount bypass with manual volume management** to create separate credential contexts for MCA and application containers.

### Specific Implementation:
- Set `automountServiceAccountToken: false` to disable automatic mounting
- Create manual projected volume for MCA with real ServiceAccount credentials
- Create emptyDir volume for fake ServiceAccount structure used by applications
- MCA populates fake ServiceAccount with custom CA and empty token

## Rationale

### The Core Problem
Standard Kubernetes behavior creates this conflict:
```
/var/run/secrets/kubernetes.io/serviceaccount/
├── ca.crt      # Real cluster CA (needed for MCA → API server)
├── token       # Real SA token (needed for MCA → API server)  
└── namespace   # Pod namespace info

BUT applications need:
├── ca.crt      # Custom MCA CA (for app → MCA trust)
├── token       # Empty/fake token (MCA handles real auth)
└── namespace   # Same namespace info
```

### Alternatives Considered:

#### 1. Override Real ServiceAccount Mount (Rejected)
```yaml
# This doesn't work - ca.crt is read-only projected volume
volumeMounts:
- name: serviceaccount
  mountPath: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
  subPath: custom-ca.crt  # Cannot override projected volume files
```
**Problem**: Projected volumes are immutable and managed by kubelet

#### 2. CA Bundle Concatenation (Rejected)
- Combine real cluster CA + custom MCA CA into single file
- **Problem**: Applications would trust both CAs (security issue)
- **Problem**: Complex certificate chain management
- **Problem**: MCA still needs original cluster CA for upstream

#### 3. System-wide CA Store (Rejected)
- Install custom CA in `/etc/ssl/certs/` or similar
- **Problem**: Not all client libraries use system CA store
- **Problem**: Varies by base image (Alpine, Ubuntu, etc.)
- **Problem**: Requires container image modifications

#### 4. Environment Variable Override (Rejected)
- Use `SSL_CERT_FILE` or `SSL_CERT_DIR` environment variables
- **Problem**: Not respected by most Kubernetes client libraries
- **Problem**: Kubernetes clients hardcode ServiceAccount path

#### 5. ServiceAccount Bypass with Manual Volumes (Selected)
- Complete control over ServiceAccount credential distribution
- Clean separation between real and fake credentials
- Standard Kubernetes volume management patterns

## Implementation Details

### Pod Configuration
```yaml
spec:
  automountServiceAccountToken: false  # Critical: disables automatic mounting
  
  initContainers:
  - name: mca
    volumeMounts:
    - name: kube-api-access-sa                    # Real ServiceAccount
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      readOnly: true
    - name: kube-api-access-mca-sa                # Fake ServiceAccount  
      mountPath: /var/run/secrets/kubernetes.io/mca-serviceaccount
      
  containers:
  - name: application  
    volumeMounts:
    - name: kube-api-access-mca-sa                # Fake ServiceAccount
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount  
      readOnly: true
      
  volumes:
  - name: kube-api-access-sa          # Manual real ServiceAccount
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
              
  - name: kube-api-access-mca-sa      # Fake ServiceAccount structure
    emptyDir: {}
```

### Credential Flow Architecture
```
┌─────────────────┐    ┌─────────────────┐
│  Real SA Volume │    │  Fake SA Volume │
│ (Projected)     │    │ (emptyDir)      │  
├─────────────────┤    ├─────────────────┤
│ ca.crt (real)   │    │ ca.crt (custom) │
│ token (real)    │    │ token (empty)   │
│ namespace       │    │ namespace       │
└─────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│  MCA Container  │    │ App Container   │
│                 │    │                 │
│ Reads real SA   │    │ Reads fake SA   │
│ ↓               │    │ ↓               │  
│ Authenticates   │    │ Connects to     │
│ with real API   │    │ MCA proxy with  │
│ server          │    │ custom CA       │
└─────────────────┘    └─────────────────┘
```

### MCA ServiceAccount Setup Logic
```go  
type ServiceAccountManager struct {
    realSAPath  string // "/var/run/secrets/kubernetes.io/serviceaccount"
    fakeSAPath  string // "/var/run/secrets/kubernetes.io/mca-serviceaccount"
}

func (sam *ServiceAccountManager) CreateFakeServiceAccount() error {
    // 1. Read namespace from real SA
    namespace, err := sam.readNamespace()
    
    // 2. Write custom CA certificate 
    err = sam.writeFakeCA()
    
    // 3. Write empty token file
    err = sam.writeEmptyToken()
    
    // 4. Copy namespace file
    err = sam.copyNamespace(namespace)
    
    return nil
}
```

## Security Model

### Credential Isolation
- **MCA Container**: Full access to real cluster credentials
  - Can authenticate with real API server
  - Can perform any operation allowed by ServiceAccount RBAC
  - Isolated from application containers

- **Application Containers**: Only fake credentials  
  - Cannot access real cluster credentials
  - Must go through MCA proxy for all API calls
  - MCA controls all actual API authentication

### Authentication Flow
1. **Application** → Makes API call with empty/fake token
2. **MCA Proxy** → Strips fake token, adds real token from its ServiceAccount  
3. **API Server** → Validates real token, applies RBAC policies
4. **Response** → Flows back through MCA to application

### Token Management Benefits
- **No Token Copying**: Avoids complex token synchronization
- **No Token Renewal**: MCA reads fresh tokens directly from projected volume
- **Credential Rotation**: Handled automatically by Kubernetes for real SA
- **Security Isolation**: Applications never see real credentials

## Technical Challenges & Solutions

### Challenge 1: Volume Mount Conflicts
Cannot mount different volumes to same path in different containers.

**Solution**: Use different mount paths and create fake SA structure in shared volume.

### Challenge 2: Token Expiration Handling
Projected ServiceAccount tokens rotate automatically.

**Solution**: MCA reads real token fresh from its volume for each request, no caching.

### Challenge 3: Timing Dependencies
Application containers must wait for MCA to create fake SA structure.

**Solution**: True sidecar pattern ensures MCA runs first and creates structure at startup.

### Challenge 4: Container Image Compatibility
Some images might expect specific ServiceAccount file permissions or ownership.

**Solution**: Create fake SA files with standard permissions (644) and proper ownership.

## Implementation Verification

### Testing Strategy
```go
func TestServiceAccountBypass(t *testing.T) {
    // Create temporary directory structure
    tmpDir := t.TempDir()
    realSAPath := filepath.Join(tmpDir, "real-sa")  
    fakeSAPath := filepath.Join(tmpDir, "fake-sa")
    
    // Setup real SA structure
    setupRealServiceAccount(realSAPath)
    
    // Test fake SA creation
    sam := &ServiceAccountManager{
        realSAPath: realSAPath,
        fakeSAPath: fakeSAPath,  
    }
    
    err := sam.CreateFakeServiceAccount()
    require.NoError(t, err)
    
    // Verify fake SA structure
    assert.FileExists(t, filepath.Join(fakeSAPath, "ca.crt"))
    assert.FileExists(t, filepath.Join(fakeSAPath, "token"))
    assert.FileExists(t, filepath.Join(fakeSAPath, "namespace"))
    
    // Verify ca.crt contains custom CA (not real cluster CA)
    customCA, err := os.ReadFile(filepath.Join(fakeSAPath, "ca.crt"))
    require.NoError(t, err)
    
    realCA, err := os.ReadFile(filepath.Join(realSAPath, "ca.crt"))  
    require.NoError(t, err)
    
    assert.NotEqual(t, customCA, realCA, "Fake SA should have custom CA")
}
```

## Consequences

### Positive:
- **Complete Control**: Full control over credential distribution
- **Security Isolation**: Real credentials never exposed to applications
- **Standard Patterns**: Uses normal Kubernetes volume management
- **Token Rotation**: Automatic handling via projected volumes
- **No Synchronization**: No complex token copying or refresh logic

### Negative:  
- **Manual Configuration**: Requires explicit volume configuration
- **Complexity**: More complex than default ServiceAccount mounting
- **Volume Overhead**: Additional volumes and mounts required

### Operational:
- **Debugging**: Need to understand dual ServiceAccount setup
- **Documentation**: Must clearly explain credential flow
- **Pod Injection**: Automation required for ease of use (Phase 4)

## Future Considerations

### Phase 2 Multi-Cluster:
- Same pattern scales to multiple target clusters  
- MCA can manage credentials for multiple clusters
- Applications remain isolated with fake SA

### Phase 3 Observability:
- Clear audit trail of credential usage
- Monitoring real vs fake credential access
- Security alerts for credential anomalies  

### Phase 4 Production:
- Admission webhook automates volume configuration
- Policy-based ServiceAccount isolation
- Integration with external credential systems

This ServiceAccount bypass provides the security foundation needed for safe multi-cluster proxy operations while maintaining standard Kubernetes authentication patterns.