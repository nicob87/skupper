package tcp_echo

import (
	"context"
	"fmt"
	"testing"

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
}

func RemoveNamespacesForContexes(r *base.ClusterTestRunnerBase, public []int, priv []int) error {
	removeNamespaces := func(private bool, ids []int) error {
		for id := range ids {
			cc, err := r.GetContext(private, id)
			if err != nil {
				return err
			}
			cc.DeleteNamespace()
		}
		return nil
	}
	err := removeNamespaces(true, priv)
	if err != nil {
		return err
	}
	return removeNamespaces(false, public)
}

func Run(ctx context.Context, t *testing.T, r *base.ClusterTestRunnerBase) {
	defer base.TearDownSimplePublicAndPrivate(r)
	Setup(ctx, t, r)
	RunTests(ctx, t, r)
}
