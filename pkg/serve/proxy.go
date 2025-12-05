package serve

import (
	"fmt"
	"log"
	"net"
	"path/filepath"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/proxy"
	"github.com/spf13/afero"
)

func StartProxy() error {
	log.Println("Starting MCA Proxy...")

	tlsCert, caCertPEM, err := certs.GenerateCAAndTLSCert([]string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback})
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	if err := writeCACertificate(caCertPEM); err != nil {
		return err
	}

	if err := writeNamespaceFile(); err != nil {
		return err
	}

	if err := writeTokenFile(); err != nil {
		return err
	}

	server := proxy.NewServer(tlsCert)
	log.Println("Starting proxy server...")

	return server.Start()
}

func writeCACertificate(caCertPEM []byte) error {
	mcaCACertPath := filepath.Join(conf.ServiceAccountPath, "ca.crt")
	if err := afero.WriteFile(conf.FS, mcaCACertPath, caCertPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	log.Printf("CA certificate saved to: %s", mcaCACertPath)
	return nil
}

func writeNamespaceFile() error {
	namespacePath := filepath.Join(conf.ServiceAccountPath, "namespace")
	mcaNamespacePath := filepath.Join(conf.MCAServiceAccountPath, "namespace")

	namespace, err := afero.ReadFile(conf.FS, namespacePath)
	if err != nil {
		return fmt.Errorf("failed to read namespace file: %w (hint: ensure 'automountServiceAccountToken: true')", err)
	}

	if err := afero.WriteFile(conf.FS, mcaNamespacePath, namespace, 0644); err != nil {
		return fmt.Errorf("failed to write namespace file: %w", err)
	}

	log.Printf("Namespace file copied to: %s", mcaNamespacePath)
	return nil
}

func writeTokenFile() error {
	mcaTokenPath := filepath.Join(conf.MCAServiceAccountPath, "token")

	if err := afero.WriteFile(conf.FS, mcaTokenPath, []byte("-"), 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	log.Printf("Placeholder token file created at: %s", mcaTokenPath)
	return nil
}
