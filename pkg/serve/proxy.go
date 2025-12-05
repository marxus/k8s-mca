// Package serve provides high-level functions for starting the MCA proxy and webhook servers.
// It handles certificate generation, Kubernetes client creation, and service account credential management.
package serve

import (
	"fmt"
	"log"
	"net"
	"net/http/httputil"
	"net/url"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/proxy"
	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

// StartProxy starts the MCA proxy server with service account credential management.
// It generates TLS certificates, writes CA certificate and service account files,
// creates reverse proxies for the Kubernetes API, and starts the proxy server.
//
// Returns an error if certificate generation fails, file writing fails,
// reverse proxy creation fails, or server startup fails.
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

	reverseProxies, err := buildReverseProxies()
	if err != nil {
		return err
	}

	server := proxy.NewServer(tlsCert, reverseProxies)
	log.Println("Starting proxy server...")

	return server.Start()
}

func buildReverseProxies() (map[string]*httputil.ReverseProxy, error) {
	config, err := conf.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	apiURL, err := url.Parse(config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL: %w", err)
	}

	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(apiURL)
	reverseProxy.Transport = transport

	return map[string]*httputil.ReverseProxy{
		"in-cluster": reverseProxy,
	}, nil
}

func writeCACertificate(caCertPEM []byte) error {
	mcaCACertPath := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	if err := afero.WriteFile(conf.FS, mcaCACertPath, caCertPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	log.Printf("CA certificate saved to: %s", mcaCACertPath)
	return nil
}

func writeNamespaceFile() error {
	namespacePath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	mcaNamespacePath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace"

	namespace, err := afero.ReadFile(conf.FS, namespacePath)
	if err != nil {
		return fmt.Errorf("failed to read namespace file: %w", err)
	}

	if err := afero.WriteFile(conf.FS, mcaNamespacePath, namespace, 0644); err != nil {
		return fmt.Errorf("failed to write namespace file: %w", err)
	}

	log.Printf("Namespace file copied to: %s", mcaNamespacePath)
	return nil
}

func writeTokenFile() error {
	mcaTokenPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount/token"

	if err := afero.WriteFile(conf.FS, mcaTokenPath, []byte("-"), 0644); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	log.Printf("Placeholder token file created at: %s", mcaTokenPath)
	return nil
}
