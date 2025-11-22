package inject

import (
	"strings"
	"testing"

	"github.com/lithammer/dedent"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var testPodYAML = dedent.Dedent(`
	apiVersion: v1
	kind: Pod
	metadata:
	  name: test-pod
	spec:
	  containers:
	    - name: app
	      image: nginx
	      env:
	        - name: KUBERNETES_SERVICE_HOST
	          value: "kubernetes.default.svc.cluster.local"
`)

func TestInjectMCA_Success(t *testing.T) {
	result, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("InjectMCA failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("No result returned")
	}
}

func TestInjectMCA_HasMCAInitContainer(t *testing.T) {
	result, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("InjectMCA failed: %v", err)
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal(result, &pod); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(pod.Spec.InitContainers) == 0 {
		t.Fatal("No init containers found")
	}

	if pod.Spec.InitContainers[0].Name != "mca" {
		t.Error("First init container should be 'mca'")
	}

	if pod.Spec.InitContainers[0].RestartPolicy == nil || string(*pod.Spec.InitContainers[0].RestartPolicy) != "Always" {
		t.Error("MCA init container should have restartPolicy: Always")
	}
}

func TestInjectMCA_AddsEnvVars(t *testing.T) {
	result, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("InjectMCA failed: %v", err)
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal(result, &pod); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	container := pod.Spec.Containers[0]
	found := 0
	for _, env := range container.Env {
		if (env.Name == "KUBERNETES_SERVICE_HOST" && env.Value == "127.0.0.1") ||
			(env.Name == "KUBERNETES_SERVICE_PORT" && env.Value == "6443") {
			found++
		}
	}

	if found != 2 {
		t.Error("Missing required environment variables")
	}
}

func TestInjectMCA_AddsVolumes(t *testing.T) {
	result, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("InjectMCA failed: %v", err)
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal(result, &pod); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if len(pod.Spec.Volumes) != 2 {
		t.Errorf("Expected 2 volumes, got %d", len(pod.Spec.Volumes))
	}

	volumeNames := make(map[string]bool)
	for _, vol := range pod.Spec.Volumes {
		volumeNames[vol.Name] = true
	}

	if !volumeNames["kube-api-access-sa"] || !volumeNames["kube-api-access-mca-sa"] {
		t.Error("Missing required volumes")
	}
}

func TestInjectMCA_AddsVolumeMount(t *testing.T) {
	result, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("InjectMCA failed: %v", err)
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal(result, &pod); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	container := pod.Spec.Containers[0]
	found := false
	for _, mount := range container.VolumeMounts {
		if mount.Name == "kube-api-access-mca-sa" &&
			mount.MountPath == "/var/run/secrets/kubernetes.io/serviceaccount" &&
			mount.ReadOnly {
			found = true
			break
		}
	}

	if !found {
		t.Error("Missing required volume mount")
	}
}

func TestInjectMCA_Idempotent(t *testing.T) {
	// First injection
	result1, err := InjectMCA([]byte(testPodYAML))
	if err != nil {
		t.Fatalf("First injection failed: %v", err)
	}

	// Second injection
	result2, err := InjectMCA(result1)
	if err != nil {
		t.Fatalf("Second injection failed: %v", err)
	}

	var pod1, pod2 corev1.Pod
	if err := yaml.Unmarshal(result1, &pod1); err != nil {
		t.Fatalf("Failed to parse first result: %v", err)
	}
	if err := yaml.Unmarshal(result2, &pod2); err != nil {
		t.Fatalf("Failed to parse second result: %v", err)
	}

	// Should have same number of init containers
	if len(pod1.Spec.InitContainers) != len(pod2.Spec.InitContainers) {
		t.Error("Init container count changed after second injection")
	}

	// Should have same number of volumes
	if len(pod1.Spec.Volumes) != len(pod2.Spec.Volumes) {
		t.Error("Volume count changed after second injection")
	}

	// MCA should still be first
	if pod2.Spec.InitContainers[0].Name != "mca" {
		t.Error("MCA should remain first init container")
	}
}

func TestInjectMCA_InvalidYAML(t *testing.T) {
	invalidYAML := "invalid: yaml: content: ["
	_, err := InjectMCA([]byte(invalidYAML))
	if err == nil {
		t.Error("Should fail with invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to unmarshal pod") {
		t.Errorf("Expected unmarshal error, got: %v", err)
	}
}
