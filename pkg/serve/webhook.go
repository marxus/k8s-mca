package serve

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var mutatingConfigYAML = `
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: mca-webhook
webhooks:
  - name: mca-webhook.k8s.io
    clientConfig:
      service:
        name: mca-webhook
        namespace: default
        path: /mutate
      caBundle: <CA_BUNDLE>
    rules:
      - operations: ["CREATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    objectSelector:
      matchLabels:
        mca.k8s.io/inject: "true"
    admissionReviewVersions: ["v1", "v1beta1"]
    sideEffects: None
    failurePolicy: Fail
    reinvocationPolicy: IfNeeded
`

func StartWebhook() error {
	log.Println("Starting MCA Webhook...")

	// Generate webhook TLS certificate with service DNS names
	webhookCertDNSNames := []string{
		"mca-webhook",
		"mca-webhook.default",
		"mca-webhook.default.svc",
		"mca-webhook.default.svc.cluster.local",
	}

	tlsCert, caCertPEM, err := certs.GenerateCAAndTLSCert(webhookCertDNSNames, nil)
	if err != nil {
		return fmt.Errorf("failed to generate webhook certificates: %w", err)
	}

	// Apply mutating webhook configuration with CA bundle
	if err := applyMutatingConfig(caCertPEM); err != nil {
		return err
	}

	// Create and start webhook server
	server := webhook.NewServer(tlsCert)
	log.Println("Starting webhook server...")

	return server.Start()
}

func applyMutatingConfig(caCertPEM []byte) error {
	log.Println("Applying mutating webhook configuration...")

	// Create Kubernetes client
	config, err := conf.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx := context.Background()

	// Apply YAML directly using server-side apply
	_, err = clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(
		ctx,
		"mca-webhook",
		types.ApplyPatchType,
		[]byte(strings.ReplaceAll(mutatingConfigYAML, "<CA_BUNDLE>", base64.StdEncoding.EncodeToString(caCertPEM))),
		metav1.PatchOptions{FieldManager: "mca-webhook"},
	)
	if err != nil {
		return fmt.Errorf("failed to apply mutating webhook: %w", err)
	}

	log.Printf("Applied mutating webhook: mca-webhook")
	return nil
}
