package webhook

import (
	"crypto/tls"
	"net/http"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", s.handleMutate)
	mux.HandleFunc("/health", s.handleHealth)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{s.tlsCert},
	}

	server := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServeTLS("", "")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleMutate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}