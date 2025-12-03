# MCA Deployment Guide

**Last Updated**: 2025-12-03
**Target Version**: v0.1.0+

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
- [Helm Chart Deployment](#helm-chart-deployment)
- [Manual Deployment](#manual-deployment)
- [CI/CD Integration](#cicd-integration)
- [Configuration](#configuration)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Upgrades & Rollbacks](#upgrades--rollbacks)
- [Uninstallation](#uninstallation)

## Prerequisites

### Kubernetes Cluster Requirements

- **Kubernetes Version**: 1.16+ (admission webhooks require 1.16+)
- **RBAC Enabled**: Required for ServiceAccount and webhook permissions
- **Admission Controllers**: `MutatingAdmissionWebhook` must be enabled
- **Network Policy**: Allow webhook pod to access API server (if network policies enforced)

### Client Tools

| Tool | Minimum Version | Purpose |
|------|----------------|---------|
| kubectl | 1.16+ | Kubernetes CLI |
| helm | 3.0+ | Helm chart installation (for webhook deployment) |
| docker | 19.03+ | Building MCA images (optional) |
| go | 1.21+ | Building from source (optional) |

### Verify Prerequisites

```bash
# Check Kubernetes version
kubectl version --short

# Check if MutatingAdmissionWebhook is enabled
kubectl api-versions | grep admissionregistration.k8s.io

# Check RBAC
kubectl auth can-i create clusterroles --all-namespaces
```

## Deployment Options

### Option 1: Webhook Deployment (Recommended for Production)

**Pros**:
- Automatic injection on pod creation
- Cluster-wide policy enforcement
- No manual intervention required
- GitOps friendly (labels in manifests)

**Cons**:
- Requires cluster admin permissions
- More complex setup
- Potential single point of failure

**Use Cases**: Production environments, multi-team clusters

### Option 2: CLI Injection (Recommended for Development)

**Pros**:
- No cluster-side components
- Simple and fast
- Full control over which pods get injected
- Works offline

**Cons**:
- Manual process for each pod
- Requires CI/CD integration for automation
- No enforcement of injection policies

**Use Cases**: Local development, CI/CD pipelines, testing

## Helm Chart Deployment

### Quick Start

```bash
# Add the MCA Helm repository (if published)
helm repo add mca https://marxus.github.io/k8s-mca
helm repo update

# Or use local chart
cd charts/mca

# Install with default values
helm install mca . --namespace mca-system --create-namespace

# Install with custom values
helm install mca . \
  --namespace mca-system \
  --create-namespace \
  --set image.repository=ghcr.io/marxus/k8s-mca \
  --set image.tag=v0.1.0 \
  --set webhook.replicas=2
```

### Chart Values

Create a `values.yaml` file for customization:

```yaml
# Image configuration
image:
  repository: ghcr.io/marxus/k8s-mca
  tag: v0.1.0
  pullPolicy: IfNotPresent

# Webhook deployment
webhook:
  replicas: 1  # Increase for high availability
  resources:
    requests:
      memory: "64Mi"
      cpu: "100m"
    limits:
      memory: "128Mi"
      cpu: "200m"

# ServiceAccount
serviceAccount:
  create: true
  name: mca-webhook
  annotations: {}

# RBAC
rbac:
  create: true

# Webhook configuration
webhookConfig:
  failurePolicy: Fail  # or Ignore
  namespaceSelector: {}  # Can add namespace filters
  objectSelector:
    matchLabels:
      mca.k8s.io/inject: "true"

# Security context
securityContext:
  runAsNonRoot: true
  runAsUser: 999
  fsGroup: 999
```

Install with custom values:

```bash
helm install mca ./charts/mca \
  --namespace mca-system \
  --create-namespace \
  --values custom-values.yaml
```

### Helm Chart Structure

```
charts/mca/
├── Chart.yaml                 # Chart metadata
├── values.yaml               # Default values
└── templates/
    ├── deployment.yaml       # Webhook deployment
    ├── service.yaml         # Webhook service
    ├── serviceaccount.yaml  # ServiceAccount for webhook
    ├── rbac.yaml           # ClusterRole & RoleBinding
    ├── webhook.yaml        # MutatingWebhookConfiguration
    └── _helpers.tpl        # Template helpers
```

### Post-Installation Verification

```bash
# Check deployment status
helm status mca -n mca-system

# Check pod status
kubectl get pods -n mca-system -l app=mca-webhook

# Check webhook configuration
kubectl get mutatingwebhookconfiguration mca-webhook

# Check logs
kubectl logs -n mca-system -l app=mca-webhook --tail=50
```

## Manual Deployment

### Step 1: Build MCA Image

```bash
# Clone repository
git clone https://github.com/marxus/k8s-mca.git
cd k8s-mca

# Build Docker image
docker build -t mca:v0.1.0 .

# Push to registry
docker tag mca:v0.1.0 your-registry/mca:v0.1.0
docker push your-registry/mca:v0.1.0
```

### Step 2: Create Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: mca-system
```

```bash
kubectl apply -f namespace.yaml
```

### Step 3: Create ServiceAccount

```yaml
# serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mca-webhook
  namespace: mca-system
```

```bash
kubectl apply -f serviceaccount.yaml
```

### Step 4: Create RBAC Resources

```yaml
# rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mca-webhook
rules:
- apiGroups: ["admissionregistration.k8s.io"]
  resources: ["mutatingwebhookconfigurations"]
  verbs: ["get", "list", "watch", "patch", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: mca-webhook
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: mca-webhook
subjects:
- kind: ServiceAccount
  name: mca-webhook
  namespace: mca-system
```

```bash
kubectl apply -f rbac.yaml
```

### Step 5: Create Webhook Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mca-webhook
  namespace: mca-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mca-webhook
  template:
    metadata:
      labels:
        app: mca-webhook
    spec:
      serviceAccountName: mca-webhook
      containers:
      - name: mca-webhook
        image: your-registry/mca:v0.1.0
        imagePullPolicy: IfNotPresent
        args: ["--webhook"]
        env:
        - name: MCA_PROXY_IMAGE
          value: "your-registry/mca:v0.1.0"
        - name: MCA_WEBHOOK_NAME
          value: "mca-webhook"
        ports:
        - containerPort: 8443
          name: webhook
          protocol: TCP
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 999
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
```

```bash
kubectl apply -f deployment.yaml
```

### Step 6: Create Webhook Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: mca-webhook
  namespace: mca-system
spec:
  selector:
    app: mca-webhook
  ports:
  - name: webhook
    port: 443
    targetPort: 8443
    protocol: TCP
```

```bash
kubectl apply -f service.yaml
```

### Step 7: Create MutatingWebhookConfiguration

```yaml
# webhook-config.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mca-webhook
webhooks:
- name: mca-webhook.k8s.io
  clientConfig:
    service:
      name: mca-webhook
      namespace: mca-system
      path: /mutate
    # CA bundle will be automatically patched by webhook at startup
    caBundle: ""
  rules:
  - operations: [CREATE]
    apiGroups: [""]
    apiVersions: [v1]
    resources: [pods]
  objectSelector:
    matchLabels:
      mca.k8s.io/inject: "true"
  admissionReviewVersions: [v1, v1beta1]
  sideEffects: None
  failurePolicy: Fail
  reinvocationPolicy: IfNeeded
```

```bash
kubectl apply -f webhook-config.yaml
```

### Step 8: Wait for Webhook to Self-Configure

```bash
# Wait for deployment
kubectl rollout status deployment/mca-webhook -n mca-system

# Check logs (should see CA bundle patching)
kubectl logs -n mca-system -l app=mca-webhook

# Verify CA bundle is set
kubectl get mutatingwebhookconfiguration mca-webhook \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | wc -c
# Should output non-zero number (base64 encoded CA)
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yaml
name: Deploy MCA

on:
  push:
    tags:
      - 'v*'

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: |
          docker build -t ghcr.io/${{ github.repository }}:${{ github.ref_name }} .

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Push image
        run: |
          docker push ghcr.io/${{ github.repository }}:${{ github.ref_name }}

      - name: Deploy with Helm
        run: |
          helm upgrade --install mca ./charts/mca \
            --namespace mca-system \
            --create-namespace \
            --set image.repository=ghcr.io/${{ github.repository }} \
            --set image.tag=${{ github.ref_name }}
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - build
  - deploy

build:
  stage: build
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG
  only:
    - tags

deploy:
  stage: deploy
  script:
    - helm upgrade --install mca ./charts/mca \
        --namespace mca-system \
        --create-namespace \
        --set image.repository=$CI_REGISTRY_IMAGE \
        --set image.tag=$CI_COMMIT_TAG
  only:
    - tags
```

### CLI Injection in CI/CD

```yaml
# .github/workflows/inject-and-deploy.yaml
name: Deploy with MCA Injection

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install MCA CLI
        run: |
          go install github.com/marxus/k8s-mca/cmd/mca@latest

      - name: Inject and deploy
        run: |
          mca --inject < k8s/deployment.yaml | kubectl apply -f -
```

## Configuration

### Environment Variables

Configure the webhook deployment with environment variables:

```yaml
env:
  # Required: Image to inject as proxy
  - name: MCA_PROXY_IMAGE
    value: "ghcr.io/marxus/k8s-mca:v0.1.0"

  # Required: Webhook configuration name
  - name: MCA_WEBHOOK_NAME
    value: "mca-webhook"
```

### Namespace Selectors

Limit webhook to specific namespaces:

```yaml
# In MutatingWebhookConfiguration
webhooks:
- name: mca-webhook.k8s.io
  namespaceSelector:
    matchLabels:
      mca-injection: enabled
  # ... rest of config
```

Then label namespaces:

```bash
kubectl label namespace my-app mca-injection=enabled
```

### Failure Policies

Choose webhook behavior on failure:

```yaml
# Fail pod creation if webhook fails (recommended for production)
failurePolicy: Fail

# Allow pod creation if webhook fails (for testing)
failurePolicy: Ignore
```

### Resource Limits

Adjust based on cluster size and pod creation rate:

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"  # Increase for high-traffic clusters
    cpu: "500m"      # Increase for high pod creation rate
```

## Verification

### Verify Webhook Installation

```bash
# Check webhook deployment
kubectl get deployment -n mca-system mca-webhook
kubectl get pods -n mca-system -l app=mca-webhook

# Check webhook service
kubectl get svc -n mca-system mca-webhook

# Check mutating webhook configuration
kubectl get mutatingwebhookconfiguration mca-webhook

# Verify CA bundle is populated
kubectl get mutatingwebhookconfiguration mca-webhook \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | base64 -d | openssl x509 -text -noout
```

### Test Injection

```bash
# Create test pod with injection label
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-injection
  labels:
    mca.k8s.io/inject: "true"
spec:
  containers:
  - name: nginx
    image: nginx:latest
EOF

# Verify MCA init container was injected
kubectl get pod test-injection -o jsonpath='{.spec.initContainers[0].name}'
# Expected output: mca-proxy

# Verify environment variables
kubectl get pod test-injection -o jsonpath='{.spec.containers[0].env[?(@.name=="KUBERNETES_SERVICE_HOST")].value}'
# Expected output: 127.0.0.1

# Verify volumes
kubectl get pod test-injection -o jsonpath='{.spec.volumes[*].name}' | grep kube-api-access-mca-sa
# Should find the volume

# Check logs
kubectl logs test-injection -c mca-proxy
```

### Test API Calls Through Proxy

```bash
# Create pod with kubectl
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-api-calls
  labels:
    mca.k8s.io/inject: "true"
spec:
  containers:
  - name: kubectl
    image: bitnami/kubectl:latest
    command: ["/bin/sh", "-c", "kubectl get pods && sleep 3600"]
EOF

# Check logs - should see successful API calls
kubectl logs test-api-calls -c kubectl

# Check proxy logs - should see proxied requests
kubectl logs test-api-calls -c mca-proxy
```

## Troubleshooting

### Webhook Pod Not Starting

```bash
# Check pod status
kubectl get pods -n mca-system -l app=mca-webhook

# Check pod events
kubectl describe pod -n mca-system -l app=mca-webhook

# Check logs
kubectl logs -n mca-system -l app=mca-webhook --tail=100

# Common issues:
# - Image pull errors: Check image name and registry credentials
# - RBAC errors: Check ServiceAccount has correct permissions
# - CrashLoopBackOff: Check logs for startup errors
```

### CA Bundle Not Set

```bash
# Check webhook logs
kubectl logs -n mca-system -l app=mca-webhook | grep "CA bundle"

# Manually verify patch permissions
kubectl auth can-i patch mutatingwebhookconfigurations --as=system:serviceaccount:mca-system:mca-webhook

# If permissions missing, reapply RBAC
kubectl apply -f charts/mca/templates/rbac.yaml
```

### Pod Creation Blocked

```bash
# Check webhook configuration
kubectl get mutatingwebhookconfiguration mca-webhook -o yaml

# Check webhook service endpoints
kubectl get endpoints -n mca-system mca-webhook

# Test webhook endpoint manually
kubectl run test --image=nginx --labels="mca.k8s.io/inject=true" --dry-run=server

# Check API server logs (if accessible)
# Look for webhook call failures

# Temporary workaround: Set failurePolicy to Ignore
kubectl patch mutatingwebhookconfiguration mca-webhook \
  --type='json' \
  -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Ignore"}]'
```

### Injection Not Working

```bash
# Verify pod has injection label
kubectl get pod <pod-name> --show-labels

# Check webhook selector
kubectl get mutatingwebhookconfiguration mca-webhook \
  -o jsonpath='{.webhooks[0].objectSelector}'

# Test with explicit label
kubectl run test-inject \
  --image=nginx \
  --labels="mca.k8s.io/inject=true" \
  --dry-run=server -o yaml

# Check webhook logs for incoming requests
kubectl logs -n mca-system -l app=mca-webhook --tail=20
```

### Proxy Not Forwarding Requests

```bash
# Check proxy logs in application pod
kubectl logs <pod-name> -c mca-proxy

# Verify environment variables
kubectl exec <pod-name> -c <app-container> -- env | grep KUBERNETES_SERVICE

# Verify volume mounts
kubectl exec <pod-name> -c <app-container> -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Test connectivity to proxy
kubectl exec <pod-name> -c <app-container> -- curl -k https://127.0.0.1:6443/version
```

## Upgrades & Rollbacks

### Upgrade Helm Release

```bash
# Update chart values
vim values.yaml  # Change image.tag

# Upgrade release
helm upgrade mca ./charts/mca \
  --namespace mca-system \
  --values values.yaml

# Or upgrade with inline values
helm upgrade mca ./charts/mca \
  --namespace mca-system \
  --set image.tag=v0.2.0 \
  --reuse-values

# Check upgrade status
helm status mca -n mca-system

# Monitor rollout
kubectl rollout status deployment/mca-webhook -n mca-system
```

### Rollback

```bash
# List release history
helm history mca -n mca-system

# Rollback to previous version
helm rollback mca -n mca-system

# Rollback to specific revision
helm rollback mca 2 -n mca-system

# Verify rollback
kubectl get pods -n mca-system -l app=mca-webhook
```

### Manual Upgrade

```bash
# Update image in deployment
kubectl set image deployment/mca-webhook \
  mca-webhook=ghcr.io/marxus/k8s-mca:v0.2.0 \
  -n mca-system

# Monitor rollout
kubectl rollout status deployment/mca-webhook -n mca-system

# Verify new version
kubectl get pods -n mca-system -l app=mca-webhook -o jsonpath='{.items[0].spec.containers[0].image}'
```

## Uninstallation

### Helm Uninstall

```bash
# Uninstall release
helm uninstall mca -n mca-system

# Delete namespace (optional)
kubectl delete namespace mca-system

# Verify webhook configuration removed
kubectl get mutatingwebhookconfiguration mca-webhook
# Should return "not found"
```

### Manual Uninstall

```bash
# Delete webhook configuration (important: do this first!)
kubectl delete mutatingwebhookconfiguration mca-webhook

# Delete deployment and service
kubectl delete deployment mca-webhook -n mca-system
kubectl delete service mca-webhook -n mca-system

# Delete RBAC
kubectl delete clusterrolebinding mca-webhook
kubectl delete clusterrole mca-webhook
kubectl delete serviceaccount mca-webhook -n mca-system

# Delete namespace
kubectl delete namespace mca-system
```

### Clean Up Injected Pods

```bash
# List pods with MCA injection
kubectl get pods --all-namespaces \
  -l mca.k8s.io/inject=true

# Delete injected pods (they will recreate without MCA)
kubectl delete pods -l mca.k8s.io/inject=true --all-namespaces

# Or remove label to prevent re-injection
kubectl label pods -l mca.k8s.io/inject=true \
  mca.k8s.io/inject- \
  --all-namespaces
```

## Production Considerations

### High Availability

Deploy multiple webhook replicas:

```yaml
# values.yaml
webhook:
  replicas: 3  # Or more based on cluster size

  # Add pod anti-affinity
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: mca-webhook
          topologyKey: kubernetes.io/hostname
```

### Monitoring

```yaml
# Add Prometheus annotations
template:
  metadata:
    annotations:
      prometheus.io/scrape: "true"
      prometheus.io/port: "8443"
      prometheus.io/path: "/metrics"
```

### Resource Planning

| Cluster Size | Webhook Replicas | Memory Request | CPU Request |
|--------------|-----------------|----------------|-------------|
| < 100 nodes | 1 | 64Mi | 100m |
| 100-500 nodes | 2 | 128Mi | 200m |
| 500-1000 nodes | 3 | 256Mi | 500m |
| > 1000 nodes | 5+ | 512Mi | 1000m |

### Security Hardening

```yaml
# Pod security context
securityContext:
  runAsNonRoot: true
  runAsUser: 999
  fsGroup: 999
  seccompProfile:
    type: RuntimeDefault

# Container security context
containers:
- securityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
      - ALL
```

---

For more information, see:
- [Architecture Documentation](architecture.md)
- [API Documentation](API.md)
- [GitHub Repository](https://github.com/marxus/k8s-mca)
