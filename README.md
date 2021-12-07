# wy

`wy` (Abbreviation of `Would You`) is a set of command-line tools to test your container-based platform.

Currently, it provides the following commands:

- `serve`
- `get`

`serve` is intended to be run inside containers and Kubernetes pods, so that you can interact with it with `wy get` and see e.g. Datadog, Prometheus, Grafana dashboards to see if it works.

## serve

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

## get

```
$ wy get -h
Usage of wy:
  -print
        Print response body to stdout (default true)
  -url string
        The URL to where send request (default "http://localhost:8080/")
```
