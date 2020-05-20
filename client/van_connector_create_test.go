package client_test

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
	. "github.com/skupperproject/skupper/client"
	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func testConnectorCreateError(t *testing.T, cli *VanClient, ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := cli.VanConnectorCreate(ctx, "./somefile.yaml", types.VanConnectorCreateOptions{
		Name: "",
		Cost: 1,
	})
	assert.Error(t, err, "open ./somefile.yaml: no such file or directory", "Expect error when file not found")
}

func TestConnectorCreateError(t *testing.T) {
	cli, err := NewMockClient("skupper", "", "")
	assert.Check(t, err)
	testConnectorCreateError(t, cli, context.Background())
}

func TestConnectorCreateInterior(t *testing.T) {
	cli, err := NewMockClient("skupper", "", "")
	assert.Check(t, err)
	testConnectorCreateInterior(t, cli, context.Background())
}

func testConnectorCreateInterior(t *testing.T, cli *VanClient, ctx context.Context) {
	//main difference here is the fact that you are using always the same client,
	//you have to consider, when iterating over the table that you are using
	//the same client on the same persisten namespace. i.e. you can not
	//create the same resource every time.
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
			connName:        "conn22", //just named different than conn1 + 1
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	//cli, err := NewMockClient("skupper", "", "")
	secrets := make(chan *corev1.Secret, 10) //TODO why 10?
	informers := informers.NewSharedInformerFactory(cli.KubeClient, 0)
	secretsInformer := informers.Core().V1().Secrets().Informer()
	secretsInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*corev1.Secret)
			t.Logf("secret! added: %s/%s", secret.Namespace, secret.Name)
			//if !strings.HasPrefix(secret.Name, "skupper") {
			if strings.HasPrefix(secret.Name, "conn") { //just changed to make the example work
				t.Logf("meet condition! \n")
				secrets <- secret
			}
		},
	})

	informers.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), secretsInformer.HasSynced)

	err := cli.VanRouterCreate(ctx, types.VanRouterCreateOptions{
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

		secretsFound := []string{}
		//is it connecting to itself?
		err = cli.VanConnectorTokenCreate(ctx, c.connName, testPath+c.connName+".yaml")
		assert.Check(t, err, "Unable to create token")

		err = cli.VanConnectorCreate(ctx, testPath+c.connName+".yaml", types.VanConnectorCreateOptions{
			Name: c.connName,
			Cost: 1,
		})
		assert.Check(t, err, "Unable to create connector")

		select {
		case secret := <-secrets:
			t.Logf("Got secret from channel: %s/%s", secret.Namespace, secret.Name)
			secretsFound = append(secretsFound, secret.Name)
		case <-time.After(time.Second * 10): //TODO why 10?
			t.Error("Informer did not get the added secret")
		}

		fmt.Printf("=====SecretsFound = %#q \n", secretsFound)
		assert.Assert(t, cmp.Equal(c.secretsExpected, secretsFound, trans), c.doc)

		secretsFound = []string{} //simple "Tear Down"
	}

	// clean up
	//defer this?
	os.RemoveAll(testPath)
}
