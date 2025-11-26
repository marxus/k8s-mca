package certs

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"sync"
	"testing"

	"github.com/marxus/k8s-mca/conf"
	"github.com/stretchr/testify/assert"
)

var (
	once     sync.Once
	testData struct {
		tlsCert    tls.Certificate
		caCertPEM  []byte
		serverCert *x509.Certificate
		caCert     *x509.Certificate
	}
	testErr error
)

func setTestData() error {
	once.Do(func() {
		tlsCert, caCertPEM, err := GenerateCAAndTLSCert([]string{"localhost"}, conf.ProxyCertIPAddresses)
		if err != nil {
			testErr = err
			return
		}

		serverCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			testErr = err
			return
		}

		block, _ := pem.Decode(caCertPEM)
		if block == nil {
			testErr = err
			return
		}

		caCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			testErr = err
			return
		}

		testData.tlsCert = tlsCert
		testData.caCertPEM = caCertPEM
		testData.serverCert = serverCert
		testData.caCert = caCert
	})

	return testErr
}

func TestGenerateCAAndTLSCert_Success(t *testing.T) {
	if err := setTestData(); err != nil {
		t.Fatalf("setTestData failed: %v", err)
	}

	if len(testData.tlsCert.Certificate) == 0 {
		t.Fatal("No certificate generated")
	}

	if len(testData.caCertPEM) == 0 {
		t.Fatal("No CA certificate generated")
	}
}

func TestServerCert_HasCorrectSANIPs(t *testing.T) {
	if err := setTestData(); err != nil {
		t.Fatalf("setTestData failed: %v", err)
	}

	expectedIPs := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	
	// Check exact length
	assert.Len(t, testData.serverCert.IPAddresses, len(expectedIPs), "Certificate should have exactly %d IP addresses", len(expectedIPs))
	
	// Check each expected IP is present
	for _, expectedIP := range expectedIPs {
		found := false
		for _, certIP := range testData.serverCert.IPAddresses {
			if certIP.Equal(expectedIP) {
				found = true
				break
			}
		}
		assert.True(t, found, "Certificate should contain IP %v", expectedIP)
	}
}

func TestServerCert_HasLocalhostDNS(t *testing.T) {
	if err := setTestData(); err != nil {
		t.Fatalf("setTestData failed: %v", err)
	}

	assert.Contains(t, testData.serverCert.DNSNames, "localhost", "Certificate should contain localhost DNS name")
}

func TestCACert_IsValidCA(t *testing.T) {
	if err := setTestData(); err != nil {
		t.Fatalf("setTestData failed: %v", err)
	}

	if !testData.caCert.IsCA {
		t.Error("CA certificate is not marked as CA")
	}
}

func TestServerCert_SignedByCA(t *testing.T) {
	if err := setTestData(); err != nil {
		t.Fatalf("setTestData failed: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(testData.caCert)

	opts := x509.VerifyOptions{Roots: roots}
	_, err := testData.serverCert.Verify(opts)
	if err != nil {
		t.Errorf("Server certificate not signed by CA: %v", err)
	}
}
