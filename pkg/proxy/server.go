package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/marxus/k8s-mca/conf"
	"k8s.io/client-go/rest"
)

type Server struct {
	tlsCert tls.Certificate
}

func NewServer(tlsCert tls.Certificate) *Server {
	return &Server{
		tlsCert: tlsCert,
	}
}

func (s *Server) Start() error {
	// Get Kubernetes API server URL from in-cluster config
	config, err := conf.InClusterConfig()
	if err != nil {
		return err
	}

	// Parse the API server URL
	apiURL, err := url.Parse(config.Host)
	if err != nil {
		return err
	}

	// Create transport for the reverse proxy
	transport, err := rest.TransportFor(config)
	if err != nil {
		return err
	}

	// Create reverse proxy
	reverseProxy := httputil.NewSingleHostReverseProxy(apiURL)
	reverseProxy.Transport = transport

	// Create handler that removes Authorization header
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying request: %s %s", r.Method, r.URL.Path)
		r.Header.Del("Authorization")
		reverseProxy.ServeHTTP(w, r)
	})

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{s.tlsCert},
	}

	server := &http.Server{
		Addr:      conf.ProxyServerAddr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServeTLS("", "")
}
