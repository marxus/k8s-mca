//go:build release

// Package conf/release provides configuration for release (production) builds.
package conf

import (
	"net"
	"os"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

// FS provides direct access to the real filesystem for production use
var FS = afero.NewOsFs()

// InClusterConfig returns Kubernetes config from in-cluster service account
var InClusterConfig = rest.InClusterConfig

// ProxyCertIPAddresses defines IP addresses included in generated proxy TLS certificates for production
// Includes localhost (127.0.0.1) and IPv6 loopback (::1) for sidecar proxy
var ProxyCertIPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}

// ProxyServerAddr defines the address for the proxy server to bind to
// Production uses 127.0.0.1:6443 for sidecar deployment (localhost-only access)
var ProxyServerAddr = "127.0.0.1:6443"

var ProxyImage = os.Getenv("MCA_PROXY_IMAGE")

var WebhookName = os.Getenv("MCA_WEBHOOK_NAME")
