package cluster

import (
	"context"
	"testing"
)

type SimpleClusterTestRunner struct {
	ClusterTestRunnerBase
}

func (r *SimpleClusterTestRunner) Setup(ctx context.Context) {
	r.Pub1Cluster.CreateNamespace()
}

func (r *SimpleClusterTestRunner) TearDown(ctx context.Context) {
	r.Pub1Cluster.DeleteNamespace()
}

func (r *SimpleClusterTestRunner) Run(ctx context.Context, runTests func(r *SimpleClusterTestRunner, ctx context.Context)) {
	defer r.TearDown(ctx)
	r.Setup(ctx)
	runTests(r, ctx)
}

func NewSimpleClusterTestRunner(t *testing.T) *SimpleClusterTestRunner {
	testRunner := &SimpleClusterTestRunner{}
	testRunner.Build(t)
	return testRunner
}
