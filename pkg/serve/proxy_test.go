// Package serve tests high-level server startup and file operations.
package serve

import (
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCACertificate(t *testing.T) {
	defer conf.FS.Remove("/var/run/secrets/kubernetes.io/mca-serviceaccount/ca.crt")

	caCertPEM := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	err := writeCACertificate(caCertPEM)
	require.NoError(t, err)

	content, err := afero.ReadFile(conf.FS, "/var/run/secrets/kubernetes.io/mca-serviceaccount/ca.crt")
	require.NoError(t, err)
	assert.Equal(t, caCertPEM, content)
}

func TestWriteNamespaceFile(t *testing.T) {
	defer conf.FS.Remove("/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace")

	err := writeNamespaceFile()
	require.NoError(t, err)

	content, err := afero.ReadFile(conf.FS, "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace")
	require.NoError(t, err)
	assert.Equal(t, []byte("default"), content)
}

func TestWriteTokenFile(t *testing.T) {
	defer conf.FS.Remove("/var/run/secrets/kubernetes.io/mca-serviceaccount/token")

	err := writeTokenFile()
	require.NoError(t, err)

	content, err := afero.ReadFile(conf.FS, "/var/run/secrets/kubernetes.io/mca-serviceaccount/token")
	require.NoError(t, err)
	assert.Equal(t, []byte("-"), content)
}
