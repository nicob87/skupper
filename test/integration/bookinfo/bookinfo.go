package tcp_echo

import (
	"context"
	"fmt"
	"testing"

	"github.com/skupperproject/skupper/api/types"
	"github.com/skupperproject/skupper/test/utils/base"

	"gotest.tools/assert"
)

//func int32Ptr(i int32) *int32 { return &i }

func RunTests(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	fmt.Printf("Running tests!!!\n")
}

func Setup(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	pub1Cluster, err := r.GetPublicContext(1)
	assert.Assert(t, err)

	prv1Cluster, err := r.GetPrivateContext(1)
	assert.Assert(t, err)

	err = base.SetupSimplePublicPrivateAndConnect(ctx, r, "tcp_echo")
	assert.Assert(t, err)

	_, err = pub1Cluster.KubectlExec("apply -f https://raw.githubusercontent.com/skupperproject/skupper-example-bookinfo/master/public-cloud.yaml")
	assert.Assert(t, err)

	_, err = prv1Cluster.KubectlExec("apply -f https://raw.githubusercontent.com/skupperproject/skupper-example-bookinfo/master/private-cloud.yaml")
	assert.Assert(t, err)

	detailsService := types.ServiceInterface{
		Address:  "details",
		Protocol: "http",
		//Port:     80,
	}

	reviewsService := types.ServiceInterface{
		Address:  "reviews",
		Protocol: "http",
		//Port:     80,
	}

	ratingsService := types.ServiceInterface{
		Address:  "ratings",
		Protocol: "http",
		//Port:     80,
	}

	err = prv1Cluster.VanClient.ServiceInterfaceCreate(ctx, &detailsService)
	assert.Assert(t, err)

	err = prv1Cluster.VanClient.ServiceInterfaceCreate(ctx, &reviewsService)
	assert.Assert(t, err)

	err = pub1Cluster.VanClient.ServiceInterfaceCreate(ctx, &ratingsService)
	assert.Assert(t, err)

	err = prv1Cluster.VanClient.ServiceInterfaceBind(ctx, &detailsService, "service", "details", "http", 0)
	assert.Assert(t, err)

	err = prv1Cluster.VanClient.ServiceInterfaceBind(ctx, &detailsService, "service", "reviews", "http", 0)
	assert.Assert(t, err)

	err = pub1Cluster.VanClient.ServiceInterfaceBind(ctx, &detailsService, "service", "ratings", "http", 0)
	assert.Assert(t, err)

}

func Run(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	defer base.TearDownSimplePublicAndPrivate(r)
	Setup(ctx, t, r)
	RunTests(ctx, t, r)
}
