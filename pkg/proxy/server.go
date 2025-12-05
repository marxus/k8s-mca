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
	config, err := conf.InClusterConfig()
	if err != nil {
		return err
	}

	apiURL, err := url.Parse(config.Host)
	if err != nil {
		return err
	}

	transport, err := rest.TransportFor(config)
	if err != nil {
		return err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(apiURL)
	reverseProxy.Transport = transport

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		r.Header.Del("Authorization")
		reverseProxy.ServeHTTP(w, r)
	})

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{s.tlsCert},
	}

	server := &http.Server{
		Addr:      "127.0.0.1:6443",
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServeTLS("", "")
}
