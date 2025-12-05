package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
)

type Server struct {
	tlsCert        tls.Certificate
	reverseProxies map[string]*httputil.ReverseProxy
}

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
