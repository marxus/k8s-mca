package serve

import (
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/spf13/afero"
)

func TestWriteCACertificate(t *testing.T) {
	// Test data
	testCACert := []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert-data\n-----END CERTIFICATE-----")

	// Call the function
	err := writeCACertificate(testCACert)
	if err != nil {
		t.Fatalf("writeCACertificate failed: %v", err)
	}

	// Verify the file was written correctly
	expectedPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/ca.crt"
	content, err := afero.ReadFile(conf.FS, expectedPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(content) != string(testCACert) {
		t.Errorf("File content mismatch. Expected %q, got %q", string(testCACert), string(content))
	}
}

func TestWriteNamespaceFile(t *testing.T) {
	// Call the function
	err := writeNamespaceFile()
	if err != nil {
		t.Fatalf("writeNamespaceFile failed: %v", err)
	}

	// Verify the file was copied correctly
	expectedPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace"
	content, err := afero.ReadFile(conf.FS, expectedPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	// Verify content is not empty
	if len(content) == 0 {
		t.Error("Namespace file should not be empty")
	}
}

func TestWriteTokenFile(t *testing.T) {
	// Call the function
	err := writeTokenFile()
	if err != nil {
		t.Fatalf("writeTokenFile failed: %v", err)
	}

	// Verify the file was created
	expectedPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/token"
	content, err := afero.ReadFile(conf.FS, expectedPath)
	if err != nil {
		t.Fatalf("Failed to read token file: %v", err)
	}

	// Verify placeholder content
	if string(content) != "-" {
		t.Errorf("Token file should be empty, got %q", string(content))
	}
}
