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
	tests := []struct {
		name      string
		setupFS   func() afero.Fs
		wantErr   bool
		errMsg    string
	}{
		{
			name: "successfully writes certificate",
			setupFS: func() afero.Fs {
				fs := afero.NewMemMapFs()
				fs.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)
				return fs
			},
			wantErr: false,
		},
		{
			name: "fails on read-only filesystem",
			setupFS: func() afero.Fs {
				return afero.NewReadOnlyFs(afero.NewMemMapFs())
			},
			wantErr: true,
			errMsg:  "failed to write CA certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFS()
			originalFS := conf.FS
			conf.FS = fs
			defer func() { conf.FS = originalFS }()

			caCertPEM := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
			err := writeCACertificate(caCertPEM)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
				require.NoError(t, err)
				assert.Equal(t, caCertPEM, content)
			}
		})
	}
}

func TestWriteNamespaceFile(t *testing.T) {
	tests := []struct {
		name    string
		setupFS func() afero.Fs
		wantErr bool
		errMsg  string
	}{
		{
			name: "successfully copies namespace file",
			setupFS: func() afero.Fs {
				fs := afero.NewMemMapFs()
				fs.MkdirAll("/var/run/secrets/kubernetes.io/serviceaccount", 0755)
				afero.WriteFile(fs, "/var/run/secrets/kubernetes.io/serviceaccount/namespace", []byte("default"), 0644)
				fs.MkdirAll("/var/run/secrets/kubernetes.io/mca-serviceaccount", 0755)
				return fs
			},
			wantErr: false,
		},
		{
			name: "fails when source file not found",
			setupFS: func() afero.Fs {
				return afero.NewMemMapFs()
			},
			wantErr: true,
			errMsg:  "failed to read namespace file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFS()
			originalFS := conf.FS
			conf.FS = fs
			defer func() { conf.FS = originalFS }()

			err := writeNamespaceFile()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/mca-serviceaccount/namespace")
				require.NoError(t, err)
				assert.Equal(t, []byte("default"), content)
			}
		})
	}
}

func TestWriteTokenFile(t *testing.T) {
	tests := []struct {
		name    string
		setupFS func() afero.Fs
		wantErr bool
		errMsg  string
	}{
		{
			name: "successfully writes token file",
			setupFS: func() afero.Fs {
				fs := afero.NewMemMapFs()
				fs.MkdirAll("/var/run/secrets/kubernetes.io/mca-serviceaccount", 0755)
				return fs
			},
			wantErr: false,
		},
		{
			name: "fails on read-only filesystem",
			setupFS: func() afero.Fs {
				return afero.NewReadOnlyFs(afero.NewMemMapFs())
			},
			wantErr: true,
			errMsg:  "failed to write token file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFS()
			originalFS := conf.FS
			conf.FS = fs
			defer func() { conf.FS = originalFS }()

			err := writeTokenFile()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				content, err := afero.ReadFile(fs, "/var/run/secrets/kubernetes.io/mca-serviceaccount/token")
				require.NoError(t, err)
				assert.Equal(t, []byte("-"), content)
			}
		})
	}
}
