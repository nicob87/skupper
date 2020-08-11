package cluster

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"
	"time"

	"gotest.tools/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batchv1 "k8s.io/api/batch/v1"

	vanClient "github.com/skupperproject/skupper/client"
)

const (
	//until this issue: https://github.com/skupperproject/skupper/issues/163
	//is fixed, this is the best we can do
	SkupperServiceReadyPeriod              time.Duration = time.Minute
	DefaultTick                                          = time.Second * 5
	TestJobBackOffLimit                                  = 3
	ImagePullingAndResourceCreationTimeout               = 10 * time.Minute
)

type ClusterTestRunnerInterface interface {
	Build(t *testing.T, ns_suffix string) //is this interface used?
	Run()
}

type ClusterTestRunnerBase struct {
	Pub1Cluster  *ClusterContext
	Pub2Cluster  *ClusterContext
	Priv1Cluster *ClusterContext
	Priv2Cluster *ClusterContext
	T            *testing.T
}

func (r *ClusterTestRunnerBase) Build(t *testing.T, ns_suffix string) {

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homedir, err := os.UserHomeDir()
		assert.Assert(t, err)
		kubeconfig = path.Join(homedir, ".kube/config")
	}

	//TODO assign here uniq, publicX and privateX namespaces instead of
	//generic ones
	r.Pub1Cluster = BuildClusterContext(t, "public1-"+ns_suffix, kubeconfig, vanClient.NewClient)
	r.Pub2Cluster = BuildClusterContext(t, "public2-"+ns_suffix, kubeconfig, vanClient.NewClient)
	r.Priv1Cluster = BuildClusterContext(t, "private1-"+ns_suffix, kubeconfig, vanClient.NewClient)
	r.Priv2Cluster = BuildClusterContext(t, "private2-"+ns_suffix, kubeconfig, vanClient.NewClient)
	r.T = t
}

type ClusterContext struct {
	NamespacePrefix   string
	CurrentNamespace  string
	Namespaces        []string
	ClusterConfigFile string
	VanClient         *vanClient.VanClient
	t                 *testing.T
}

func BuildClusterContext(t *testing.T, namespacePrefix string, configFile string, newVanClient func(namespace, context, kubeConfigPath string) (*vanClient.VanClient, error)) *ClusterContext {
	var err error
	cc := &ClusterContext{}
	cc.t = t
	cc.NamespacePrefix = namespacePrefix
	cc.ClusterConfigFile = configFile
	cc.VanClient, err = newVanClient("", "", cc.ClusterConfigFile)
	assert.Check(cc.t, err)
	return cc
}

func _exec(command string) ([]byte, error) {
	var output []byte
	var err error
	fmt.Println(command)
	cmd := exec.Command("sh", "-c", command)
	output, err = cmd.CombinedOutput()
	fmt.Println(string(output))
	return output, err
}

func (cc *ClusterContext) exec(main_command string, sub_command string) ([]byte, error) {
	return _exec("KUBECONFIG=" + cc.ClusterConfigFile + " " + main_command + " " + cc.CurrentNamespace + " " + sub_command)
}

//do a simple test of this
func (cc *ClusterContext) KubectlExec(command string) ([]byte, error) {
	return cc.exec("kubectl -n ", command)
}

func (cc *ClusterContext) getNextNamespace() string {
	return cc.NamespacePrefix + "-" + strconv.Itoa((len(cc.Namespaces) + 1))
}

func (cc *ClusterContext) moveToNextNamespace() {
	next := cc.getNextNamespace()
	cc.Namespaces = append(cc.Namespaces, next)
	cc.CurrentNamespace = next
	cc.VanClient.Namespace = cc.CurrentNamespace
}

func (cc *ClusterContext) CreateNamespace() error {
	ns := cc.getNextNamespace()
	NsSpec := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	_, err := cc.VanClient.KubeClient.CoreV1().Namespaces().Create(NsSpec)
	if err != nil {
		return err
	}
	cc.moveToNextNamespace()
	return nil
}

func (cc *ClusterContext) deleteNamespace(ns string) {
	//remove from the list
	err := cc.VanClient.KubeClient.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
	assert.Check(cc.t, err)
}

func (cc *ClusterContext) DeleteNamespaces() {
	for _, ns := range cc.Namespaces {
		cc.deleteNamespace(ns)
	}
	cc.Namespaces = cc.Namespaces[:0]
	cc.CurrentNamespace = ""
}

func (cc *ClusterContext) DeleteNamespace() {
	assert.Equal(cc.t, 1, len(cc.Namespaces), "Use DeleteNamespaces")
	cc.DeleteNamespaces()
}

func (cc *ClusterContext) GetService(name string, timeout time.Duration) (*apiv1.Service, error) {
	guardTime := 500 * time.Millisecond

	getService := func() (*apiv1.Service, error) {
		return cc.VanClient.KubeClient.CoreV1().Services(cc.CurrentNamespace).Get(name, metav1.GetOptions{})

	}

	timedOut := func() error {
		return fmt.Errorf("Timeout waiting for service: %s\n", name)
	}

	service, err := getService()
	if err == nil {
		return service, nil
	}

	if timeout < DefaultTick+guardTime {
		return nil, timedOut()
	}

	timeout_elapsed := time.After(timeout)
	tick := time.Tick(DefaultTick)
	for {
		select {
		case <-timeout_elapsed:
			return nil, timedOut()
		case <-tick:
			service, err := getService()
			if err == nil {
				return service, nil
			} else {
				log.Println("Service not ready yet, current pods state: ")
				cc.KubectlExec("get pods -o wide") //TODO use clientset
			}
		}
	}
}

func getTestImage() string {
	testImage := os.Getenv("TEST_IMAGE")
	if testImage == "" {
		testImage = "quay.io/skupper/skupper-tests"
	}
	return testImage
}

func int32Ptr(i int32) *int32 { return &i }

func (cc *ClusterContext) CreateTestJob(name string, command []string) (*batchv1.Job, error) {

	namespace := cc.CurrentNamespace
	testImage := getTestImage()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(3),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:            name,
							Image:           testImage,
							ImagePullPolicy: apiv1.PullAlways,
							Command:         command,
							Env: []apiv1.EnvVar{
								{Name: "JOB", Value: name},
							},
						},
					},
					RestartPolicy: apiv1.RestartPolicyNever,
				},
			},
		},
	}

	jobsClient := cc.VanClient.KubeClient.BatchV1().Jobs(namespace)

	job, err := jobsClient.Create(job)

	if err != nil {
		return nil, err
	}
	return job, nil
}

func AssertJob(t *testing.T, job *batchv1.Job) {
	t.Helper()
	assert.Equal(t, int(job.Status.Succeeded), 1)
	assert.Equal(t, int(job.Status.Active), 0)

	//Now that we are using a
	//backoff limit grater than 1, evaluate what to assert here
	//assert.Equal(r.T, int(job.Status.Failed), 0)
}

func SkipTestJobIfMustBeSkipped(t *testing.T) {
	if os.Getenv("JOB") == "" {
		t.Skip("JOB environment variable not defined")
	}
}

//TODO evaluate modifying this implementation to use informers instead of
//pooling.
func (cc *ClusterContext) WaitForJob(jobName string, timeout time.Duration) (*batchv1.Job, error) {

	if timeout < DefaultTick {
		return nil, fmt.Errorf("timeout too small: %v", timeout)
	}

	jobsClient := cc.VanClient.KubeClient.BatchV1().Jobs(cc.CurrentNamespace)

	//TODO: in case of multiple retries, is it possible to print last, and
	//previous logs?
	defer cc.KubectlExec("logs job/" + jobName)

	timeoutCh := time.After(timeout)
	tick := time.Tick(DefaultTick)
	for {
		select {
		case <-timeoutCh:
			return nil, fmt.Errorf("Timeout: Job is still active: %s", jobName)
		case <-tick:
			job, _ := jobsClient.Get(jobName, metav1.GetOptions{})

			cc.KubectlExec(fmt.Sprintf("get job/%s -o wide", jobName))
			cc.KubectlExec("get pods -o wide")

			if job.Status.Active > 0 {
				fmt.Println("Job is still active")
			} else {
				if job.Status.Succeeded > 0 {
					fmt.Println("Job Successful!")
					return job, nil
				}
				fmt.Printf("Job failed?, status = %v\n", job.Status)
				return job, nil
			}
		}
	}

}

func (cc *ClusterContext) WaitForSkupperServiceToBeCreatedAndReadyToUse(service string, timeout time.Duration) (*apiv1.Service, error) {

	svc, err := cc.GetService(service, timeout)
	if err != nil {
		return nil, err
	}

	//Provide mechanism to wait until a newly defined service is 'ready'
	//https://github.com/skupperproject/skupper/issues/163
	time.Sleep(SkupperServiceReadyPeriod)
	return svc, nil
}
