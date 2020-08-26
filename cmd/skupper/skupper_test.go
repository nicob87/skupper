package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/assert"
)

func TestMain(m *testing.M) {
	silenceCobra()
	os.Exit(m.Run())
}

func Test_requiredArg(t *testing.T) {
	r := func(args []string) error {
		return requiredArg("testArg")(nil, args)
	}

	assert.Error(t, r([]string{}), "testArg must be specified")
	assert.Error(t, r([]string{"too", "many"}), "illegal argument: many")
	assert.Error(t, r([]string{"too", "many", "more"}), "illegal argument: many")

	assert.Assert(t, r([]string{"OneArgument"}))
}

func Test_bindArgs(t *testing.T) {
	genericError := "Service name, target type and target name must all be specified (e.g. 'skupper bind <service-name> <target-type> <target-name>')"
	b := func(args []string) error {
		return bindArgs(nil, args)
	}

	assert.Error(t, b([]string{}), genericError)
	assert.Error(t, b([]string{"oneArg"}), genericError)
	assert.Error(t, b([]string{"one/Arg"}), genericError)
	assert.Error(t, b([]string{"one", "resource"}), genericError)

	//must this fail?
	//assert.Error(t, b([]string{"one/two", "resource/name"}), genericError)

	assert.Assert(t, b([]string{"one", "resource/name"}))
	//note  illegal vs extra
	assert.Error(t, b([]string{"one", "resource/name", "three"}), "extra argument: three")
	assert.Error(t, b([]string{"one", "resource/name", "three", "four"}), "illegal argument: four")
	assert.Error(t, b([]string{"one", "resource/name", "three", "four", "five"}), "illegal argument: four")

	assert.Assert(t, b([]string{"one", "resource", "name"}))
	assert.Error(t, b([]string{"one", "resource", "name", "four"}), "illegal argument: four")
	assert.Error(t, b([]string{"one", "resource", "name", "four", "five"}), "illegal argument: four")
}

func Test_createServiceArgs(t *testing.T) {
	c := func(args []string) error {
		return createServiceArgs(nil, args)
	}

	assert.Error(t, c([]string{}), "Name and port must be specified")
	assert.Error(t, c([]string{"noport"}), "Name and port must be specified")

	assert.Assert(t, c([]string{"service:port"}))

	assert.Error(t, c([]string{"service:port", "other"}), "extra argument: other")
	assert.Error(t, c([]string{"service:port", "other", "arg"}), "illegal argument: arg")

	assert.Assert(t, c([]string{"service", "port"}))
	assert.Error(t, c([]string{"service", "port", "other"}), "illegal argument: other")
	assert.Error(t, c([]string{"service", "port", "other", "arg"}), "illegal argument: other")
}

func Test_exposeTargetArgs(t *testing.T) {
	genericError := "expose target and name must be specified (e.g. 'skupper expose deployment <name>'"
	targetError := "expose target type must be one of: [deployment, statefulset, pods, service]"

	e := func(args []string) error {
		return exposeTargetArgs(nil, args)
	}

	assert.Error(t, e([]string{}), genericError)
	assert.Error(t, e([]string{"depl/name"}), targetError)

	assert.Error(t, e([]string{"depl/name", "two"}), "extra argument: two")
	assert.Error(t, e([]string{"depl/name", "two", "three"}), "illegal argument: three")
	assert.Error(t, e([]string{"depl/name", "two", "three", "four"}), "illegal argument: three")

	assert.Error(t, e([]string{"depl/name"}), targetError)
	assert.Error(t, e([]string{"anything", "name"}), targetError)

	assert.Error(t, e([]string{"deployment"}), genericError)

	assert.Assert(t, e([]string{"deployment", "name"}))

	assert.Error(t, e([]string{"deployment", "name", "three"}), "illegal argument: three")
	assert.Error(t, e([]string{"deployment", "name", "three", "four"}), "illegal argument: three")

	for _, target := range validExposeTargets {
		assert.Assert(t, e([]string{target, "name"}))
	}
}

type serviceInterfaceUnbindCallArgs struct {
	targetType, targetName, address string
	deleteIfNoTargets               bool
}

type vanClientMock struct {
	serviceInterfaceUnbindCalledWith   []serviceInterfaceUnbindCallArgs
	serviceInterfaceUnbindReturnsError string
}

func (v *vanClientMock) ServiceInterfaceUnbind(ctx context.Context, targetType string, targetName string, address string, deleteIfNoTargets bool) error {
	var calledWith = serviceInterfaceUnbindCallArgs{
		targetType:        targetType,
		targetName:        targetName,
		address:           address,
		deleteIfNoTargets: deleteIfNoTargets,
	}
	v.serviceInterfaceUnbindCalledWith = append(v.serviceInterfaceUnbindCalledWith, calledWith)

	if v.serviceInterfaceUnbindReturnsError != "" {
		return fmt.Errorf("%s", v.serviceInterfaceUnbindReturnsError)
	}

	return nil
}

func Test_cmdUnexpose(t *testing.T) {
	test := func(targetType, targetName, address string, injectedError string) {
		options := Options{
			unexposeAddress: address,
		}
		cli := vanClientMock{}
		cli.serviceInterfaceUnbindReturnsError = injectedError

		args := []string{targetType}

		//supporting "targetType TargetName" and "targetType/targetName" notations
		if targetName != "" {
			args = append(args, targetName)
		} else {
			parts := strings.Split(targetType, "/")
			targetType = parts[0]
			targetName = parts[1]
		}

		err := unexposeRun(nil, args, options, &cli)

		if injectedError != "" {
			assert.Error(t, err, "Error, unable to skupper service: "+injectedError)
		} else {
			assert.Assert(t, err)
		}

		assert.Equal(t, len(cli.serviceInterfaceUnbindCalledWith), 1)

		expected := serviceInterfaceUnbindCallArgs{
			targetType:        targetType,
			targetName:        targetName,
			address:           address,
			deleteIfNoTargets: true}

		assert.Assert(t, cmp.Equal(cli.serviceInterfaceUnbindCalledWith[0], expected, cmp.AllowUnexported(serviceInterfaceUnbindCallArgs{})))
	}

	testSuccess := func(targetType, targetName, address string) {
		test(targetType, targetName, address, "")
	}

	testError := func(targetType, targetName, address string, errorString string) {
		test(targetType, targetName, address, errorString)
	}

	testSuccess("depl", "Name", "theService:8080")
	testSuccess("depl/Name", "", "theService:8080")

	testError("depl", "Name", "theService:8080", "some error")
	testError("depl/Name", "", "theService:8080", "other error")
}
