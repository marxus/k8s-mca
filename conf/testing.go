//go:build !release

// Package conf/testing handles the distinction between development and testing environments.
package conf

import (
	"net"
	"testing"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

var isTestRun = testing.Testing()

func init() {
	if !isTestRun {
		initDevelop()
	} else {
		initTesting()
	}
}

func initTesting() {
	// Initialize in-memory filesystem for test isolation
	func() {
		FS = afero.NewMemMapFs()
		initFS()
	}()

	// Initialize Kubernetes config for testing context
	InClusterConfig = func() (*rest.Config, error) {
		return nil, nil
	}

	// Initialize certificate IP addresses for testing (same as release)
	CertIPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
}
