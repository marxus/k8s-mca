package serve

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestMutatingConfigYAMLStructure(t *testing.T) {
	// Test that the YAML contains expected fields
	expectedFields := []string{
		"apiVersion: admissionregistration.k8s.io/v1",
		"kind: MutatingWebhookConfiguration",
		"name: mca-webhook",
		"mca-webhook.k8s.io",
		"service:",
		"name: mca-webhook",
		"namespace: default",
		"path: /mutate",
		"caBundle: <CA_BUNDLE>",
		"operations: [\"CREATE\"]",
		"resources: [\"pods\"]",
		"reinvocationPolicy: IfNeeded",
		"failurePolicy: Fail",
	}

	for _, field := range expectedFields {
		if !strings.Contains(mutatingConfigYAML, field) {
			t.Errorf("Expected field %q not found in mutatingConfigYAML", field)
		}
	}
}

func TestMutatingConfigYAMLPlaceholder(t *testing.T) {
	// Test that CA_BUNDLE placeholder exists
	if !strings.Contains(mutatingConfigYAML, "<CA_BUNDLE>") {
		t.Error("CA_BUNDLE placeholder not found in mutatingConfigYAML")
	}

	// Test that object selector is present
	if !strings.Contains(mutatingConfigYAML, "mca.k8s.io/inject") {
		t.Error("Object selector for mca.k8s.io/inject not found")
	}
}

func TestCABundleReplacement(t *testing.T) {
	// Test CA bundle replacement logic (without actual K8s client)
	testCA := []byte("test-ca-certificate")
	expectedB64 := base64.StdEncoding.EncodeToString(testCA)

	result := strings.ReplaceAll(mutatingConfigYAML, "<CA_BUNDLE>", expectedB64)

	// Should not contain placeholder anymore
	if strings.Contains(result, "<CA_BUNDLE>") {
		t.Error("CA_BUNDLE placeholder still present after replacement")
	}

	// Should contain the base64 encoded CA
	if !strings.Contains(result, expectedB64) {
		t.Error("Base64 encoded CA not found in result")
	}
}