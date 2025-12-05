// Package proxy provides an HTTP reverse proxy that intercepts Kubernetes API requests.
// It removes Authorization headers and forwards requests to configured cluster endpoints.
//
// The proxy supports multiple target clusters through a map of reverse proxy instances,
// allowing for multi-cluster API request routing.
package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
)

// Server represents an HTTPS proxy server that intercepts Kubernetes API calls.
// It removes Authorization headers and forwards requests to configured cluster endpoints.
// The server is safe for concurrent use by multiple goroutines.
type Server struct {
	tlsCert        tls.Certificate
	reverseProxies map[string]*httputil.ReverseProxy
}

// NewServer creates a new proxy server with the given TLS certificate and reverse proxies.
// The reverseProxies map must contain at least an "in-cluster" key for the default cluster.
func NewServer(tlsCert tls.Certificate, reverseProxies map[string]*httputil.ReverseProxy) *Server {
	return &Server{
		tlsCert:        tlsCert,
		reverseProxies: reverseProxies,
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	r.Header.Del("Authorization")
	s.reverseProxies["in-cluster"].ServeHTTP(w, r)
}

// Start starts the proxy server on 127.0.0.1:6443 and blocks until it exits.
// The server listens for HTTPS connections using the configured TLS certificate.
// Returns an error if the server fails to start or encounters a fatal error.
func (s *Server) Start() error {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{s.tlsCert},
	}

	server := &http.Server{
		Addr:      "127.0.0.1:6443",
		Handler:   http.HandlerFunc(s.handler),
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServeTLS("", "")
}
