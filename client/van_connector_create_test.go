package client

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/skupperproject/skupper/api/types"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func TestConnectorCreateError(t *testing.T) {
	cli, err := newMockClient("skupper", "", "")
	assert.Check(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = cli.VanConnectorCreate(ctx, "./somefile.yaml", types.VanConnectorCreateOptions{
		Name: "",
		Cost: 1,
	})
	assert.Error(t, err, "open ./somefile.yaml: no such file or directory", "Expect error when file not found")
}

func TestConnectorCreateInterior(t *testing.T) {
	testcases := []struct {
		doc             string
		expectedError   string
		connName        string
		connFile        string
		secretsExpected []string
	}{
		{
			doc:           "Expect generated name to be conn1",
			expectedError: "",
			//connName:        "connNamemustbedifferent",
			connName:        "",
			secretsExpected: []string{"conn1"},
		},
		{
			doc:             "Expect secret name to be as provided: conn22",
			expectedError:   "",
			connName:        "conn22",
			secretsExpected: []string{"conn22"},
		},
	}

	//TODO do a symple loop verifying and asserting no repeated table
	//connection.

	trans := cmp.Transformer("Sort", func(in []string) []string {
		out := append([]string(nil), in...)
		sort.Strings(out)
		return out
	})

	testPath := "./tmp/"
	os.Mkdir(testPath, 0755)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secretsFound := []string{}

	cli, err := newMockClient("skupper", "", "")
	assert.Check(t, err)

	informers := informers.NewSharedInformerFactory(cli.KubeClient, 0)
	secretsInformer := informers.Core().V1().Secrets().Informer()
	secretsInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*corev1.Secret)
			if !strings.HasPrefix(secret.Name, "skupper") {
				secretsFound = append(secretsFound, secret.Name)
			}
		},
	})

	informers.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), secretsInformer.HasSynced)

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

		//is it connecting to itself?
		err = cli.VanConnectorTokenCreate(ctx, c.connName, testPath+c.connName+".yaml")
		assert.Check(t, err, "Unable to create token")

		err = cli.VanConnectorCreate(ctx, testPath+c.connName+".yaml", types.VanConnectorCreateOptions{
			Name: c.connName,
			Cost: 1,
		})
		assert.Check(t, err, "Unable to create connector")

		// TODO: make more deterministic
		time.Sleep(time.Second * 1)
		fmt.Printf("secretsFound= %q", secretsFound)
		assert.Assert(t, cmp.Equal(c.secretsExpected, secretsFound, trans), c.doc)

		secretsFound = []string{} //simple "Tear Down"
	}

	// clean up
	//defer this?
	os.RemoveAll(testPath)
}
