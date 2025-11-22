//go:build !release

// Package conf/develop provides configuration for development runs.
package conf

import (
	"net"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// FS provides sandboxed filesystem
	FS afero.Fs

	// InClusterConfig returns Kubernetes config using local kubeconfig
	InClusterConfig = func() func() (*rest.Config, error) {
		context := os.Getenv("K8S_MCA_CTX")
		if context == "" {
			context = "mca-develop"
		}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{CurrentContext: context},
		).ClientConfig
	}()

	// CertIPAddresses defines IP addresses included in generated TLS certificates
	CertIPAddresses = []net.IP{net.IPv4(192, 168, 5, 2)}

	// ServerAddr defines the address for the proxy server to bind to
	// Development uses 0.0.0.0:6443 to allow external access for testing
	ServerAddr = "0.0.0.0:6443"
)

func initDevelop() {
	// Initialize sandboxed filesystem at projectRoot/tmp/
	func() {
		_, filename, _, _ := runtime.Caller(0)
		projectRoot := filepath.Dir(filepath.Dir(filename))
		FS = afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(projectRoot, "tmp"))
		initFS()
	}()
}

// initFS sets up the required directory structure for ServiceAccount simulation
func initFS() {
	// Create original ServiceAccount mount point with default namespace
	realServiceAccountPath := "/var/run/secrets/kubernetes.io/serviceaccount"
	FS.MkdirAll(realServiceAccountPath, 0755)
	afero.WriteFile(FS, filepath.Join(realServiceAccountPath, "namespace"), []byte("default"), 0644)

	// Create MCA ServiceAccount mount point (populated by MCA at runtime)
	mcaServiceAccountPath := "/var/run/secrets/kubernetes.io/mca-serviceaccount"
	FS.MkdirAll(mcaServiceAccountPath, 0755)
}
