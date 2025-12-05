// Package certs tests certificate generation with various DNS names and IP addresses.
package certs

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCAAndTLSCert_Basic(t *testing.T) {
	dnsNames := []string{"localhost", "example.com"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}

	tlsCert, caCertPEM, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)

	require.NoError(t, err)
	assert.NotEmpty(t, tlsCert.Certificate)
	assert.NotEmpty(t, caCertPEM)
}

func TestGenerateCAAndTLSCert_CACertificateValid(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	_, caCertPEM, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	block, _ := pem.Decode(caCertPEM)
	require.NotNil(t, block, "Failed to decode CA certificate PEM")
	assert.Equal(t, "CERTIFICATE", block.Type)

	caCert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	assert.True(t, caCert.IsCA, "CA certificate IsCA flag should be set")
	assert.Equal(t, "MCA CA", caCert.Subject.CommonName)
	require.NotEmpty(t, caCert.Subject.Organization)
	assert.Equal(t, "MCA", caCert.Subject.Organization[0])

	expectedKeyUsage := x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	assert.Equal(t, expectedKeyUsage, caCert.KeyUsage&expectedKeyUsage, "CA certificate has incorrect key usage")
}

func TestGenerateCAAndTLSCert_ServerCertificateValid(t *testing.T) {
	dnsNames := []string{"localhost", "example.com"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}

	tlsCert, _, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	assert.Equal(t, "localhost", serverCert.Subject.CommonName)
	require.NotEmpty(t, serverCert.Subject.Organization)
	assert.Equal(t, "MCA", serverCert.Subject.Organization[0])

	expectedKeyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	assert.Equal(t, expectedKeyUsage, serverCert.KeyUsage&expectedKeyUsage, "Server certificate has incorrect key usage")

	require.NotEmpty(t, serverCert.ExtKeyUsage)
	assert.Equal(t, x509.ExtKeyUsageServerAuth, serverCert.ExtKeyUsage[0])
}

func TestGenerateCAAndTLSCert_DNSNames(t *testing.T) {
	dnsNames := []string{"localhost", "example.com", "*.example.org"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	tlsCert, _, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	assert.Equal(t, dnsNames, serverCert.DNSNames)
}

func TestGenerateCAAndTLSCert_IPAddresses(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{
		net.IPv4(127, 0, 0, 1),
		net.IPv6loopback,
		net.ParseIP("192.168.1.1"),
	}

	tlsCert, _, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	require.Len(t, serverCert.IPAddresses, len(ipAddresses))
	for i, expected := range ipAddresses {
		assert.True(t, serverCert.IPAddresses[i].Equal(expected),
			"IP address at index %d: expected '%s', got '%s'", i, expected, serverCert.IPAddresses[i])
	}
}

func TestGenerateCAAndTLSCert_EmptyDNSAndIP(t *testing.T) {
	tlsCert, caCertPEM, err := GenerateCAAndTLSCert(nil, nil)
	require.NoError(t, err)

	assert.NotEmpty(t, tlsCert.Certificate)
	assert.NotEmpty(t, caCertPEM)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	assert.Empty(t, serverCert.DNSNames)
	assert.Empty(t, serverCert.IPAddresses)
}

func TestGenerateCAAndTLSCert_CertificateChain(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	tlsCert, caCertPEM, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	block, _ := pem.Decode(caCertPEM)
	require.NotNil(t, block)

	caCert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		DNSName:   "localhost",
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	_, err = serverCert.Verify(opts)
	assert.NoError(t, err, "Server certificate should be verifiable against CA")
}

func TestGenerateCAAndTLSCert_ValidityPeriod(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	tlsCert, caCertPEM, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	block, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	now := time.Now()
	oneYearLater := now.Add(365 * 24 * time.Hour)

	assert.False(t, caCert.NotBefore.After(now), "CA certificate NotBefore should not be in the future")
	assert.False(t, caCert.NotAfter.Before(oneYearLater.Add(-1*time.Hour)), "CA certificate should be valid for ~1 year")

	assert.False(t, serverCert.NotBefore.After(now), "Server certificate NotBefore should not be in the future")
	assert.False(t, serverCert.NotAfter.Before(oneYearLater.Add(-1*time.Hour)), "Server certificate should be valid for ~1 year")
}

func TestGenerateCAAndTLSCert_TLSUsable(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	tlsCert, _, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	config := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	require.NotEmpty(t, config.Certificates)

	cert := config.Certificates[0]
	assert.NotNil(t, cert.PrivateKey)
	assert.NotEmpty(t, cert.Certificate)
}

func TestGenerateCAAndTLSCert_SerialNumbers(t *testing.T) {
	dnsNames := []string{"localhost"}
	ipAddresses := []net.IP{net.IPv4(127, 0, 0, 1)}

	tlsCert, caCertPEM, err := GenerateCAAndTLSCert(dnsNames, ipAddresses)
	require.NoError(t, err)

	block, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err)

	assert.NotEqual(t, caCert.SerialNumber, serverCert.SerialNumber, "CA and server certificates should have different serial numbers")
}
