# Phase 2: Multi-Cluster Routing & Configuration

**Status**: HIGH-LEVEL DESIGN - Needs Detailed Implementation Planning

## Overview

Phase 2 extends MCA beyond simple MITM proxy functionality to enable routing of Kubernetes API calls across multiple clusters. This phase introduces cluster registry management, routing policies, and load balancing capabilities.

## Goals

- 🔄 Route API calls to different target clusters based on configurable rules
- 🔄 Implement cluster registry for managing multiple cluster configurations  
- 🔄 Add routing policies (namespace-based, resource-based, label-based)
- 🔄 Provide load balancing across multiple target clusters
- 🔄 Handle cluster-specific authentication and credential management
- 🔄 Support dynamic configuration updates via ConfigMaps/Secrets

## High-Level Architecture

### Multi-Cluster Request Flow
```
Application Container
        │
        ▼ (127.0.0.1:6443)
┌─────────────────┐
│   MCA Proxy     │
├─────────────────┤
│ Request Parser  │ ← Parse K8s API request
│ Routing Engine  │ ← Apply routing rules
│ Load Balancer   │ ← Select target cluster
│ Cluster Client  │ ← Forward with correct auth
└─────────────────┘
        │
        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Cluster A     │    │   Cluster B     │    │   Cluster C     │
│  (Production)   │    │  (Staging)      │    │  (Development)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Core Components

### 1. Cluster Registry
Configuration management for multiple target clusters.

#### Cluster Configuration
```go
type ClusterConfig struct {
    Name            string            `yaml:"name"`
    Endpoint        string            `yaml:"endpoint"`
    CertificateData []byte            `yaml:"certificateData"`
    Token           string            `yaml:"token"`
    Namespace       string            `yaml:"namespace,omitempty"`
    Labels          map[string]string `yaml:"labels,omitempty"`
    Weight          int               `yaml:"weight,omitempty"` // For load balancing
    HealthCheck     HealthCheckConfig `yaml:"healthCheck"`
}

type ClusterRegistry struct {
    Clusters map[string]*ClusterConfig
    mutex    sync.RWMutex
}
```

#### Configuration Sources
- **ConfigMaps**: Cluster endpoint and metadata
- **Secrets**: Authentication tokens and certificates
- **File-based**: Static configuration files
- **Dynamic Updates**: Watch ConfigMap/Secret changes

### 2. Routing Engine
Rule-based routing to determine target clusters.

#### Routing Rules
```go
type RoutingRule struct {
    Name         string              `yaml:"name"`
    Priority     int                 `yaml:"priority"`
    Conditions   []RoutingCondition  `yaml:"conditions"`
    Actions      RoutingAction       `yaml:"actions"`
}

type RoutingCondition struct {
    Type      string `yaml:"type"`      // namespace, resource, verb, label
    Operator  string `yaml:"operator"`  // equals, contains, regex
    Value     string `yaml:"value"`
}

type RoutingAction struct {
    TargetCluster string   `yaml:"targetCluster,omitempty"`
    TargetClusters []string `yaml:"targetClusters,omitempty"` // For load balancing
    LoadBalancing  string   `yaml:"loadBalancing,omitempty"`  // round-robin, weighted, random
}
```

#### Example Routing Configuration
```yaml
rules:
- name: production-workloads
  priority: 100
  conditions:
  - type: namespace
    operator: equals
    value: production
  actions:
    targetCluster: prod-cluster

- name: staging-workloads  
  priority: 90
  conditions:
  - type: namespace
    operator: contains
    value: staging
  actions:
    targetClusters: [staging-cluster-1, staging-cluster-2]
    loadBalancing: round-robin

- name: development-resources
  priority: 80
  conditions:
  - type: resource
    operator: equals
    value: pods
  - type: verb
    operator: equals
    value: GET
  actions:
    targetClusters: [dev-cluster-1, dev-cluster-2, dev-cluster-3]
    loadBalancing: weighted
```

### 3. Load Balancing
Distribute requests across multiple target clusters.

#### Load Balancing Strategies
```go
type LoadBalancer interface {
    SelectCluster(clusters []string, request *http.Request) (string, error)
}

type RoundRobinBalancer struct {
    counters map[string]uint64
    mutex    sync.Mutex
}

type WeightedBalancer struct {
    weights map[string]int
}

type RandomBalancer struct{}
```

### 4. Authentication Management
Handle cluster-specific credentials and authentication.

#### Credential Management
```go
type AuthProvider interface {
    GetCredentials(clusterName string) (*Credentials, error)
    RefreshToken(clusterName string) error
}

type Credentials struct {
    Token           string
    CertificateData []byte
    KeyData         []byte
}

type SecretAuthProvider struct {
    clientset kubernetes.Interface
    namespace string
}
```

## Request Processing Pipeline

### Enhanced Middleware Stack
```go
type MiddlewareStack []Middleware

var DefaultMiddleware = MiddlewareStack{
    LoggingMiddleware(),
    MetricsMiddleware(),          // Phase 3 addition
    RequestParsingMiddleware(),   // Parse K8s API request
    RoutingMiddleware(),          // Apply routing rules
    LoadBalancingMiddleware(),    // Select target cluster
    AuthenticationMiddleware(),   // Handle cluster credentials
    ProxyMiddleware(),            // Forward to target cluster
}
```

### Request Context Enhancement
```go
type RequestContext struct {
    OriginalRequest  *http.Request
    ParsedRequest    *K8sAPIRequest
    RoutingRules     []RoutingRule
    TargetCluster    string
    Credentials      *Credentials
    LoadBalanceState interface{}
}

type K8sAPIRequest struct {
    APIVersion  string
    Kind        string
    Namespace   string
    Name        string
    Resource    string
    SubResource string
    Verb        string
    Labels      map[string]string
}
```

## Configuration Management

### Dynamic Configuration Updates
```go
type ConfigManager struct {
    clusterRegistry *ClusterRegistry
    routingEngine   *RoutingEngine
    configWatcher   *ConfigWatcher
}

type ConfigWatcher struct {
    configMapWatcher cache.SharedInformer
    secretWatcher    cache.SharedInformer
}

func (cm *ConfigManager) WatchConfigMaps() error {
    // Watch for ConfigMap changes and update routing rules
}

func (cm *ConfigManager) WatchSecrets() error {
    // Watch for Secret changes and update cluster credentials  
}
```

### Configuration Schema
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mca-config
  namespace: mca-system
data:
  clusters.yaml: |
    clusters:
    - name: prod-cluster
      endpoint: https://prod.k8s.example.com
      namespace: production
      weight: 100
    - name: staging-cluster  
      endpoint: https://staging.k8s.example.com
      weight: 50
      
  routing.yaml: |
    rules:
    - name: production-routing
      conditions: [...]
      actions: [...]
---
apiVersion: v1
kind: Secret
metadata:
  name: mca-cluster-credentials
  namespace: mca-system
type: Opaque
data:
  prod-cluster-token: <base64-encoded-token>
  staging-cluster-cert: <base64-encoded-cert>
```

## Health Monitoring

### Cluster Health Checks
```go
type HealthChecker struct {
    clusters map[string]*ClusterConfig
    results  map[string]*HealthResult
    interval time.Duration
}

type HealthResult struct {
    Cluster     string
    Healthy     bool
    LastCheck   time.Time
    ResponseTime time.Duration
    Error       error
}

func (hc *HealthChecker) CheckCluster(cluster string) *HealthResult {
    // Perform health check against cluster API server
    // Update routing decisions based on cluster health
}
```

## Error Handling & Failover

### Failover Strategies
```go
type FailoverStrategy interface {
    HandleFailure(cluster string, err error, request *RequestContext) (*RequestContext, error)
}

type RetryStrategy struct {
    MaxRetries    int
    BackoffFactor float64
    MaxDelay      time.Duration
}

type CircuitBreakerStrategy struct {
    FailureThreshold int
    RecoveryTimeout  time.Duration
    HalfOpenRequests int
}
```

## Future Considerations

### Advanced Routing Features
- **Geographic Routing**: Route based on cluster location/region
- **Cost-Based Routing**: Factor in cluster costs for optimization
- **Capacity-Based Routing**: Consider cluster resource availability
- **Time-Based Routing**: Route based on time windows (maintenance, etc.)

### Security Enhancements
- **mTLS Between Clusters**: Secure inter-cluster communication
- **Token Rotation**: Automatic credential refresh
- **Audit Logging**: Track all multi-cluster API calls
- **Policy Enforcement**: Implement fine-grained access controls

### Performance Optimizations
- **Connection Pooling**: Reuse connections to target clusters
- **Request Caching**: Cache responses for read operations
- **Batch Operations**: Combine multiple requests where possible
- **Async Processing**: Non-blocking request forwarding

## Implementation Roadmap

### Phase 2.1: Basic Multi-Cluster Support
- Cluster registry with static configuration
- Simple namespace-based routing rules
- Round-robin load balancing
- Basic health checking

### Phase 2.2: Advanced Routing
- Complex routing conditions and rules
- Weighted load balancing
- Failover and retry mechanisms
- Dynamic configuration updates

### Phase 2.3: Production Features
- Comprehensive health monitoring
- Circuit breakers and error handling
- Performance optimizations
- Security enhancements

## Success Criteria

### Functional Requirements
- Route API calls to appropriate clusters based on rules
- Support dynamic addition/removal of clusters
- Handle cluster failures gracefully with failover
- Maintain session affinity where required

### Performance Requirements
- Routing decision latency < 5ms
- Support for 1000+ concurrent requests
- Health check frequency configurable (default 30s)
- Graceful degradation under high load

### Operational Requirements
- Zero-downtime configuration updates
- Comprehensive monitoring and alerting
- Clear error messages and troubleshooting
- Documentation for common routing patterns

This high-level design provides the foundation for multi-cluster capabilities while building on the solid MITM proxy foundation from Phase 1.