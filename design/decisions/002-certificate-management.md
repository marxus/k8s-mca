# ADR-002: Custom Certificate Management with Self-Signed CA

**Date**: 2025-11-19  
**Status**: Accepted  
**Context**: Phase 1 MITM Proxy Implementation

## Problem

MCA needs to terminate TLS connections from applications while maintaining security and trust. Kubernetes clients expect valid TLS certificates when connecting to API servers. We need a certificate management approach that:

1. Allows MCA to terminate HTTPS connections on localhost
2. Maintains client trust and certificate validation 
3. Works without access to the cluster's real Certificate Authority
4. Supports multiple client libraries and programming languages

## Decision

We will implement **custom certificate generation with self-signed Certificate Authority** and inject the custom CA into application containers.

### Specific Implementation:
- MCA generates its own CA certificate at startup
- Creates server certificate for localhost with proper SAN extensions
- Injects custom CA into application container's trust store
- Uses shared volumes for certificate distribution

## Rationale

### Certificate Requirements Analysis:

#### Option A: Use Real Cluster CA (Rejected)
- **Problem**: No access to cluster's private CA key
- **Security**: Cluster CA is tightly controlled by Kubernetes
- **Feasibility**: Would require privileged access to control plane

#### Option B: TLS Passthrough with SNI Inspection (Rejected)  
- **Problem**: Cannot inspect request content (core requirement)
- **Limitation**: Would only allow connection routing, not content analysis
- **Future**: Breaks Phase 2 routing based on request content

#### Option C: HTTP Proxy (Rejected)
- **Problem**: Kubernetes clients expect HTTPS for API servers
- **Security**: Breaks TLS security model within pod network
- **Compatibility**: Many clients hardcode HTTPS for security

#### Option D: Custom CA with Certificate Injection (Selected)
- **Advantages**: Full TLS termination and content inspection
- **Security**: Maintains encryption in pod network
- **Flexibility**: Enables content-based routing in future phases
- **Compatibility**: Works with all TLS-capable clients

## Implementation Details

### Certificate Generation
```go
type CertificateManager struct {
    CACert     *x509.Certificate
    CAKey      *rsa.PrivateKey  
    ServerCert tls.Certificate
}
```

### Server Certificate Requirements
- **Subject Alternative Names (SAN)**:
  - `DNS:localhost`
  - `IP:127.0.0.1` 
  - `IP:::1` (IPv6 localhost)
- **Key Usage**: Digital Signature, Key Encipherment
- **Extended Key Usage**: Server Authentication
- **Common Name**: Can be `localhost` or empty (SAN takes precedence)

### CA Injection Strategy
Applications cannot access real ServiceAccount CA due to our bypass mechanism, so we create a complete fake ServiceAccount structure:

```yaml
/var/run/secrets/kubernetes.io/serviceaccount/
├── ca.crt      # Our custom CA certificate
├── token       # Empty file (MCA handles real authentication)  
└── namespace   # Copied from real ServiceAccount
```

### Volume Architecture
```yaml
volumes:
- name: kube-api-access-sa          # Real SA for MCA
  projected: [...]
  
- name: kube-api-access-mca-sa      # Fake SA for application  
  emptyDir: {}
```

## Security Considerations

### Trust Boundary
- **MCA Container**: Has access to real cluster credentials
- **Application Container**: Only trusts MCA's custom CA
- **Network**: TLS encryption maintained end-to-end

### Certificate Lifecycle
- **Generation**: At MCA container startup
- **Distribution**: Via shared emptyDir volume
- **Rotation**: Handled in future phases (Phase 3)
- **Revocation**: Not required (pod-scoped certificates)

### Threat Model
- **Compromised Application**: Cannot access real cluster credentials
- **Certificate Extraction**: Certificates are pod-scoped and ephemeral
- **Man-in-the-Middle**: MCA IS the intended MITM proxy
- **Certificate Validation**: Standard TLS validation applies

## Technical Challenges & Solutions

### Challenge 1: SAN Extension Requirements
Modern TLS clients validate Subject Alternative Names over Common Name.

**Solution**: Generate certificates with comprehensive SAN entries covering all localhost variations.

### Challenge 2: Multi-Language Client Support
Different Kubernetes client libraries handle certificate validation differently.

**Solution**: Use standard x509 certificate format and proper SAN extensions that all clients recognize.

### Challenge 3: Certificate Distribution
Application containers need access to custom CA for trust validation.

**Solution**: Shared emptyDir volume with MCA writing certificates at startup.

### Challenge 4: Timing Dependencies  
Application must wait for MCA to generate certificates before starting.

**Solution**: True sidecar pattern ensures MCA runs first, plus health checks.

## Implementation Steps

1. **Certificate Generation Module**
   ```go
   func (cm *CertificateManager) GenerateCA() error
   func (cm *CertificateManager) GenerateServerCert() error  
   func (cm *CertificateManager) SaveCertificates(outputDir string) error
   ```

2. **Volume Management**
   - MCA writes to `/var/run/secrets/kubernetes.io/mca-serviceaccount/`
   - Application reads from `/var/run/secrets/kubernetes.io/serviceaccount/`

3. **TLS Server Configuration**
   ```go
   server := &http.Server{
       Addr: "127.0.0.1:6443",
       TLSConfig: &tls.Config{
           Certificates: []tls.Certificate{serverCert},
       },
   }
   ```

## Consequences

### Positive:
- Full TLS termination enables content inspection
- Standard certificate validation maintains security
- Works with all Kubernetes client libraries
- Foundation for advanced routing in future phases

### Negative:
- Adds complexity with custom certificate generation
- Requires careful certificate lifecycle management
- Additional volume mounts and file system operations

### Operational:
- Certificates are ephemeral (regenerated per pod)
- No certificate distribution infrastructure needed
- Debugging requires understanding of custom CA setup

## Future Considerations

### Phase 2 Benefits:
- Request content inspection enables sophisticated routing rules
- Certificate management scales to multi-cluster scenarios

### Phase 3 Enhancements:
- Certificate rotation and lifecycle management
- Integration with external CA systems (cert-manager)
- Enhanced security with certificate monitoring

### Phase 4 Production:
- Automated certificate management via operators
- Integration with enterprise PKI systems
- Certificate compliance and auditing

This approach provides the security foundation needed for production deployments while enabling the content inspection capabilities required for advanced multi-cluster routing.