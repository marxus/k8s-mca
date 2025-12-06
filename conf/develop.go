//go:build !release

package conf

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	FS afero.Fs

	InClusterConfig func() (*rest.Config, error)

	ProxyImage = "mca:latest"

	WebhookName = "mca-webhook"

	PodNamespace = "default"
)

func initDevelop() {
	FS = func() afero.Fs {
		_, filename, _, _ := runtime.Caller(0)
		projectRoot := filepath.Dir(filepath.Dir(filename))
		return afero.NewBasePathFs(afero.NewOsFs(), filepath.Join(projectRoot, "tmp"))
	}()
	initFS()

	InClusterConfig = func() func() (*rest.Config, error) {
		context := os.Getenv("MCA_K8S_CTX")
		if context == "" {
			context = "mca-k8s-ctx"
		}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{CurrentContext: context},
		).ClientConfig
	}()
}

func initFS() {
	FS.MkdirAll("/var/run/secrets/kubernetes.io/mca-serviceaccount", 0755)
}
