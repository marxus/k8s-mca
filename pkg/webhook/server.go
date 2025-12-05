package webhook

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/marxus/k8s-mca/pkg/inject"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

func (s *Server) handleErr(w http.ResponseWriter, err error, message string, statusCode int) {
	log.Printf("%s: %v", message, err)
	http.Error(w, message, statusCode)
}

func (s *Server) handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.handleErr(w, err, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var admissionReview admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &admissionReview); err != nil {
		s.handleErr(w, err, "Failed to unmarshal admission review", http.StatusBadRequest)
		return
	}

	res, err := json.Marshal(s.mutate(&admissionReview))
	if err != nil {
		s.handleErr(w, err, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func (s *Server) mutateErr(uid types.UID, err error, message string) *admissionv1.AdmissionReview {
	log.Printf("%s: %v", message, err)
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:     uid,
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("%s: %v", message, err),
			},
		},
	}
}

func (s *Server) mutate(admissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionReview {
	req := admissionReview.Request

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return s.mutateErr(req.UID, err, "Failed to unmarshal pod")
	}

	mutatedPod, err := inject.ViaWebhook(pod)
	if err != nil {
		return s.mutateErr(req.UID, err, "Failed to inject MCA")
	}

	patches, err := s.generateJSONPatch(mutatedPod)
	if err != nil {
		return s.mutateErr(req.UID, err, "Failed to generate JSON patch")
	}

	log.Printf("Applied MCA injection to pod %s/%s", pod.Namespace, pod.Name)

	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:       req.UID,
			Allowed:   true,
			PatchType: &patchType,
			Patch:     patches,
		},
	}
}

func (s *Server) generateJSONPatch(mutatedPod corev1.Pod) ([]byte, error) {
	return json.Marshal([]map[string]interface{}{
		{
			"op":    "replace",
			"path":  "/spec",
			"value": mutatedPod.Spec,
		},
	})
}
