// +build integration

// this ^ comment here is what makes test in this file only discovered when go
// test executed with the -tags=integration flag, i.e. "$ go test -count=1 -v -tags=integration ./client"

package client_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/skupperproject/skupper/test/cluster"
	"gotest.tools/assert"
)

type VanConnectorCreateInteriorTestRunner struct {
	cluster.ClusterTestRunnerBase
}

func (r *VanConnectorCreateInteriorTestRunner) Setup(ctx context.Context) {
	r.Pub1Cluster.CreateNamespace()
}

func (r *VanConnectorCreateInteriorTestRunner) TearDown(ctx context.Context) {
	r.Pub1Cluster.DeleteNamespace()
}

//these three are the only "NON" boiler plate or duplicated lines in this file
func (r *VanConnectorCreateInteriorTestRunner) RunTests(ctx context.Context) {
	fmt.Println("Executing Create Interior integration test!")
	testConnectorCreateInterior(r.T, r.Pub1Cluster.VanClient, ctx)
}

/////////////////////////////////////////////////////////////////////////////

func (r *VanConnectorCreateInteriorTestRunner) Run(ctx context.Context) { //this tear down has to be optional
	defer r.TearDown(ctx)
	r.Setup(ctx)
	r.RunTests(ctx)
}

func TestIntegrationVanConnectorCreateInterior(t *testing.T) {
	//all this goes inside the Build!
	testRunner := &VanConnectorCreateInteriorTestRunner{}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homedir, err := os.UserHomeDir()
		assert.Check(t, err)
		kubeconfig = path.Join(homedir, ".kube/config")
	}

	testRunner.Build(t, kubeconfig, kubeconfig, kubeconfig, kubeconfig)
	ctx := context.Background()

	testRunner.Run(ctx)
}
