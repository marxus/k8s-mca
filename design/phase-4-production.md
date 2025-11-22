# Phase 4: Production Deployment & Automation

**Status**: HIGH-LEVEL DESIGN - Needs Detailed Implementation Planning

## Overview

Phase 4 focuses on making MCA enterprise-ready for production deployments at scale. This phase introduces automation, self-service capabilities, and advanced lifecycle management through Kubernetes operators, admission webhooks, and comprehensive deployment tooling.

## Goals

- 🔄 Build Kubernetes admission webhook for automatic sidecar injection
- 🔄 Create comprehensive Helm charts for production deployment
- 🔄 Implement Kubernetes operator for advanced lifecycle management
- 🔄 Add CI/CD integration and automated testing pipelines
- 🔄 Provide self-service tools for application teams
- 🔄 Enable zero-downtime updates and blue-green deployments

## High-Level Architecture

### Production Deployment Stack
```
┌─────────────────────────────────────────────────────────────┐
│                 Kubernetes Cluster                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │ MCA Operator    │  │ Admission       │  │ ConfigMap   │  │
│  │ (Lifecycle Mgmt)│  │ Webhook         │  │ Controller  │  │  
│  └─────────────────┘  │ (Auto-Inject)   │  │ (Config)    │  │
│           │            └─────────────────┘  └─────────────┘  │
│           ▼                        │                 │       │
│  ┌─────────────────────────────────┼─────────────────┼──┐    │
│  │           Application Pods      │                 │  │    │
│  │  ┌────────────┬─────────────────┼─────────────────┼┐ │    │
│  │  │    App     │      MCA        │     Injected    ││ │    │
│  │  │ Container  │   Sidecar       │   Automatically ││ │    │
│  │  └────────────┴─────────────────┴─────────────────┼┘ │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
         │                           │                   │
         ▼                           ▼                   ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Helm Charts   │    │   CI/CD         │    │ Monitoring &    │
│  (Deployment)   │    │ Pipelines       │    │ Alerting        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Core Components

### 1. Admission Webhook for Automatic Injection

#### MutatingAdmissionWebhook
```go
type MCAAdmissionController struct {
    server      *http.Server
    certManager *webhook.CertManager
    injector    *inject.PodInjector
    validator   *ConfigValidator
}

type WebhookConfig struct {
    Port            int    `yaml:"port"`
    CertDir         string `yaml:"certDir"`
    WebhookName     string `yaml:"webhookName"`
    NamespaceLabel  string `yaml:"namespaceLabel"`  // mca.io/injection: enabled
    PodAnnotation   string `yaml:"podAnnotation"`   // mca.io/inject: "true"
}

func (ac *MCAAdmissionController) ServeMutate(w http.ResponseWriter, r *http.Request) {
    // 1. Parse admission review request
    // 2. Extract pod spec from request
    // 3. Apply injection logic based on annotations/labels
    // 4. Return mutated pod spec in admission response
}
```

#### Injection Rules & Configuration
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionWebhook
metadata:
  name: mca-sidecar-injector
webhooks:
- name: sidecar-injector.mca.io
  clientConfig:
    service:
      name: mca-admission-webhook
      namespace: mca-system
      path: /mutate
  rules:
  - operations: ["CREATE", "UPDATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  failurePolicy: Fail
```

#### Selective Injection Logic
```go
type InjectionPolicy struct {
    NamespaceSelector *metav1.LabelSelector `yaml:"namespaceSelector"`
    PodSelector       *metav1.LabelSelector `yaml:"podSelector"`
    ExcludeNamespaces []string              `yaml:"excludeNamespaces"`
    InjectionMode     string                `yaml:"injectionMode"` // always, opt-in, opt-out
}

func (ip *InjectionPolicy) ShouldInject(pod *corev1.Pod, namespace *corev1.Namespace) bool {
    // Check namespace exclusions
    // Evaluate namespace selector
    // Check pod annotations for opt-in/opt-out
    // Apply injection mode logic
}
```

### 2. Kubernetes Operator for Lifecycle Management

#### Custom Resource Definitions
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: mcaconfigs.mca.io
spec:
  group: mca.io
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              clusters:
                type: array
                items:
                  type: object
                  properties:
                    name: {type: string}
                    endpoint: {type: string}
                    weight: {type: integer}
              routing:
                type: object
                properties:
                  rules:
                    type: array
                    items:
                      type: object
          status:
            type: object
            properties:
              phase: {type: string}
              conditions:
                type: array
                items:
                  type: object
```

#### MCA Operator Implementation
```go
type MCAOperator struct {
    client          client.Client
    scheme          *runtime.Scheme
    configManager   *ConfigManager
    webhookManager  *WebhookManager
    metricsRecorder metrics.Recorder
}

func (r *MCAOperator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Fetch MCAConfig instance
    // Validate configuration
    // Update cluster registry
    // Reconcile routing rules
    // Update status and conditions
    // Handle deletions and cleanup
}

// Controller watches:
// - MCAConfig CRs
// - ConfigMaps with cluster configuration
// - Secrets with cluster credentials  
// - Pods with MCA sidecars (for health status)
```

#### Operator Capabilities
```go
type OperatorFeatures struct {
    AutoScaling        bool `yaml:"autoScaling"`
    HealthMonitoring   bool `yaml:"healthMonitoring"`
    ConfigValidation   bool `yaml:"configValidation"`
    CertificateRotation bool `yaml:"certificateRotation"`
    BackupRestore      bool `yaml:"backupRestore"`
    MultiTenancy       bool `yaml:"multiTenancy"`
}

func (r *MCAOperator) HandleAutoScaling(config *v1.MCAConfig) error {
    // Monitor proxy load and performance
    // Scale sidecar resources based on traffic
    // Adjust connection pools and timeouts
}

func (r *MCAOperator) HandleCertificateRotation(config *v1.MCAConfig) error {
    // Monitor certificate expiration
    // Generate new certificates before expiry
    // Rolling update of pods with new certificates
}
```

### 3. Comprehensive Helm Charts

#### Chart Structure
```
charts/mca/
├── Chart.yaml
├── values.yaml
├── values-production.yaml
├── templates/
│   ├── operator/
│   │   ├── deployment.yaml
│   │   ├── rbac.yaml
│   │   └── crd.yaml
│   ├── webhook/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── mutatingwebhook.yaml
│   │   └── certificates.yaml
│   ├── monitoring/
│   │   ├── servicemonitor.yaml
│   │   ├── grafana-dashboard.yaml
│   │   └── alertmanager-rules.yaml
│   └── rbac/
│       ├── clusterrole.yaml
│       ├── serviceaccount.yaml
│       └── rolebinding.yaml
└── crds/
    └── mcaconfig-crd.yaml
```

#### Production Values Configuration
```yaml
# values-production.yaml
global:
  imageRegistry: registry.company.com
  imagePullSecrets:
  - name: registry-credentials

operator:
  image:
    repository: mca/operator
    tag: v1.0.0
  replicas: 3
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi

webhook:
  image:
    repository: mca/webhook
    tag: v1.0.0
  replicas: 2
  tls:
    certManager: true
    issuerRef:
      name: ca-issuer
      kind: ClusterIssuer

sidecar:
  image:
    repository: mca/proxy
    tag: v1.0.0
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 256Mi

monitoring:
  prometheus:
    enabled: true
    serviceMonitor:
      enabled: true
  grafana:
    enabled: true
    dashboards:
      enabled: true

security:
  podSecurityPolicy:
    enabled: true
  networkPolicy:
    enabled: true
  rbac:
    create: true
```

### 4. CI/CD Integration & Automated Testing

#### GitHub Actions Pipeline
```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run Unit Tests
      run: go test -v -cover ./...
    
    - name: Run Integration Tests
      run: make integration-test
    
    - name: Run E2E Tests
      run: make e2e-test
      
  security-scan:
    runs-on: ubuntu-latest
    steps:
    - name: Run Trivy Scanner
      run: trivy fs --security-checks vuln .
    
    - name: Run Gosec
      run: gosec ./...
      
  build-and-deploy:
    needs: [test, security-scan]
    runs-on: ubuntu-latest
    steps:
    - name: Build Docker Images
      run: |
        docker build -t mca/operator:${{ github.sha }} -f build/operator/Dockerfile .
        docker build -t mca/webhook:${{ github.sha }} -f build/webhook/Dockerfile .
        docker build -t mca/proxy:${{ github.sha }} -f build/proxy/Dockerfile .
    
    - name: Deploy to Staging
      if: github.ref == 'refs/heads/develop'
      run: |
        helm upgrade --install mca-staging ./charts/mca \
          --namespace mca-staging \
          --set global.imageTag=${{ github.sha }}
    
    - name: Deploy to Production
      if: github.ref == 'refs/heads/main'
      run: |
        helm upgrade --install mca-production ./charts/mca \
          --namespace mca-system \
          --values charts/mca/values-production.yaml \
          --set global.imageTag=${{ github.sha }}
```

#### Automated Testing Framework
```go
// test/e2e/e2e_test.go
func TestMCAE2E(t *testing.T) {
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{filepath.Join("..", "..", "crds")},
    }
    
    cfg, err := testEnv.Start()
    require.NoError(t, err)
    defer testEnv.Stop()
    
    t.Run("Webhook Injection", testWebhookInjection)
    t.Run("Multi-Cluster Routing", testMultiClusterRouting)
    t.Run("Health Monitoring", testHealthMonitoring)
    t.Run("Certificate Rotation", testCertificateRotation)
}

func testWebhookInjection(t *testing.T) {
    // Create test pod
    // Verify sidecar injection
    // Validate configuration
}
```

### 5. Self-Service Tools & Developer Experience

#### MCA CLI Tool
```go
// cmd/mcactl/main.go
type MCACtl struct {
    clientset  kubernetes.Interface
    mcaClient  versioned.Interface
    namespace  string
}

var rootCmd = &cobra.Command{
    Use:   "mcactl",
    Short: "MCA management CLI",
}

// Commands:
// mcactl install --namespace mca-system
// mcactl config validate --file config.yaml  
// mcactl status --cluster prod-cluster
// mcactl logs --pod app-pod --follow
// mcactl debug routing --request request.json
// mcactl certificate rotate --cluster all
```

#### Developer Portal Integration
```yaml
# docs/developer-portal.yaml
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: mca-multi-cluster-adapter
  description: Kubernetes Multi-Cluster Adapter
spec:
  type: service
  lifecycle: production
  owner: platform-team
  system: kubernetes-infrastructure
  dependsOn:
  - component:prometheus
  - component:grafana
  - resource:kubernetes-cluster
  providesApis:
  - mca-proxy-api
  - mca-management-api
```

### 6. Enterprise Features

#### Multi-Tenancy Support
```go
type TenantManager struct {
    tenants map[string]*Tenant
    rbac    *RBACManager
}

type Tenant struct {
    Name        string
    Namespaces  []string
    Clusters    []string
    ResourceQuota *ResourceQuota
    NetworkPolicies []NetworkPolicy
}
```

#### Disaster Recovery & Backup
```go
type BackupManager struct {
    s3Client    *s3.S3
    encryption  *EncryptionManager
    schedule    string
    retention   time.Duration
}

func (bm *BackupManager) BackupConfiguration() error {
    // Backup MCAConfig CRs
    // Backup routing rules
    // Backup cluster credentials (encrypted)
    // Upload to S3 with versioning
}
```

#### Compliance & Auditing
```go
type ComplianceManager struct {
    auditLogger   *AuditLogger  
    policyEngine  *PolicyEngine
    reportGenerator *ReportGenerator
}

func (cm *ComplianceManager) GenerateComplianceReport() error {
    // SOX compliance reporting
    // GDPR data handling audit
    // Security posture assessment
    // Access control review
}
```

## Deployment Strategies

### Blue-Green Deployment
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: mca-operator
spec:
  replicas: 3
  strategy:
    blueGreen:
      activeService: mca-operator-active
      previewService: mca-operator-preview
      autoPromotionEnabled: false
      scaleDownDelaySeconds: 30
      prePromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: mca-operator-preview
      postPromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: mca-operator-active
```

### Canary Deployment with Flagger
```yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: mca-proxy
  namespace: mca-system
spec:
  targetRef:
    apiVersion: apps/v1
    kind: DaemonSet
    name: mca-proxy
  progressDeadlineSeconds: 60
  service:
    port: 6443
  analysis:
    interval: 30s
    threshold: 5
    maxWeight: 50
    stepWeight: 10
    metrics:
    - name: request-success-rate
      thresholdRange:
        min: 99
      interval: 1m
    - name: request-duration
      thresholdRange:
        max: 500
      interval: 30s
```

## Monitoring & Observability in Production

### SLO/SLI Framework
```yaml
apiVersion: sloth.slok.dev/v1
kind: PrometheusServiceLevel
metadata:
  name: mca-proxy-availability
spec:
  service: "mca-proxy"
  labels:
    team: "platform"
  slos:
  - name: "requests-availability"
    objective: 99.9
    description: "99.9% of requests should be successful"
    sli:
      events:
        errorQuery: sum(rate(mca_requests_total{code=~"5.."}[5m]))
        totalQuery: sum(rate(mca_requests_total[5m]))
  - name: "requests-latency"
    objective: 99.5
    description: "99.5% of requests should be faster than 100ms"
    sli:
      events:
        errorQuery: sum(rate(mca_request_duration_seconds_bucket{le="0.1"}[5m]))
        totalQuery: sum(rate(mca_request_duration_seconds_count[5m]))
```

## Future Considerations

### Advanced Automation
- **AI/ML-powered Optimization**: Intelligent routing and scaling decisions
- **Chaos Engineering**: Automated resilience testing with Chaos Monkey
- **Cost Optimization**: Automated right-sizing and cluster selection
- **Predictive Analytics**: Proactive issue detection and prevention

### Ecosystem Integration
- **Service Mesh Integration**: Deep integration with Istio, Linkerd, Consul Connect
- **Policy as Code**: Integration with OPA Gatekeeper, Falco, Kyverno  
- **GitOps**: Native ArgoCD/Flux operator integration
- **Secrets Management**: Vault, External Secrets Operator integration

## Success Criteria

### Deployment Automation
- Zero-touch installation via Helm charts
- Automatic sidecar injection with 99.9% success rate
- Blue-green deployments with automated rollback
- Configuration changes propagated within 30 seconds

### Developer Experience  
- Self-service onboarding in under 5 minutes
- Comprehensive CLI tooling and documentation
- IDE plugins and development environment integration
- 24/7 support with SLA guarantees

### Enterprise Readiness
- Multi-tenant isolation and resource quotas
- Comprehensive audit logging and compliance reporting
- Disaster recovery with <15 minute RTO
- 99.99% uptime SLA for critical components

Phase 4 delivers a complete, enterprise-grade platform that enables organizations to deploy and manage MCA at scale with confidence, automation, and comprehensive operational support.