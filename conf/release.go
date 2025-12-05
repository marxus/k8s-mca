//go:build release

package conf

import (
	"os"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

var FS = afero.NewOsFs()

var InClusterConfig = rest.InClusterConfig

var ProxyImage = os.Getenv("MCA_PROXY_IMAGE")

var WebhookName = os.Getenv("MCA_WEBHOOK_NAME")
