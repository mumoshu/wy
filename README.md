# wy

`wy` (Abbreviation of `Would You`) is a set of command-line tools to test your container-based platform.

Currently, it provides the following commands:

- [`serve`](#serve)
- [`get`](#get)
- [`repeat get`](#repeat-get)

`serve` is intended to be run inside containers and Kubernetes pods, so that you can interact with it with `wy get` and see e.g. Datadog, Prometheus, Grafana dashboards to see if it works.

## serve

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

## get

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

## repeat get

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

## Related Projects

This project has been inspired by the following projects. Thanks for the authors!

- https://github.com/a-h/slowloris
