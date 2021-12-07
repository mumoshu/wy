# wy

`wy` (Abbreviation of `Would You`) is a set of command-line tools to test your container-based platform.

ToC:

- [Commands](#commands)
- [Deployment](#deployment)

## Commands

Currently, it provides the following commands:

- [`serve`](#serve)
- [`get`](#get)
- [`repeat get`](#repeat-get)

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
  -count int
        Number of repetitions (default 5)
  -print
        Print response body to stdout (default true)
  -url string
        The URL to where send request (default "http://localhost:8080/")
```

## Deployment

Technically speaking, `wy` can be deployment onto any platform based on baremetal or VMs or containers.

That said, I'll show you an example for deploying it onto a Kubernetes cluster.

First, create the deployment YAML using `kubectl` command:

```
$ kubectl create deployment --image mumoshu/wy:latest --port=8080 --replicas=1 --dry-run=client -o=yaml wy-serve > wy-serve.yaml
$ echo '---' >> wy-serve.yaml
$ kubectl create service nodeport --node-port=30080 --tcp=8080:8080 --dry-run=client -o=yaml wy-serve >> wy-serve.yaml
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
    spec:
      containers:
      - image: mumoshu/wy:latest
        name: wy
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

Finally, try accessing it to verify that it's actually working or not:

```shell
$ wy repeat get -count 5 -url http://$AWS_ALB_HOST:30080/
```

## Related Projects

This project has been inspired by the following projects. Thanks for the authors!

- https://github.com/a-h/slowloris
