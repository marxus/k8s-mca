//go:build !release

package conf

import (
	"testing"

	"github.com/spf13/afero"
	"k8s.io/client-go/rest"
)

var isTestRun = testing.Testing()

func init() {
	if !isTestRun {
		initDevelop()
	} else {
		initTesting()
	}
}

func initTesting() {
	FS = afero.NewMemMapFs()
	initFS()
	
	InClusterConfig = func() (*rest.Config, error) {
		return nil, nil
	}
}
