package client_test

import (
	"context"
	"os"
	"testing"

	"github.com/skupperproject/skupper/api/types"
	. "github.com/skupperproject/skupper/client"
	"gotest.tools/assert"
)

func TestConnectorInspectError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the client
	cli, err := NewMockClient("skupper", "", "")

	_, err = cli.VanConnectorInspect(ctx, "conn1")
	assert.Error(t, err, `deployments.apps "skupper-router" not found`, "Expect error when VAN is not deployed")
}

func TestConnectorInspectNotFound(t *testing.T) {
	testcases := []struct {
		doc           string
		expectedError string
		connName      string
	}{
		{
			expectedError: `secrets "conn1" not found`,
			doc:           "test one",
			connName:      "conn1",
		},
		{
			expectedError: `secrets "all" not found`,
			doc:           "test two",
			connName:      "all",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewMockClient("skupper", "", "")

	err = cli.VanRouterCreate(ctx, types.VanRouterCreateOptions{
		SkupperName:       "skupper",
		IsEdge:            false,
		EnableController:  true,
		EnableServiceSync: true,
		EnableConsole:     false,
		AuthMode:          "",
		User:              "",
		Password:          "",
		ClusterLocal:      true,
	})
	assert.Check(t, err, "Unable to create VAN router")

	for _, c := range testcases {
		_, err := cli.VanConnectorInspect(ctx, c.connName)
		assert.Error(t, err, c.expectedError, c.doc)
	}
}

func TestConnectorInspectDefaults(t *testing.T) {
	testcases := []struct {
		doc           string
		expectedError string
		connName      string
	}{
		{
			expectedError: "",
			doc:           "test one",
			connName:      "conn1",
		},
		{
			expectedError: "",
			doc:           "test one",
			connName:      "conn2",
		},
		{
			expectedError: "",
			doc:           "test two",
			connName:      "conn3",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := NewMockClient("skupper", "", "")

	testPath := "./tmp/"
	os.Mkdir(testPath, 0755)

	err = cli.VanRouterCreate(ctx, types.VanRouterCreateOptions{
		SkupperName:       "skupper",
		IsEdge:            false,
		EnableController:  true,
		EnableServiceSync: true,
		EnableConsole:     false,
		AuthMode:          "",
		User:              "",
		Password:          "",
		ClusterLocal:      true,
	})
	assert.Check(t, err, "Unable to create VAN router")

	for _, c := range testcases {
		err = cli.VanConnectorTokenCreate(ctx, c.connName, testPath+c.connName+".yaml")
		assert.Check(t, err, "Unable to create connector token "+c.connName)
	}
	for _, c := range testcases {
		err = cli.VanConnectorCreate(ctx, testPath+c.connName+".yaml", types.VanConnectorCreateOptions{
			Name: c.connName,
			Cost: 1,
		})
		assert.Check(t, err, "Unable to create connector for "+c.connName)
	}
	for _, c := range testcases {
		_, err := cli.VanConnectorInspect(ctx, c.connName)
		assert.Check(t, err, "Unabled to inspect connector for "+c.connName)
	}

	os.RemoveAll(testPath)

}
