package webhook

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestServer_handleHealth(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %q", w.Body.String())
	}
}

func TestServer_handleMutate_InvalidJSON(t *testing.T) {
	server := &Server{}
	invalidJSON := "invalid json"
	req := httptest.NewRequest("POST", "/mutate", bytes.NewBufferString(invalidJSON))
	w := httptest.NewRecorder()

	server.handleMutate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestServer_mutate_NonPodResource(t *testing.T) {
	server := &Server{}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-configmap",
		},
	}
	configMapBytes, _ := json.Marshal(configMap)

	admissionReview := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID("test-uid"),
			Kind: metav1.GroupVersionKind{
				Kind: "ConfigMap",
			},
			Object: runtime.RawExtension{Raw: configMapBytes},
		},
	}

	response := server.mutate(admissionReview)

	if !response.Response.Allowed {
		t.Error("Non-pod resources should be allowed")
	}

	if response.Response.UID != types.UID("test-uid") {
		t.Error("UID should be preserved in response")
	}
}

func TestServer_mutate_ValidPod(t *testing.T) {
	server := &Server{}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx"},
			},
		},
	}

	podBytes, _ := json.Marshal(pod)

	admissionReview := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID("test-uid"),
			Kind: metav1.GroupVersionKind{
				Kind: "Pod",
			},
			Object: runtime.RawExtension{Raw: podBytes},
		},
	}

	response := server.mutate(admissionReview)

	if !response.Response.Allowed {
		t.Error("Valid pods should be allowed")
	}

	if response.Response.PatchType == nil {
		t.Error("Patch should be applied to valid pods")
	}

	if *response.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Error("Patch type should be JSONPatch")
	}
}

func TestServer_mutate_MalformedJSON(t *testing.T) {
	server := &Server{}

	malformedJSON := []byte(`{invalid json}`)

	admissionReview := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID("test-uid"),
			Kind: metav1.GroupVersionKind{
				Kind: "Pod",
			},
			Object: runtime.RawExtension{Raw: malformedJSON},
		},
	}

	response := server.mutate(admissionReview)

	if response.Response.Allowed {
		t.Error("Malformed JSON should not be allowed")
	}

	if response.Response.Result == nil {
		t.Error("Error response should have result")
	}

	if response.Response.UID != types.UID("test-uid") {
		t.Error("UID should be preserved in error response")
	}
}

func TestServer_createErrorResponse(t *testing.T) {
	server := &Server{}
	testUID := types.UID("test-uid")
	testErr := fmt.Errorf("test error")
	message := "Test message"

	response := server.mutateErr(testUID, testErr, message)

	if response.Response.UID != testUID {
		t.Error("UID should be preserved")
	}

	if response.Response.Allowed {
		t.Error("Error response should not be allowed")
	}

	if response.Response.Result == nil {
		t.Error("Error response should have result")
	}

	expectedMessage := "Test message: test error"
	if response.Response.Result.Message != expectedMessage {
		t.Errorf("Expected message %q, got %q", expectedMessage, response.Response.Result.Message)
	}

	if response.TypeMeta.APIVersion != "admission.k8s.io/v1" {
		t.Error("APIVersion should be admission.k8s.io/v1")
	}

	if response.TypeMeta.Kind != "AdmissionReview" {
		t.Error("Kind should be AdmissionReview")
	}
}

func TestServer_generateJSONPatch(t *testing.T) {
	server := &Server{}

	mutatedPod := corev1.Pod{
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: func() *bool { b := false; return &b }(),
			InitContainers: []corev1.Container{
				{Name: "mca", Image: "mca:latest"},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
					Env: []corev1.EnvVar{
						{Name: "KUBERNETES_SERVICE_HOST", Value: "127.0.0.1"},
						{Name: "KUBERNETES_SERVICE_PORT", Value: "6443"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "kube-api-access-mca-sa", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount"},
					},
				},
			},
			Volumes: []corev1.Volume{
				{Name: "kube-api-access-sa"},
				{Name: "kube-api-access-mca-sa"},
			},
		},
	}

	patches, err := server.generateJSONPatch(mutatedPod)
	if err != nil {
		t.Fatalf("generateJSONPatch failed: %v", err)
	}

	var patchList []map[string]interface{}
	if err := json.Unmarshal(patches, &patchList); err != nil {
		t.Fatalf("Failed to unmarshal patches: %v", err)
	}

	if len(patchList) == 0 {
		t.Error("Patches should not be empty")
	}

	found := false
	for _, patch := range patchList {
		if patch["op"] == "replace" && patch["path"] == "/spec" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should contain a patch that replaces /spec")
	}
}

func TestNewServer(t *testing.T) {
	cert := tls.Certificate{}
	server := NewServer(cert)

	if server == nil {
		t.Error("NewServer should return a server instance")
	}

	if server.tlsCert.Certificate == nil && cert.Certificate == nil {
	} else if len(server.tlsCert.Certificate) != len(cert.Certificate) {
		t.Error("TLS certificate should be preserved")
	}
}
