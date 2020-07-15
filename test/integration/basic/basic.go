package basic

import (
	"context"
	"log"
	"time"

	"github.com/skupperproject/skupper/api/types"
	"github.com/skupperproject/skupper/test/cluster"
	"gotest.tools/assert"
)

type BasicTestRunner struct {
	cluster.ClusterTestRunnerBase
}

func (r *BasicTestRunner) RunTests(ctx context.Context) {

	tick := time.Tick(5 * time.Second)
	wait_for_conn := func(cc *cluster.ClusterContext, timeout_S time.Duration) {
		timeout := time.After(timeout_S * time.Second)
		for {
			select {
			case <-timeout:
				log.Panicln("Timed Out Waiting for service.")
				assert.Assert(r.T, false, "Timeout waiting for connection")
			case <-tick:
				vir, err := cc.VanClient.VanRouterInspect(ctx)
				if err == nil && vir.Status.ConnectedSites.Total == 1 {
					log.Println("Connection found!!!")
					return
				} else {
					log.Println("Connection not ready yet, current pods state: ")
					r.Pub1Cluster.KubectlExec("get pods -o wide")
				}
			}
		}
	}
	wait_for_conn(r.Pub1Cluster, 120)
	wait_for_conn(r.Priv1Cluster, 30)
}

func (r *BasicTestRunner) Setup(ctx context.Context, vanRouterCreateOpts types.VanSiteConfig) {
	var err error
	err = r.Pub1Cluster.CreateNamespace()
	assert.Assert(r.T, err)

	err = r.Priv1Cluster.CreateNamespace()
	assert.Assert(r.T, err)

	vanRouterCreateOpts.Spec.SkupperNamespace = r.Pub1Cluster.CurrentNamespace
	r.Pub1Cluster.VanClient.VanRouterCreate(ctx, vanRouterCreateOpts)

	err = r.Pub1Cluster.VanClient.VanConnectorTokenCreateFile(ctx, types.DefaultVanName, "/tmp/public_secret.yaml")
	assert.Assert(r.T, err)

	vanRouterCreateOpts.Spec.SkupperNamespace = r.Priv1Cluster.CurrentNamespace
	err = r.Priv1Cluster.VanClient.VanRouterCreate(ctx, vanRouterCreateOpts)

	var vanConnectorCreateOpts types.VanConnectorCreateOptions = types.VanConnectorCreateOptions{
		SkupperNamespace: r.Priv1Cluster.CurrentNamespace,
		Name:             "",
		Cost:             0,
	}
	r.Priv1Cluster.VanClient.VanConnectorCreateFromFile(ctx, "/tmp/public_secret.yaml", vanConnectorCreateOpts)
}

func (r *BasicTestRunner) TearDown(ctx context.Context) {
	r.Pub1Cluster.DeleteNamespaces()
	r.Priv1Cluster.DeleteNamespaces()
}

//TODO test isEdge condition also (true and false)
func (r *BasicTestRunner) Run(ctx context.Context) {
	testcases := []struct {
		createOpts types.VanSiteConfig
	}{
		{
			createOpts: types.VanSiteConfig{
				Spec: types.VanSiteConfigSpec{
					SkupperName:       "",
					IsEdge:            false,
					EnableController:  true,
					EnableServiceSync: true,
					EnableConsole:     false,
					AuthMode:          types.ConsoleAuthModeUnsecured,
					User:              "nicob?",
					Password:          "nopasswordd",
					ClusterLocal:      true,
					Replicas:          1,
				},
			},
		},
		{
			createOpts: types.VanSiteConfig{
				Spec: types.VanSiteConfigSpec{
					SkupperName:       "",
					IsEdge:            false,
					EnableController:  true,
					EnableServiceSync: true,
					EnableConsole:     false,
					AuthMode:          types.ConsoleAuthModeUnsecured,
					User:              "nicob?",
					Password:          "nopasswordd",
					ClusterLocal:      false,
					Replicas:          1,
				},
			},
		},
	}

	defer r.TearDown(ctx)

	for _, c := range testcases {
		r.Setup(ctx, c.createOpts)
		r.RunTests(ctx)
	}
}
