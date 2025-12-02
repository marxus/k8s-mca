package serve

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

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

	_, err = clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(
		ctx,
		conf.MCAWebhook,
		types.JSONPatchType,
		[]byte(fmt.Sprintf(`[{ "op": "replace", "path": "/webhooks/0/clientConfig/caBundle", "value": "%s" }]`, base64.StdEncoding.EncodeToString(caCertPEM))),
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch mutating webhook: %w", err)
	}

	log.Printf("Patched mutating webhook: %s", conf.MCAWebhook)
	return nil
}
