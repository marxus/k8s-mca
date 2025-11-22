# MCA (Multi Cluster Adapter) Design Documentation

## Project Overview

MCA is a Multi Cluster Adapter that acts as a Man-in-the-Middle (MITM) proxy sidecar for Kubernetes API calls. It intercepts all API requests from application pods by overriding the standard `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables, allowing for request inspection, routing, and multi-cluster management.

## Architecture Vision

MCA operates as a sidecar container that:
- Intercepts all Kubernetes API calls from applications
- Acts as a transparent HTTPS proxy with custom certificate management
- Provides multi-cluster routing capabilities
- Enables comprehensive observability and monitoring of K8s API usage
- Supports production deployment automation

## Documentation Structure

### Core Architecture
- [**architecture.md**](./architecture.md) - Overall MCA architecture across all phases

### Phase Documentation

#### Phase 1: MITM Proxy (Ready for Implementation)
- [**phase-1-mitm-proxy.md**](./phase-1-mitm-proxy.md) - **FULLY DETAILED** specification for MITM proxy implementation

#### Future Phases (High-Level Designs)
- [**phase-2-multi-cluster.md**](./phase-2-multi-cluster.md) - **HIGH-LEVEL DESIGN** Multi-cluster routing & configuration
- [**phase-3-observability.md**](./phase-3-observability.md) - **HIGH-LEVEL DESIGN** Advanced features & monitoring  
- [**phase-4-production.md**](./phase-4-production.md) - **HIGH-LEVEL DESIGN** Production deployment & automation

*Note: Phases 2-4 contain conceptual designs that need detailed implementation planning.*

### Architecture Decision Records
- [**decisions/**](./decisions/) - ADRs documenting key technical choices for Phase 1

## Quick Start

### Development
```bash
# Initialize the project
go mod init github.com/your-org/k8s-mca

# Build the MCA binary
go build -o mca ./cmd/mca

# Run in proxy mode (default)
./mca

# Run Pod manifest injection
./mca --inject < pod.yaml > pod-with-mca.yaml
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## Key Features

### Phase 1 (Current Implementation)
- ✅ HTTPS MITM proxy with custom certificates
- ✅ Kubernetes sidecar pattern with `restartPolicy: Always`  
- ✅ ServiceAccount bypass and custom CA injection
- ✅ Idempotent Pod manifest injection tool
- ✅ Comprehensive test coverage

### Future Phases
- 🔄 Multi-cluster routing and load balancing
- 🔄 Advanced observability and metrics
- 🔄 Production deployment automation
- 🔄 Admission webhook controller

## Contributing

1. Read the relevant phase documentation
2. Review Architecture Decision Records in `decisions/`
3. Follow the testing patterns established in Phase 1
4. Update documentation for any architectural changes