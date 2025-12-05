package serve

import (
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/spf13/afero"
)

func TestWriteCACertificate(t *testing.T) {
	testCACert := []byte("-----BEGIN CERTIFICATE-----\ntest-ca-cert-data\n-----END CERTIFICATE-----")

	err := writeCACertificate(testCACert)
	if err != nil {
		t.Fatalf("writeCACertificate failed: %v", err)
	}

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
	err := writeNamespaceFile()
	if err != nil {
		t.Fatalf("writeNamespaceFile failed: %v", err)
	}

	expectedPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace"
	content, err := afero.ReadFile(conf.FS, expectedPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Namespace file should not be empty")
	}
}

func TestWriteTokenFile(t *testing.T) {
	err := writeTokenFile()
	if err != nil {
		t.Fatalf("writeTokenFile failed: %v", err)
	}

	expectedPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/token"
	content, err := afero.ReadFile(conf.FS, expectedPath)
	if err != nil {
		t.Fatalf("Failed to read token file: %v", err)
	}

	if string(content) != "-" {
		t.Errorf("Token file should be empty, got %q", string(content))
	}
}
