# Pod Injection Tests

This document describes the test suite for the pod injection package (`pkg/inject`).

## Overview

The injection package provides functionality to inject the MCA proxy sidecar into Kubernetes pods. It modifies pod specifications by adding an init container, updating volume mounts, and setting environment variables to redirect Kubernetes API traffic through the proxy.

## Test Coverage

### ViaCLI Function
- **TestViaCLI_ValidPod** - validates that valid pod YAML is successfully unmarshaled, injected, and remarshaled with the mca-proxy init container
- **TestViaCLI_InvalidYAML** - validates that invalid YAML returns an error with appropriate error message

### ViaWebhook Function
- **TestViaWebhook_BasicPod** - validates that webhook injection adds mca-proxy init container with correct image and configuration

### injectProxy Function
- **TestInjectProxy_AddsProxyInitContainer** - validates that mca-proxy init container is added with correct name, image, args, and security context
- **TestInjectProxy_PreservesExistingProxyContainer** - validates that existing mca-proxy container is preserved with its custom configuration
- **TestInjectProxy_PreservesOtherInitContainers** - validates that other init containers are preserved and mca-proxy is placed first in the order
- **TestInjectProxy_UpdatesVolumeMountAndAddsEnvVars** - validates that serviceaccount volume mount is renamed and KUBERNETES_SERVICE_HOST/PORT env vars are added
- **TestInjectProxy_DoesNotUpdateContainerWithoutServiceAccountMount** - validates that containers without serviceaccount mounts are not modified
- **TestInjectProxy_AddsRequiredVolume** - validates that kube-api-access-mca-sa emptyDir volume is added to the pod
- **TestInjectProxy_DoesNotDuplicateVolume** - validates that kube-api-access-mca-sa volume is not added if it already exists
- **TestInjectProxy_MultipleContainersWithMixedVolumeMounts** - validates that only containers with serviceaccount mounts are updated while others remain unchanged

### addVolumeMount Function
- **TestAddVolumeMount_UpdatesExistingMount** - validates that serviceaccount volume mount name is changed to kube-api-access-mca-sa and returns true
- **TestAddVolumeMount_ReturnsFalseWhenNoMatch** - validates that function returns false when no serviceaccount mount is found
- **TestAddVolumeMount_HandlesEmptyVolumeMounts** - validates that empty volume mounts array is handled gracefully without errors

### addEnvVars Function
- **TestAddEnvVars_AddsNewEnvVars** - validates that KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT env vars are added to container
- **TestAddEnvVars_UpdatesExistingEnvVars** - validates that existing KUBERNETES_SERVICE_HOST/PORT env vars are updated with new values
- **TestAddEnvVars_PreservesOtherEnvVars** - validates that other existing env vars are preserved when adding/updating Kubernetes env vars

### addRequiredVolume Function
- **TestAddRequiredVolume_AddsVolumeWhenMissing** - validates that kube-api-access-mca-sa emptyDir volume is added when not present
- **TestAddRequiredVolume_DoesNotAddDuplicateVolume** - validates that volume is not duplicated if kube-api-access-mca-sa already exists
- **TestAddRequiredVolume_PreservesExistingVolumes** - validates that existing volumes are preserved when adding kube-api-access-mca-sa volume
