// +build integration

package http

import (
	"context"
	"testing"
)

func TestHttp(t *testing.T) {
	testRunner := &HttpClusterTestRunner{}

	testRunner.Build(t, "tcp-echo")
	ctx := context.Background()
	testRunner.Run(ctx)
}
