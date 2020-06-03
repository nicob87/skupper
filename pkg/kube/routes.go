package kube

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/skupperproject/skupper/api/types"
)

func GetRoute(name string, namespace string, rc *routev1client.RouteV1Client) (*routev1.Route, error) {
	current, err := rc.Routes(namespace).Get(name, metav1.GetOptions{})
	return current, err
}

func NewRoute(rte types.Route, owner *metav1.OwnerReference, namespace string, rc *routev1client.RouteV1Client) (*routev1.Route, error) {
	insecurePolicy := routev1.InsecureEdgeTerminationPolicyNone
	if rte.Termination != routev1.TLSTerminationPassthrough {
		insecurePolicy = routev1.InsecureEdgeTerminationPolicyRedirect
	}
	current, err := rc.Routes(namespace).Get(rte.Name, metav1.GetOptions{})
	if err == nil {
		return current, fmt.Errorf("Route %s already exists", rte.Name)
	} else if errors.IsNotFound(err) {
		route := &routev1.Route{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Route",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: rte.Name,
			},
			Spec: routev1.RouteSpec{
				Path: "",
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString(rte.TargetPort),
				},
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: rte.TargetService,
				},
				TLS: &routev1.TLSConfig{
					Termination:                   rte.Termination,
					InsecureEdgeTerminationPolicy: insecurePolicy,
				},
			},
		}
		if owner != nil {
			route.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
				*owner,
			}
		}
		created, err := rc.Routes(namespace).Create(route)
		if err != nil {
			return nil, fmt.Errorf("Failed to create route : %w", err)
		} else {
			return created, nil
		}
	} else {
		return nil, fmt.Errorf("Failed while checking route: %w", err)
	}
}
