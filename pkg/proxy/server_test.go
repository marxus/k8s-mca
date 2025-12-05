// Package proxy tests the HTTP reverse proxy server behavior.
package proxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	cert := tls.Certificate{
		Certificate: [][]byte{{1, 2, 3}},
	}
	reverseProxies := map[string]*httputil.ReverseProxy{
		"in-cluster": {},
	}

	server := NewServer(cert, reverseProxies)

	require.NotNil(t, server)
	assert.Equal(t, cert, server.tlsCert)
	assert.Equal(t, reverseProxies, server.reverseProxies)
}

func TestServer_Handler_RemovesAuthorizationHeader(t *testing.T) {
	// Create a mock backend server
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	// Create reverse proxy pointing to backend
	backendURL, err := url.Parse(backend.URL)
	require.NoError(t, err)
	reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Create server with the reverse proxy
	cert := tls.Certificate{}
	reverseProxies := map[string]*httputil.ReverseProxy{
		"in-cluster": reverseProxy,
	}
	server := NewServer(cert, reverseProxies)

	// Create test request with Authorization header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("X-Custom-Header", "custom-value")

	// Record response
	recorder := httptest.NewRecorder()
	server.handler(recorder, req)

	// Verify Authorization header was removed
	assert.Empty(t, receivedHeaders.Get("Authorization"))

	// Verify other headers are preserved
	assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))
}

func TestServer_Handler_ForwardsRequestToBackend(t *testing.T) {
	backendCalled := false
	var receivedMethod, receivedPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	require.NoError(t, err)
	reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)

	cert := tls.Certificate{}
	reverseProxies := map[string]*httputil.ReverseProxy{
		"in-cluster": reverseProxy,
	}
	server := NewServer(cert, reverseProxies)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces/default/pods", nil)
	recorder := httptest.NewRecorder()
	server.handler(recorder, req)

	assert.True(t, backendCalled)
	assert.Equal(t, http.MethodPost, receivedMethod)
	assert.Equal(t, "/api/v1/namespaces/default/pods", receivedPath)
	assert.Equal(t, "backend response", recorder.Body.String())
}

func TestServer_Handler_ForwardsResponseStatusAndBody(t *testing.T) {
	responseBody := `{"items":[]}`
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(responseBody))
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	require.NoError(t, err)
	reverseProxy := httputil.NewSingleHostReverseProxy(backendURL)

	cert := tls.Certificate{}
	reverseProxies := map[string]*httputil.ReverseProxy{
		"in-cluster": reverseProxy,
	}
	server := NewServer(cert, reverseProxies)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil)
	recorder := httptest.NewRecorder()
	server.handler(recorder, req)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, responseBody, recorder.Body.String())
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}
