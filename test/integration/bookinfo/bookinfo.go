package tcp_echo

import (
	"context"
	"fmt"
	"testing"

	"github.com/skupperproject/skupper/api/types"
	"github.com/skupperproject/skupper/test/utils/base"
	"github.com/skupperproject/skupper/test/utils/constants"
	"github.com/skupperproject/skupper/test/utils/k8s"

	"gotest.tools/assert"
)

func RunTests(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	pub1Cluster, err := r.GetPublicContext(1)
	assert.Assert(t, err)

	fmt.Printf("Running tests!!!\n")

	jobName := "bookinfo"
	jobCmd := []string{"/app/bookinfo_test", "-test.run", "Job"}

	_, err = k8s.CreateTestJob(pub1Cluster.Namespace, pub1Cluster.VanClient.KubeClient, jobName, jobCmd)
	assert.Assert(t, err)

	job, err := k8s.WaitForJob(pub1Cluster.Namespace, pub1Cluster.VanClient.KubeClient, jobName, constants.ImagePullingAndResourceCreationTimeout)
	assert.Assert(t, err)

	k8s.AssertJob(t, job)
}

func Setup(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	pub1Cluster, err := r.GetPublicContext(1)
	assert.Assert(t, err)

	prv1Cluster, err := r.GetPrivateContext(1)
	assert.Assert(t, err)

	err = base.SetupSimplePublicPrivateAndConnect(ctx, r, "bookinfo")
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

	err = prv1Cluster.VanClient.ServiceInterfaceBind(ctx, &reviewsService, "service", "reviews", "http", 0)
	assert.Assert(t, err)

	err = pub1Cluster.VanClient.ServiceInterfaceBind(ctx, &ratingsService, "service", "ratings", "http", 0)
	assert.Assert(t, err)

	//TODO use here a goroutine group?! or something like that, that counts
	//the number of processes to finish
	_, err = k8s.WaitForSkupperServiceToBeCreatedAndReadyToUse(pub1Cluster.Namespace, pub1Cluster.VanClient.KubeClient, "details")
	assert.Assert(t, err)

	_, err = k8s.WaitForSkupperServiceToBeCreatedAndReadyToUse(pub1Cluster.Namespace, pub1Cluster.VanClient.KubeClient, "reviews")
	assert.Assert(t, err)

	_, err = k8s.WaitForSkupperServiceToBeCreatedAndReadyToUse(prv1Cluster.Namespace, prv1Cluster.VanClient.KubeClient, "ratings")
	assert.Assert(t, err)

}

func Run(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	defer base.TearDownSimplePublicAndPrivate(r)
	Setup(ctx, t, r)
	RunTests(ctx, t, r)
}
