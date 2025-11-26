package serve

import (
	"fmt"
	"log"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/proxy"
	"github.com/spf13/afero"
)

func StartProxy() error {
	log.Println("Starting MCA Proxy...")

	// Generate TLS certificate and CA
	tlsCert, caCertPEM, err := certs.GenerateCAAndTLSCert([]string{"localhost"}, conf.ProxyCertIPAddresses)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Save CA certificate to the shared volume location
	if err := writeCACertificate(caCertPEM); err != nil {
		return err
	}

	// Copy namespace file to the shared volume location
	if err := writeNamespaceFile(); err != nil {
		return err
	}

	// Create placeholder token file
	if err := writeTokenFile(); err != nil {
		return err
	}

	// Create and start proxy server
	server := proxy.NewServer(tlsCert)
	log.Println("Starting proxy server...")

	return server.Start()
}

func writeCACertificate(caCertPEM []byte) error {
	caCertPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/ca.crt"
	if err := afero.WriteFile(conf.FS, caCertPath, caCertPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	log.Printf("CA certificate saved to: %s", caCertPath)
	return nil
}

func writeNamespaceFile() error {
	realNamespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	mcaNamespacePath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace"

	namespaceContent, err := afero.ReadFile(conf.FS, realNamespacePath)
	if err != nil {
		return fmt.Errorf("failed to read namespace file: %w (hint: ensure automountServiceAccountToken is set to true in the pod manifest - MCA respects this flag to project the ServiceAccount)", err)
	}

	if err := afero.WriteFile(conf.FS, mcaNamespacePath, namespaceContent, 0644); err != nil {
		return fmt.Errorf("failed to write namespace file: %w", err)
	}

	log.Printf("Namespace file copied to: %s", mcaNamespacePath)
	return nil
}

func writeTokenFile() error {
	tokenPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/token"

	if err := afero.WriteFile(conf.FS, tokenPath, []byte("-"), 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	log.Printf("Placeholder token file created at: %s", tokenPath)
	return nil
}