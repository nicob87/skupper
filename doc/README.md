# SIMPLE SKUPPER CONFIGURATION EXPLAINED IN DETAIL

## skupper init
### public namespace
```
$ kubectl create namespace public
namespace/public created
$ kubectl config set-context --current --namespace=public
Context "minikube" modified.
$ ./skupper init --cluster-local                             
Skupper is now installed in namespace 'public'.  Use 'skupper status' to get more information.
```
This creates two deployments:
```
$ kubectl get deployments
NAME                         READY   UP-TO-DATE   AVAILABLE   AGE
skupper-router               0/1     1            0           9m24s
skupper-service-controller   0/1     1            0           9m21s
```

skupper-router deployment contains 1 pod composed of two containers:
```
ImageID: quay.io/interconnectedcloud/qdrouterd
imageID: quay.io/gordons/bridge-server
```
The "bridge" is an extension to the qpid-dispatch router, to add support for tcp/http protocols, by default the router supports only amqp. This support will be included "builtin" in the router in the future.

skupper-service-controller deployment:

The service controller is responsible for ensuring that the bridges/proxies are configured correctly to implement the connectivity desired for skupper services. It takes as input the skupper-services config map. It will also populate that map based on annotated deployments or services.

### private namespace
Now we create and do the same in another namespace:

```
$ kubectl create namespace private
namespace/public created
$ kubectl config set-context --current --namespace=private
Context "minikube" modified.
$ ./skupper init --cluster-local                             
Skupper is now installed in namespace 'private'.  Use 'skupper status' to get more information.
$ kubectl get deployments.apps 
NAME                         READY   UP-TO-DATE   AVAILABLE   AGE
skupper-router               1/1     1            1           51s
skupper-service-controller   1/1     1            1           48s
```

So now we have two namespaces (sites?) not connected to each other in any way



## Connecting sites
### crete token
```
$ ./skupper connection-token -n public my-token.yaml
Connection token written to my-token.yaml (Note: token will only be valid for local cluster)
```
Token contains credentials information and also ports and host to be used to connect, using this token 
```
form: van_connector_token_create.go
...
caSecret, err := cli.KubeClient.CoreV1().Secrets(cli.Namespace).Get("skupper-internal-ca", metav1.GetOptions{})
...                                                                                                                                                                                                       
result.InterRouter.Port = "55671"                                                                                                                                                                                                  
result.Edge.Port = "45671" 
```
This namespace is named public, since, all the following steps will only work if incoming connections to ports 55671 and/or 45671 ports is allowed.

### connect usint token
now from "private" namespace we connect to the public namespace using the token.

```
$ kubectl config set-context --current --namespace=private
Context "minikube" modified.
$ ./skupper connect my-token.yaml 
Skupper configured to connect to skupper-internal.public:55671 (name=conn1)
```

```
current, err := kube.GetDeployment(types.TransportDeploymentName, options.SkupperNamespace, cli.KubeClient)
...
 secret, cli.createConnector(ctx, secret, options, current)
```

This adds a connector to the deployment resource:
```
connector := types.Connector{                                                                                                                   
Name: options.Name,                                                                                                                     
Cost: options.Cost,                                                                                                                     
}                                                                                                                                               
if mode == types.TransportModeInterior {                                                                                                        
connector.Host = secret.ObjectMeta.Annotations["inter-router-host"]                                                                     
connector.Port = secret.ObjectMeta.Annotations["inter-router-port"]                                                                     
connector.Role = string(types.ConnectorRoleInterRouter)                                                                                 
} else {                                                                                                                                        
connector.Host = secret.ObjectMeta.Annotations["edge-host"]                                                                             
connector.Port = secret.ObjectMeta.Annotations["edge-port"]                                                                             
connector.Role = string(types.ConnectorRoleEdge)                                                                                        
}                                                                                                                                               
qdr.AddConnector(&connector, current)                                                                                                           
_, err := cli.KubeClient.AppsV1().Deployments(options.SkupperNamespace).Update(current)

```

Basically all this is doing is updating a file pointed by an environment variable (QDROUTERD_CONF) in the "bridge-server" pod, in the skupper-router deployment.
Then the deployment itself takes care of establishing the connection.

status says not connected for some reason:

```
[nbrignon@localhost skupper (sk59)]$ skupper status -n public
Skupper is enabled for namespace '"public"'. It is not connected to any other sites.
[nbrignon@localhost skupper (sk59)]$ skupper status -n private
Skupper is enabled for namespace '"private"'. It is not connected to any other sites.
```

## exposing deployment as service in all connected skupper sites.

First just deploying a simple app that listen and respond on port 9090.
```
kubectl apply -f ${HOME}/tcp-echo/public-deployment.yaml
```

we are creating and exposing the deployment from the public site, and we expect to see the new service also in the private site, it could be done exactly in the other direction, from private to public.

```
$ skupper expose --port 9090 deployment tcp-go-echo
```
all skupper-expose does is updating the "skupper-services" config map which is monitored by the skupper-services-controller. The rest will be done when the controller is informed.
```
[nbrignon@localhost skupper (sk59)]$ kubectl get cm skupper-services -o yaml
apiVersion: v1
data:
  tcp-go-echo: '{"address":"tcp-go-echo","protocol":"tcp","port":9090,"targets":[{"selector":"app.kubernetes.io/name=tcp-go-echo-example"}]}'
kind: ConfigMap
metadata:
  creationTimestamp: "2020-06-24T14:56:07Z"
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:data:
        .: {}
        f:tcp-go-echo: {}
    manager: skupper
    operation: Update
    time: "2020-06-24T21:03:10Z"
  name: skupper-services
  namespace: public
  ownerReferences:
  - apiVersion: v1
    kind: ConfigMap
    name: skupper-site
    uid: 72121dd3-ea02-46f9-bde8-8d5c936709db
  resourceVersion: "87169"
  selfLink: /api/v1/namespaces/public/configmaps/skupper-services
  uid: d5a5571e-e96d-4448-a83e-19006baa0f04
```













# gsim notes
The site controller is optional. It offers an alternative way to setup sites and join them in/to a skupper network.

The service-controller is just the controller from prior to this PR. I renamed it to make its role clearer wrt to the new optional controller. The service controller is responsible for ensuring that the bridges/proxies are configured correctly to implement the connectivity desired for skupper services. It takes as input the skupper-services config map. It will also populate that map based on annotated deployments or services.

The cli does both site management and service management. However for the latter it just populates skupper-services, and requires the service-controller to actually setup and maintain the necessary data-plane configuration.

The site controller works by watching for:

(a) a configmap called skupper-site, which causes the namespace to be configured for skupper (i.e. essentially it then automatically does what skupper init would do from cli)

(b) a secret with label skupper.io/type=connection-token, which causes it to establish a connection based on that token if it does not already exist (this is what the cli would do if skupper connect was invoked)

(c) a secret with label skupper.io/type=connection-token-request, which causes it to generate a token and write it in to that secret (this is an alternative to skupper connection-token with cli).

Deletion of skupper can be achieved by just deleting the skupper-site configmap. This is the case whether you initialised it with the cli or the site controller did so. The cli can also still be used of course, and skupper delete will in general just delete that configmap and allow the other 'owned' objects to be automatically deleted by kubernetes.
