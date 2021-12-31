# wy

`wy` (Abbreviation of `Would You`) is a set of command-line tools to test your container-based platform.

ToC:

- [Commands](#commands)
- [Deployment](#deployment)
- [Monitoring](#monitoring)
- [Contributing](#contributing)

## Commands

Currently, it provides the following commands:

- [`serve`](#serve)
- [`get`](#get)
- [`repeat get`](#repeat-get)
- [`print kubeconfig`](#print-kubeconfig) (for exporting ArgoCD cluster secret as kubeconfig)

`serve` is intended to be run inside containers and Kubernetes pods, so that you can interact with it with `wy get` and see e.g. Datadog, Prometheus, Grafana dashboards to see if it works.

### serve

This comand starts a long-running http server with variouos useful options for testing.

```
$ wy serve -h
Usage of wy:
  -bind string
        The socket to bind to. (default ":8080")
  -delay-body-first-byte duration
    
  -delay-body-last-byte duration
    
  -delay-header-first-byte duration
    
  -h2c
        Enable h2c (http/2 over tcp) protocol.
```

### get

This command sends a single HTTP GET request against the server.

The default URL is that of `wy serve` is running locally with the default configuration.
If you had customize `wy serve` options, or it's behind a loadbalancer, set `-url` accordingly.

```
$ wy get -h
Usage of wy:
  -print
        Print response body to stdout (default true)
  -url string
        The URL to where send request (default "http://localhost:8080/")
```

Another use-case of this command is to print all the metrics exposed by the server with [the exposition fomrat](https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md):

```shell
$ wy repeat get -count 5 -url http://localhost:8080
$ wy repeat get -count 10 -url http://localhost:8080/404
$ wy repeat get -count 15 -url http://localhost:8080/500
$ wy get -url http://localhost:8080/metrics
...snip...
# HELP http_requests_total Count of all HTTP requests
# TYPE http_requests_total counter
http_requests_total{code="200",method="get"} 5
http_requests_total{code="404",method="get"} 10
http_requests_total{code="500",method="get"} 15
...snip...
```

### repeat get

This command repeatedly runs `wy get` so that the server emits more realistic metrics.

```
$ wy repeat get -h
Usage of repeat:
  -argocd-cluster-secret string
        Name of the Kubernetes secret that contains an ArgoCD-style cluster connection info. If specified, it uses port-forwarding to access the target server
  -count int
        Number of repetitions (default 5)
  -forever
        Repeat HTTP requests infinite number of times. If true, -count is ignored
  -interval duration
        Delay between each request (default 1s)
  -kubeconfig string
        Path to the kubeconfig file for port-forwarding (default "kubeconfig.okra")
  -local-port int
        Port part of the URL to the server (default 8080)
  -print
        Print response body to stdout (default true)
  -remote-port int
        Port part of the URL to the server (default 8080)
  -service string
        Name of the Kubernetes service that is connected to the pods. Required if you'd want access the app via Kubernetes port-forwarding
  -url string
        The URL to where send request (default "http://localhost:8080/")
```

### print kubeconfig

```
Usage of wy-print-kubeconfig:
  -argocd-cluster-secret string
        Name of the Kubernetes secret that contains an ArgoCD-style cluster connection info. If specified, it uses port-forwarding to access the target server
  -kubeconfig string
        Path to the kubeconfig file for port-forwarding (default "kubeconfig.okra")
  -set-namespace string
        Namespace to be set in the default context of the generated kubeconfig (default "default")
```

This command is useful in a scenario that you want to interact with a specific cluster registered to Argo CD:

```
# Deploy wy-serve onto the cluster1 cluster

$ wy print kubeconfig -argocd-cluster-secret cluster1 > kubeconfig.cluster1
$ KUBECONFIG=kubeconfig.cluster1 kubectl apply -f wy-serve.yaml

# Deploy wy client onto another cluster

$ wy print kubeconfig -argocd-cluster-secret cluster2 > kubeconfig.cluster2
$ KUBECONFIG=kubeconfig.cluster2 kubectl apply -f wy-serve.yaml
```

The combination of `wy print kubeconfig` and `kubectl apply` is convenient in order to give it a try with
[wy repeat get -forever](#calling-wy-serve-using-wy-repeat-in-a-kubernetes-cluster).

For example, a long-running `wy` client command that periodically calls `wy-serve` deployed onto `cluster1` mentioned above would look like:

```
$ wy repeat get -forever \
  -argocd-cluster-secret cluster1 -service wy-serve \
  -remote-port 8080 -local-port 8080 \
  -url http://localhost:8080

2021/12/31 08:05:49 Using kubeconfig-based Kubernetes API client
Forwarding service: wy-serve to pod wy-serve-c958ff7df-v95gr ...
Forwarding from 127.0.0.1:8080 -> 8080
Forwarding from [::1]:8080 -> 8080
Handling connection for 8080
Hello from okra example application.: 1
Hello from okra example application.: 2
Hello from okra example application.: 3
...
```

## Deployment

- [Deploy wy-serve onto a Kubernetes cluster](#deploy-wy-serve-onto-a-kubernetes-cluster)
- [Calling wy-serve using wy-repeat-get](#calling-wy-serve-using-wy-repeat-get)
- [Calling wy-serve using wy-repeat in a Kubernetes cluster](#calling-wy-serve-using-wy-repeat-in-a-kubernetes-cluster)

### Deploy wy-serve onto a Kubernetes cluster

Technically speaking, `wy` can be deployed onto any platform based on baremetal or VMs or containers.

That said, I'll show you an example for deploying it onto a Kubernetes cluster.

First, create the deployment YAML using `kubectl` command:

```
$ kubectl create deployment --image mumoshu/wy:latest --port=8080 --replicas=1 --dry-run=client -o=yaml wy-serve > wy-serve.yaml
$ echo '---' >> wy-serve.yaml
$ kubectl create service nodeport --node-port=30080 --tcp=8080:8080 --dry-run=client -o=yaml wy-serve >> wy-serve.yaml
```

Second, update the deployment's template.metadata.annotations so that a metrics agent can scape metrics from the metrics endpoint.

```yaml
annotations:
  prometheus.io/scrape: "true"
```

This gives you the following manifest file:

<details>

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: wy-serve
  name: wy-serve
spec:
  replicas: 1
  selector:
    matchLabels:
      app: wy-serve
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: wy-serve
      annotations:
        prometheus.io/scrape: "true"
    spec:
      containers:
      - image: mumoshu/wy:latest
        name: wy
        command:
        - /wy
        args:
        - serve
        ports:
        - containerPort: 8080
        resources: {}
status: {}
---
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: wy-serve
  name: wy-serve
spec:
  ports:
  - name: 8080-8080
    nodePort: 30080
    port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    app: wy-serve
  type: NodePort
status:
  loadBalancer: {}
```
</details>

This is how you'd deploy it:

```shell
$ kubectl apply -f wy-serve.yaml
```

Expose the pods via the NodePort by creating a external loadbalancer. For AWS, you'd use an ALB where the target group is associated to the nodeport of 30080.

### Calling wy-serve using wy-repeat-get

Finally, try accessing it to verify that it's actually working or not:

```shell
# Via an external loadbalancer
$ wy repeat get -count 5 -url http://$AWS_ALB_HOST:30080/

# Via the clusterIP service
$ POD_NAME=$(kubectl get po -l app=wy-serve -o json | jq -r .items[0].metadata.name)
$ kubectl exec -it ${POD_NAME} --\
  /wy repeat get -count 5 -url http://wy-serve.default.svc.cluster.local:8080

# Via the clusterIP over port-forward to the cluster associated to $KUBECONFIG
$ wy repeat get -count 5 -url http://localhost:8080 \
  -service wy-serve -remote-port 8080 -local-port 8080

# Via the clusterIP over port-forward to the cluster registered to ArgoCD
$ wy repeat get -count 5 -url http://localhost:8080 \
  -argocd-cluster-secret mycluster1 \
  -service wy-serve  -remote-port 8080 -local-port 8080

# Review metrics
$ kubectl run -it --rm --image ubuntu:latest ubuntu1 -- /bin/bash -c \
  'apt update && apt install -y curl && curl wy-serve.default.svc.cluster.local:8080/metrics'
```

### Calling wy-serve using wy-repeat in a Kubernetes cluster

There are a few cases you'd want to continuously run `wy repeat get` for a certain period from a remote machine/cluster:

- You want to use `wy` as a part of a canary experiment/analysis step (e.g. Argo Rollouts and Flagger)
- You want to simulate some user traffic on a test cluster

`wy repeat get` has a `-forever` flag that disables `-count` and hence makes `wy repeat get` running forever, periodically sending HTTP requests to the target URL. It sends a request for each `1s` by default and you can change the interval by specifing `-interval DURATION`, where `DURATION` can be a Golang's `time` package style duration string like `1s` for a second and `2m` for 2 minutes, `3h` for 3 hours, and so on.

The following example shows the command that sends a request per each 5 seconds:

```shell
wy repeat get -forever -interval 5s -url http://localhost:8080 \
  -argocd-cluster-secret mycluster1 \
  -service wy-serve  -remote-port 8080 -local-port 8080
```

Back to the original goal, the above command can be run from a Kubernetes cluster by turning it into a Kubernetes deployment, where the `command` and `args` of the `wy` container reflects the above example.

To scaffold our YAML, run:

```
kubectl create deployment --image mumoshu/wy:latest --port=8080 --replicas=1 --dry-run=client -o=yaml wy > wy.yaml
```

Open the generated `wy.yaml` in an editor and add `command` and `args` to the primary container in the pod template:

```
kind: Deployment
spec:
  template:
    spec:
      containers:
      - image: mumoshu/wy:latest
        name: wy
        ports:
        - containerPort: 8080
        resources: {}
        command:
        - wy
        - repeat
        - get
        args:
        - -forever
        - -interval=5s
        - -url=http://localhost:8080
        - -argocd-cluster-secret=mycluster1
        - -service=wy-serve
        - -remote-port=8080
        - -local-port=8080
```

Run `kubectl apply -f wy.yaml` to deploy it and see it works!

If it doesn't run `kubectl logs $POD` and see what's happening.

If you see permission errors, try granting some K8s API permissions required by `wy`:

```
2021/12/31 05:34:00 Using in-cluster Kubernetes API client
2021/12/31 05:34:00 secrets "mycluster1" is forbidden: User "system:serviceaccount:default:default" cannot get resource "secrets" in API group "" in the namespace "default"
```

Usually it's just a `get secret` permission required when you specified `-argocd-cluster-secret`

```
NS=default
SA=default

kubectl create role wy --verb=get --resource=secret --dry-run=client -o yaml > wy.rbac.yaml
echo '---' >> wy.rbac.yaml
kubectl create rolebinding wy --role=wy --serviceaccount=${NS}:${SA} --dry-run=client -o yaml >> wy.rbac.yaml
```

Similarly, if you encounter AWS API permission issue, it's usually you're missing any AWS credentials to be used by `wy`:

```
2021/12/31 05:37:23 Using in-cluster Kubernetes API client
Error: NoCredentialProviders: no valid providers in chain. Deprecated.
        For verbose messaging see aws.Config.CredentialsChainVerboseErrors
Usage:
  aws eks get-token [flags]

Flags:
      --cluster-name string   Specify the name of the Amazon EKS  cluster to create a token for.
  -h, --help                  help for get-token
      --role-arn string       Assume this role for credentials when signing the token.

2021/12/31 05:37:30 NoCredentialProviders: no valid providers in chain. Deprecated.
        For verbose messaging see aws.Config.CredentialsChainVerboseErrors
2021/12/31 05:37:30 Get "https://SOME_ID.gr7.REGION.eks.amazonaws.com/api/v1/namespaces/default/services/wy-serve": getting credentials: exec: executable aws failed with exit code 1
```

In this case, add the following snippet to your container spec:

```
envFrom:
- secretRef:
    name: wy
    optional: true
```

Create a secret with:

```
cat <<EOS > wy.env && kubectl create secret generic wy --from-env-file=wy.env -o yaml --dry-run=client > wy.secret.yaml
AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
EOS

kubectl apply -f wy.secret.yaml
```

## Monitoring

`wy serve` exposes various Prometheus metrics via the exposition format.
It's just a specifially crafted HTTP endpoint so that it can be easily scraped by Prometheus or other products and services that supports it.

One example of such services is [Datadog](https://www.datadoghq.com/).
Now, let's set it up so that you can view metrics in your Datadog dashboard.

First, install the Datadog agent onto your cluster:

```shell
$ helm upgrade --install datadog datadog/datadog \
  --set datadog.apiKey=$DATADOG_API_KEY \
  --set datadog.site=datadoghq.com \
  --set datadog.clusterName=cluster1 \
  --set datadog.prometheusScrape.enabled=true \
  -f <(cat <<EOF
datadog:
  kubelet:
    tlsVerify: false
EOF
)
```

> Note that apparently `tlsVerify: false` is required only when your cluster local CA isn't
> accepted by Datadog Agent's kubelet check.
> It's usually the case when you tried to follow this process in a [kind](https://kind.sigs.k8s.io/).
>
> Also note that cluster local CA's cert is usually found at `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`
> in every pod.
>
> If you're curious what's in it, try `kubectl-exec`ing into any pod in your cluster and run
> ```
> openssl x509 -in /var/run/secrets/kubernetes.io/serviceaccount/ca.crt -text -noout
> ```
> so you'll be able to review it in a human-friendly format.

Hold on for just dozens of seconds and datadog-agent will be up and running, scraping and forwarding metrics from your `wy-serve` deployment to Datadog.

Browse Datadog dashboard and see the metrics!

## Contributing

We welcome your contribution!

Before submitting your change as a pull request, I'd highly recommend testing it youself.

We provide a few make targets to help that.

First, build a custom docker image with:

```
$ NAME=$DOCKER_USER/wy make docker-buildx
$ docker push $DOCKER_USER/wy:latest
```

Second, deploy wy from the custom image:

```
$ cat wy-serve.yaml | sed "s/mumoshu/$DOCKER_USER/" | kubectl apply -f -
```

Finally, run `wy get` against your custom `wy-serve` pods and see if it's really working as intended.

In case you'd like to customize `wy-serve.yaml`, you can use the following snippet for the foundation:

```shell
kubectl create deployment --image mumoshu/wy:latest --port=8080 --replicas=1 --dry-run=client -o=yaml wy-serve > wy-serve.yaml
echo '---' >> wy-serve.yaml
kubectl create service nodeport --node-port=30080 --tcp=8080:8080 --dry-run=client -o=yaml wy-serve >> wy-serve.yaml
```

## Related Projects

This project has been inspired by the following projects. Thanks for the authors!

- https://github.com/a-h/slowloris
