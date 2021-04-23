package promexp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dachad/tcpgoon/debugging"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	webhtml = []byte(`
	<html>
		<head>
			<title>tcpgoon Exporter</title>
			<style>
				label{
					display:inline-block;
					width:75px;
				}
				form label {
					margin: 10px;
				}
				form input {
					margin: 10px;
				}
			</style>
		</head>
		<body>
		<h1>tcpgoon Exporter</h1>
		<form action="/tcpgoon">
			<label>Target:</label> <input type="text" name="target_ip" placeholder="X.X.X.X"><br>
			<label>Target Port:</label> <input type="text" name="target_port" placeholder="8080"><br>
			<label>Connection Count:</label> <input type="text" name="connections" placeholder="100"><br>
			<label>Sleep:</label> <input type="text" name="sleep" placeholder="10"><br>
			<input type="submit" value="Submit">
		</form>
		</body>
	</html>
	`)

	RequestMalformedErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcpgoon_request_malformed_params_total",
			Help: "Number of requests with malformed params",
		},
	)

	RequestInvalidParamsErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "tcpgoon_request_invalid_params_total",
			Help: "Number of requests with params that are not present in the config or did not pass parameter validation",
		},
	)
	queryParams = [...]string{"target_ip", "target_port", "connections", "sleep"}
)

func checkQueryParamsPresent(q url.Values) []error {
	errs := make([]error, 0, len(queryParams))
	for _, param := range queryParams {
		if q[param] == nil {
			errs = append(errs, fmt.Errorf("Param '%s' must be specified", param))
		}
		if len(q[param]) > 1 {
			errs = append(errs, fmt.Errorf("Param '%s' can only be specified once", param))
		}
	}
	if len(errs) > 1 {
		RequestMalformedErrors.Inc()
	}
	return errs
}

func checkQueryParamsValid(q url.Values) []error {
	errs := make([]error, 0, len(queryParams))

	addrs, err := net.LookupIP(q.Get("target_ip"))
	if err != nil || len(addrs) == 0 {
		return append(errs, errors.New("Param 'target_ip' is not a valid IP address or not resolvable"))
	}

	i, err := strconv.Atoi(q.Get("connections"))
	if err != nil && i < 0 && i > 65536 {
		errs = append(errs, errors.New("Param 'target_port' is not a valid port number"))
	}

	for _, param := range queryParams[2:] {
		i, err := strconv.Atoi(q.Get(param))
		if err != nil && i < 0 {
			errs = append(errs, fmt.Errorf("Param %s is not a valid positive number", param))
		}
	}

	if len(errs) > 0 {
		RequestInvalidParamsErrors.Inc()
	}
	return errs
}

func handleRequestErrors(errs []error, w http.ResponseWriter) {

	errorString := strings.Join(func() []string {
		errorStrings := make([]string, len(errs))
		for i, err := range errs {
			errorStrings[i] = err.Error()
		}
		return errorStrings
	}(), "\n")

	for _, err := range errs {
		fmt.Fprintln(debugging.DebugOut, "bad_request", err)
	}

	http.Error(w, errorString, 400)
}

func tcpgoonRequestHandler(w http.ResponseWriter, r *http.Request, connDialTimeout int) {
	query := r.URL.Query()

	fmt.Fprintln(debugging.DebugOut, "request_param", fmt.Sprint(query), "remote", r.RemoteAddr)
	errsQueryParamsPresent := checkQueryParamsPresent(query)
	if len(errsQueryParamsPresent) > 0 {
		handleRequestErrors(errsQueryParamsPresent, w)
		return
	}

	errsQueryParamsValid := checkQueryParamsValid(query)
	if len(errsQueryParamsValid) > 0 {
		handleRequestErrors(errsQueryParamsValid, w)
		return
	}

	start := time.Now()
	registry := prometheus.NewRegistry()
	registry.MustRegister(establishedCons, maxConcurrentCons, establishedConsOnClosure, minResponseTimeSecs, maxResponseTimeSecs, avgResponseTimeSecs, devResponseTimeSecs, invConnections)

	targetPort, _ := strconv.Atoi(query.Get("target_port"))
	connections, _ := strconv.Atoi(query.Get("connections"))
	sleep, _ := strconv.Atoi(query.Get("sleep"))

	collector := NewCollector(
		query.Get("target_ip"),
		targetPort,
		connections,
		connDialTimeout,
		sleep,
	)

	registry.MustRegister(collector)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
	duration := time.Since(start).Seconds()
	fmt.Fprintln(debugging.DebugOut, "msg", "Finished scrape", "duration_seconds", duration)

}

// RunHTTP starts a http server listening for exporter requests
func RunHTTP(listenAddress string, connDialTimeout int) {
	prometheus.MustRegister(RequestMalformedErrors)
	prometheus.MustRegister(RequestInvalidParamsErrors)

	fmt.Fprintln(debugging.DebugOut, "msg", "registering handler /tcpgoon")
	http.HandleFunc("/tcpgoon", func(w http.ResponseWriter, r *http.Request) {
		tcpgoonRequestHandler(w, r, connDialTimeout)
	})

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(webhtml)
	})

	fmt.Fprintln(debugging.DebugOut, "msg", "Starting http server")
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		fmt.Fprintln(debugging.DebugOut, "msg", "Error starting HTTP server", "err", err)
	}
}
