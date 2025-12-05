# Certificate Generation Tests

This document describes the test suite for the certificate generation package (`pkg/certs`).

## Overview

The certificate generation package provides functionality to generate self-signed Certificate Authority (CA) certificates and TLS server certificates for secure communication. The tests ensure that certificates are generated correctly with proper attributes, validity periods, and cryptographic properties.

## Test Coverage

- **TestGenerateCAAndTLSCert_Basic** - validates that certificates are generated successfully and contain valid TLS certificate chain and CA certificate PEM
- **TestGenerateCAAndTLSCert_CACertificateValid** - validates CA certificate has correct properties including IsCA flag, CommonName, Organization, and KeyUsage
- **TestGenerateCAAndTLSCert_ServerCertificateValid** - validates server certificate has correct CommonName, Organization, KeyUsage, and ExtKeyUsage for server authentication
- **TestGenerateCAAndTLSCert_DNSNames** - validates that all provided DNS names (including wildcards) are correctly included in the certificate SANs
- **TestGenerateCAAndTLSCert_IPAddresses** - validates that all provided IP addresses (IPv4 and IPv6) are correctly included in the certificate SANs
- **TestGenerateCAAndTLSCert_EmptyDNSAndIP** - validates that certificate generation succeeds with nil/empty DNS names and IP addresses without errors
- **TestGenerateCAAndTLSCert_CertificateChain** - validates that the server certificate can be verified against the CA certificate forming a valid chain
- **TestGenerateCAAndTLSCert_ValidityPeriod** - validates that CA and server certificates have appropriate NotBefore and NotAfter dates (~1 year validity)
- **TestGenerateCAAndTLSCert_TLSUsable** - validates that the generated TLS certificate can be used in a tls.Config with private key and certificate chain
- **TestGenerateCAAndTLSCert_SerialNumbers** - validates that CA and server certificates have unique serial numbers
