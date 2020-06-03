package cluster

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"gotest.tools/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/skupperproject/skupper/client"
	vanClient "github.com/skupperproject/skupper/client"
)

type ClusterTestRunnerInterface interface {
	Build(t *testing.T, public1ConficFile, public2ConficFile, private1ConfigFile, private2ConfigFile string)
	Run()
}

type ClusterTestRunnerBase struct {
	Pub1Cluster  *ClusterContext
	Pub2Cluster  *ClusterContext
	Priv1Cluster *ClusterContext
	Priv2Cluster *ClusterContext
	T            *testing.T
}

func (r *ClusterTestRunnerBase) Build(t *testing.T) {

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homedir, err := os.UserHomeDir()
		assert.Check(t, err)
		kubeconfig = path.Join(homedir, ".kube/config")
	}

	//TODO assign here uniq, publicX and privateX namespaces instead of
	//generic ones
	r.Pub1Cluster = BuildClusterContext(t, "public1", kubeconfig)
	r.Pub2Cluster = BuildClusterContext(t, "public2", kubeconfig)
	r.Priv1Cluster = BuildClusterContext(t, "private1", kubeconfig)
	r.Priv2Cluster = BuildClusterContext(t, "private2", kubeconfig)
	r.T = t
}

type ClusterContext struct {
	Namespace         string
	ClusterConfigFile string
	VanClient         *vanClient.VanClient
	t                 *testing.T
}

func BuildClusterContext(t *testing.T, namespace string, configFile string) *ClusterContext {
	var err error
	cc := &ClusterContext{}
	cc.t = t
	cc.Namespace = namespace
	cc.ClusterConfigFile = configFile
	cc.VanClient, err = client.NewClient(cc.Namespace, "", cc.ClusterConfigFile)
	assert.Check(cc.t, err)
	return cc
}

func _exec(command string, wait bool) {
	var output []byte
	var err error
	fmt.Println(command)
	cmd := exec.Command("sh", "-c", command)
	if wait {
		output, err = cmd.CombinedOutput()
	} else {
		cmd.Start()
		return
	}
	if err != nil {
		panic(err)
	}
	fmt.Println(string(output))
}

func (cc *ClusterContext) exec(main_command string, sub_command string, wait bool) {
	_exec("KUBECONFIG="+cc.ClusterConfigFile+" "+main_command+" "+cc.Namespace+" "+sub_command, wait)
}

func (cc *ClusterContext) SkupperExec(command string) {
	cc.exec("./skupper -n ", command, true)
}

func (cc *ClusterContext) _kubectl_exec(command string, wait bool) {
	cc.exec("kubectl -n ", command, wait)
}

func (cc *ClusterContext) KubectlExec(command string) {
	cc._kubectl_exec(command, true)
}

func (cc *ClusterContext) KubectlExecAsync(command string) {
	cc._kubectl_exec(command, false)
}

func (cc *ClusterContext) CreateNamespace() {
	NsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: cc.Namespace}}
	_, err := cc.VanClient.KubeClient.CoreV1().Namespaces().Create(NsSpec)
	assert.Check(cc.t, err)
}

func (cc *ClusterContext) DeleteNamespace() {
	err := cc.VanClient.KubeClient.CoreV1().Namespaces().Delete(cc.Namespace, &metav1.DeleteOptions{})
	if err != nil {
		log.Panic(err.Error())
	}
}

func (cc *ClusterContext) GetService(name string, timeout_S time.Duration) *apiv1.Service {
	timeout := time.After(timeout_S * time.Second)
	tick := time.Tick(3 * time.Second)
	for {
		select {
		case <-timeout:
			log.Panicln("Timed Out Waiting for service.")
		case <-tick:
			service, err := cc.VanClient.KubeClient.CoreV1().Services(cc.Namespace).Get(name, metav1.GetOptions{})
			if err == nil {
				return service
			} else {
				log.Println("Service not ready yet, current pods state: ")
				cc.KubectlExec("get pods -o wide") //TODO use clientset
			}

		}
	}
}
