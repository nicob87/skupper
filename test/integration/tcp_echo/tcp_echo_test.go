package tcp_echo

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/skupperproject/skupper/test"
	"github.com/skupperproject/skupper/test/cluster"

	"k8s.io/client-go/util/homedir"
)

func TestTcpEcho(t *testing.T) {
	if os.Getenv(test.INTEGRATION) == "" {
		t.Skipf("skipping test; %s not set", test.INTEGRATION)
	}
	testRunners := []cluster.ClusterTestRunnerInterface{&TcpEchoClusterTestRunner{}}

	//defaultKubeConfig := filepath.Join(homedir.HomeDir(), "kind.config")
	defaultKubeConfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	pub1Kubeconfig := flag.String("pub1kubeconfig", defaultKubeConfig, "(optional) absolute path to the kubeconfig file")
	pub2Kubeconfig := flag.String("pub2kubeconfig", defaultKubeConfig, "(optional) absolute path to the kubeconfig file")
	priv1Kubeconfig := flag.String("priv1kubeconfig", defaultKubeConfig, "(optional) absolute path to the kubeconfig file")
	priv2Kubeconfig := flag.String("priv2kubeconfig", defaultKubeConfig, "(optional) absolute path to the kubeconfig file")

	flag.Parse()

	for _, testRunner := range testRunners {
		testRunner.Build(*pub1Kubeconfig, *pub2Kubeconfig, *priv1Kubeconfig, *priv2Kubeconfig)
		testRunner.Run()
	}
}
