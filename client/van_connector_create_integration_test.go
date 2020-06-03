// +build integration

// this ^ comment here is what makes test in this file only discovered when go
// test executed with the -tags=integration flag, i.e. "$ go test -count=1 -v -tags=integration ./client"

package client_test

import (
	"context"
	"testing"

	"github.com/skupperproject/skupper/test/cluster"
)

func run(r *cluster.SimpleClusterTestRunner, ctx context.Context) {
	testConnectorCreateError(r.T, r.Pub1Cluster.VanClient, ctx)
	testConnectorCreateInterior(r.T, r.Pub1Cluster.VanClient, ctx)
}

func TestIntegrationVanConnectorCreate(t *testing.T) {
	//all this goes inside the Build!
	testRunner := cluster.NewSimpleClusterTestRunner(t)
	testRunner.Run(context.Background(), run)
}
