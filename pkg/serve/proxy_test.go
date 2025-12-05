package serve

import (
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCACertificate(t *testing.T) {
	// Setup test filesystem
	fs := afero.NewMemMapFs()
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	// Create parent directory
	err := fs.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)
	require.NoError(t, err)

	caCertPEM := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")

	err = writeCACertificate(caCertPEM)
	require.NoError(t, err)

	// Verify file was written
	content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	require.NoError(t, err)
	assert.Equal(t, caCertPEM, content)
}

func TestWriteCACertificate_Error(t *testing.T) {
	// Setup read-only filesystem
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	caCertPEM := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")

	err := writeCACertificate(caCertPEM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write CA certificate")
}

func TestWriteNamespaceFile(t *testing.T) {
	// Setup test filesystem
	fs := afero.NewMemMapFs()
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	// Create source namespace file
	err := fs.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/var/run/secrets/kubernetes.io/serviceaccount/namespace", []byte("default"), 0644)
	require.NoError(t, err)

	// Create destination directory
	err = fs.MkdirAll("/var/run/secrets/kubernetes.io/mca-serviceaccount", 0755)
	require.NoError(t, err)

	err = writeNamespaceFile()
	require.NoError(t, err)

	// Verify file was copied
	content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace")
	require.NoError(t, err)
	assert.Equal(t, []byte("default"), content)
}

func TestWriteNamespaceFile_SourceNotFound(t *testing.T) {
	// Setup test filesystem without source file
	fs := afero.NewMemMapFs()
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	err := writeNamespaceFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read namespace file")
}

func TestWriteTokenFile(t *testing.T) {
	// Setup test filesystem
	fs := afero.NewMemMapFs()
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	// Create directory
	err := fs.MkdirAll("/var/run/secrets/kubernetes.io/mca-serviceaccount", 0755)
	require.NoError(t, err)

	err = writeTokenFile()
	require.NoError(t, err)

	// Verify file was written
	content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/mca-serviceaccount/token")
	require.NoError(t, err)
	assert.Equal(t, []byte("-"), content)
}

func TestWriteTokenFile_Error(t *testing.T) {
	// Setup read-only filesystem
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	originalFS := conf.FS
	conf.FS = fs
	defer func() { conf.FS = originalFS }()

	err := writeTokenFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write token file")
}
