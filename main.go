package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/anthhub/forwarder"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	appVersion string
	version    = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "version",
		Help: "Version information about this binary",
		ConstLabels: map[string]string{
			"version": appVersion,
		},
	})

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Duration of all HTTP requests",
	}, []string{"code", "handler", "method"})
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

const (
	appName = "wy"
)

func run(args []string) error {
	fs := flag.NewFlagSet(appName, flag.ExitOnError)

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	switch c := fs.Arg(0); c {
	case "print":
		return print(fs.Args()[1:])
	case "serve":
		return serve(fs.Args()[1:])
	case "get":
		return get(fs.Args()[1:])
	case "repeat":
		return repeat(fs.Args()[1:])
	}

	fmt.Fprintf(os.Stderr, "Command %q does not exist\n\nAvailable commands:\n  serve\n  get\n  repeat\n", fs.Arg(0))
	fs.Usage()
	return nil
}

func repeat(args []string) error {
	fs := flag.NewFlagSet("repeat", flag.ExitOnError)

	var (
		count    int
		interval time.Duration
		forever  bool

		argocdClusterSecret string
		service             string
		localPort           int
		remotePort          int
		kubeconfigPath      string
	)

	fs.IntVar(&count, "count", 5, "Number of repetitions")
	fs.DurationVar(&interval, "interval", time.Second, "Delay between each request")
	fs.BoolVar(&forever, "forever", false, "Repeat HTTP requests infinite number of times. If true, -count is ignored")
	fs.StringVar(&argocdClusterSecret, "argocd-cluster-secret", "", "Name of the Kubernetes secret that contains an ArgoCD-style cluster connection info. If specified, it uses port-forwarding to access the target server")
	fs.StringVar(&service, "service", "", "Name of the Kubernetes service that is connected to the pods. Required if you'd want access the app via Kubernetes port-forwarding")
	fs.IntVar(&localPort, "local-port", 8080, "Port part of the URL to the server")
	fs.IntVar(&remotePort, "remote-port", 8080, "Port part of the URL to the server")
	fs.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to the kubeconfig file for port-forwarding")

	cmd := args[0]

	switch cmd {
	case "get":
		url, print, err := getFlags(fs, args[1:])
		if err != nil {
			return err
		}

		if service != "" {
			options := []*forwarder.Option{
				{
					LocalPort:   localPort,
					RemotePort:  remotePort,
					ServiceName: service,
				},
			}
			// forwarder requires rest config without argocd's custom transport
			// hence we call getClusterRestConfig instead of getRestConfig
			restConfig, err := getClusterRestConfig(kubeconfigPath, argocdClusterSecret)
			if err != nil {
				return err
			}
			ret, err := forwarder.WithRestConfig(context.Background(), options, restConfig)
			if err != nil {
				return err
			}
			defer ret.Close()
			_, err = ret.Ready()
			if err != nil {
				return err
			}
		}

		client := &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		}

		var i int

		for {
			if !forever {
				if i >= count {
					break
				}
				i++
			}

			if err := httpGet(client, url, print); err != nil {
				return err
			}

			time.Sleep(interval)
		}

		return nil
	}

	fmt.Fprintf(os.Stderr, "Command %q does not exist\nAvailable commands:\n  get\n", cmd)
	fs.Usage()

	return nil
}

func getFlags(fs *flag.FlagSet, args []string) (string, bool, error) {
	var (
		url   string
		print bool
	)

	fs.BoolVar(&print, "print", true, "Print response body to stdout")
	fs.StringVar(&url, "url", "http://localhost:8080/", "The URL to where send request")

	if err := fs.Parse(args); err != nil {
		return "", false, err
	}

	fmt.Fprintf(os.Stdout, "%v\n", fs.Args())

	return url, print, nil
}

func get(args []string) error {
	fs := flag.NewFlagSet("get", flag.ExitOnError)

	url, print, err := getFlags(fs, args)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	return httpGet(client, url, print)
}

func httpGet(client *http.Client, url string, print bool) error {
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(nil))
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if print {
		all, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, string(all)+"\n")
	}

	return nil
}

func serve(args []string) error {
	version.Set(1)
	bind := ""
	enableH2c := false

	var (
		delayBeforeHeader, delayBeforeFirstByte, delayBeforeLastByte time.Duration
	)

	fs := flag.NewFlagSet(appName, flag.ExitOnError)
	fs.StringVar(&bind, "bind", ":8080", "The socket to bind to.")
	fs.BoolVar(&enableH2c, "h2c", false, "Enable h2c (http/2 over tcp) protocol.")
	fs.DurationVar(&delayBeforeHeader, "delay-header-first-byte", 0, "")
	fs.DurationVar(&delayBeforeFirstByte, "delay-body-first-byte", 0, "")
	fs.DurationVar(&delayBeforeLastByte, "delay-body-last-byte", 0, "")

	if err := fs.Parse(args); err != nil {
		return err
	}

	r := prometheus.NewRegistry()
	r.MustRegister(httpRequestsTotal)
	r.MustRegister(httpRequestDuration)
	r.MustRegister(version)

	var requestCount int32

	flush := func(w http.ResponseWriter) {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	foundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := atomic.AddInt32(&requestCount, 1)

		time.Sleep(delayBeforeHeader)
		w.WriteHeader(http.StatusOK)
		flush(w)

		data := []byte(fmt.Sprintf("Hello from okra example application.: %d", id))

		time.Sleep(delayBeforeFirstByte)
		w.Write(data[:1])
		flush(w)

		time.Sleep(delayBeforeLastByte)
		w.Write(data[1:])
		flush(w)
	})

	notfoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	errHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	mux := http.NewServeMux()

	mux.Handle("/", promhttp.InstrumentHandlerDuration(
		httpRequestDuration.MustCurryWith(prometheus.Labels{"handler": "found"}),
		promhttp.InstrumentHandlerCounter(httpRequestsTotal, foundHandler),
	))
	mux.Handle("/404", promhttp.InstrumentHandlerCounter(
		httpRequestsTotal,
		notfoundHandler,
	))
	mux.Handle("/500", promhttp.InstrumentHandlerCounter(
		httpRequestsTotal,
		errHandler,
	))

	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	var srv *http.Server
	if enableH2c {
		srv = &http.Server{Addr: bind, Handler: h2c.NewHandler(mux, &http2.Server{})}
	} else {
		srv = &http.Server{Addr: bind, Handler: mux}
	}

	return srv.ListenAndServe()
}

func print(args []string) error {
	if args[0] != "kubeconfig" {
		return fmt.Errorf("the only supported print sub-command is \"kubeconfig\", but you provided %q", args[0])
	}

	args = args[1:]

	var (
		argocdClusterSecret string
		kubeconfigPath      string
		setNamespace        string
	)

	fs := flag.NewFlagSet(fmt.Sprintf("%s-print-kubeconfig", appName), flag.ExitOnError)
	fs.StringVar(&argocdClusterSecret, "argocd-cluster-secret", "", "Name of the Kubernetes secret that contains an ArgoCD-style cluster connection info. If specified, it uses port-forwarding to access the target server")
	fs.StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to the kubeconfig file for port-forwarding")
	fs.StringVar(&setNamespace, "set-namespace", "default", "Namespace to be set in the default context of the generated kubeconfig")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if argocdClusterSecret == "" {
		return fmt.Errorf("missing value for the required flag %s", "-argocd-cluster-secret")
	}

	kubeconfigData, err := getKubeconfig(kubeconfigPath, argocdClusterSecret, setNamespace)
	if err != nil {
		return err
	}

	n, err := os.Stdout.Write(kubeconfigData)
	if err != nil {
		return fmt.Errorf("failed to write remaining %d bytes: %w", n, err)
	}

	return nil
}
