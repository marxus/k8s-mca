package serve

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/marxus/k8s-mca/conf"
	"github.com/marxus/k8s-mca/pkg/certs"
	"github.com/marxus/k8s-mca/pkg/webhook"
	"github.com/spf13/afero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func StartWebhook() error {
	log.Println("Starting MCA Webhook...")

	namespace, err := afero.ReadFile(conf.FS, "/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return fmt.Errorf("failed to read namespace file: %w", err)
	}

	tlsCert, caCertPEM, err := certs.GenerateCAAndTLSCert([]string{fmt.Sprintf("%s.%s.svc", conf.WebhookName, namespace)}, nil)
	if err != nil {
		return fmt.Errorf("failed to generate webhook certificates: %w", err)
	}

	clientset, err := buildKubernetesClient()
	if err != nil {
		return err
	}

	if err := patchMutatingConfig(caCertPEM, clientset); err != nil {
		return err
	}

	server := webhook.NewServer(tlsCert)
	log.Println("Starting webhook server...")

	return server.Start()
}

func buildKubernetesClient() (kubernetes.Interface, error) {
	config, err := conf.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}

func buildWebhookPatch(caCertPEM []byte) []byte {
	return []byte(fmt.Sprintf(
		`[{ "op": "replace", "path": "/webhooks/0/clientConfig/caBundle", "value": "%s" }]`,
		base64.StdEncoding.EncodeToString(caCertPEM),
	))
}

func patchMutatingConfig(caCertPEM []byte, clientset kubernetes.Interface) error {
	log.Println("Applying mutating webhook configuration...")

	ctx := context.Background()
	patch := buildWebhookPatch(caCertPEM)

	_, err := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(
		ctx,
		conf.WebhookName,
		types.JSONPatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch mutating webhook: %w", err)
	}

	log.Printf("Patched mutating webhook: %s", conf.WebhookName)
	return nil
}
