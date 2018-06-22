package e2e

import (
	"flag"
	"os"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/qiniu-ava/mxnet-operator/test/e2e/framework"
	"github.com/sirupsen/logrus"
)

var fw *framework.Framework

func TestMain(m *testing.M) {
	// use KUBERNETES_CONFIG of sdk to set config
	//kubeconfig := flag.String("kubeconfig", "~/.kube/config", "kube config path, e.g. $HOME/.kube/config")
	opImage := flag.String("operator-image", "", "operator image, e.g. mxnet-operator")
	ns := flag.String("namespace", "", "e2e test namespace, use generated one if absent")
	flag.Parse()

	code := 0
	defer func() {
		os.Exit(code)
	}()

	fw = framework.New(k8sclient.GetKubeClient(), *ns, *opImage)
	err := fw.Setup()
	defer fw.Teardown()
	if err != nil {
		logrus.Fatal("failed to setup framework: ", err)
	}

	code = m.Run()
}
