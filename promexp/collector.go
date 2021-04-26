package promexp

import (
	"fmt"
	"net"
	"strconv"

	"github.com/dachad/tcpgoon/debugging"
	"github.com/dachad/tcpgoon/mtcpclient"
	"github.com/dachad/tcpgoon/tcpclient"
	"github.com/prometheus/client_golang/prometheus"
)

const prefix = "tcpgoon_"

var (
	labels          = []string{"target_ip", "target_port", "sleep_msecs", "timeout_msecs"}
	establishedCons = prometheus.NewDesc(
		prefix+"established_connections_count",
		"Number of totally established connections",
		labels, nil)
	maxConcurrentCons = prometheus.NewDesc(
		prefix+"max_concurrent_connections_count",
		"Max concurrent established connections",
		labels, nil)
	establishedConsOnClosure = prometheus.NewDesc(
		prefix+"established_connections_on_closure_count",
		"Number of established connections on closure",
		labels, nil)
	minResponseTimeSecs = prometheus.NewDesc(
		prefix+"min_response_time_secs",
		"Minimum wait for SYN-ACK",
		labels, nil)
	maxResponseTimeSecs = prometheus.NewDesc(
		prefix+"max_response_time_secs",
		"Maximum wait for SYN-ACK",
		labels, nil)
	avgResponseTimeSecs = prometheus.NewDesc(
		prefix+"avg_response_time_secs",
		"Average wait for SYN-ACK",
		labels, nil)
	devResponseTimeSecs = prometheus.NewDesc(
		prefix+"stddev_response_time_secs",
		"Standard deviation of wait for SYN-ACK",
		labels, nil)
	invConnections = prometheus.NewDesc(
		prefix+"attempted_connection_count",
		"Number of connections attempted to connect",
		labels, nil)
)

type Collector struct {
	targetPort        int
	targetIp          string
	targetName        string
	numberConnections int
	delay             int
	connDialTimeout   int
}

func NewCollector(targetName string, targetPort int, numberConnections int, delay int, connDialTimeout int) *Collector {
	addrs, _ := net.LookupIP(targetName)
	return &Collector{
		targetPort:        targetPort,
		targetIp:          addrs[0].String(),
		targetName:        targetName,
		numberConnections: numberConnections,
		delay:             delay,
		connDialTimeout:   connDialTimeout,
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- establishedCons
	ch <- maxConcurrentCons
	ch <- establishedConsOnClosure
	ch <- minResponseTimeSecs
	ch <- maxResponseTimeSecs
	ch <- avgResponseTimeSecs
	ch <- devResponseTimeSecs
	ch <- invConnections
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	tcpclient.DefaultDialTimeoutInMs = c.connDialTimeout

	connStatusCh, connStatusTracker := mtcpclient.StartBackgroundReporting(c.numberConnections, 0)
	closureCh := mtcpclient.StartBackgroundClosureTrigger(*connStatusTracker)
	mtcpclient.MultiTCPConnect(c.numberConnections, c.delay, c.targetIp, c.targetPort, connStatusCh, closureCh)
	fmt.Fprintln(debugging.DebugOut, "Tests execution completed")
	labelValues := []string{c.targetIp, strconv.Itoa(c.targetPort), strconv.Itoa(c.delay), strconv.Itoa(c.connDialTimeout)}
	fmr := mtcpclient.NewFinalMetricsReport(*connStatusTracker)
	mr := fmr.SuccessfulConnectionReport()

	ch <- prometheus.MustNewConstMetric(establishedCons, prometheus.GaugeValue, float64(fmr.EstablishedCons()), labelValues...)
	ch <- prometheus.MustNewConstMetric(maxConcurrentCons, prometheus.GaugeValue, float64(fmr.MaxConcurrentCons()), labelValues...)
	ch <- prometheus.MustNewConstMetric(establishedConsOnClosure, prometheus.GaugeValue, float64(fmr.EstablishedConsOnClosure()), labelValues...)
	ch <- prometheus.MustNewConstMetric(minResponseTimeSecs, prometheus.GaugeValue, mr.Min().Seconds(), labelValues...)
	ch <- prometheus.MustNewConstMetric(maxResponseTimeSecs, prometheus.GaugeValue, mr.Max().Seconds(), labelValues...)
	ch <- prometheus.MustNewConstMetric(avgResponseTimeSecs, prometheus.GaugeValue, mr.Avg().Seconds(), labelValues...)
	ch <- prometheus.MustNewConstMetric(devResponseTimeSecs, prometheus.GaugeValue, mr.StdDev().Seconds(), labelValues...)
	ch <- prometheus.MustNewConstMetric(invConnections, prometheus.GaugeValue, float64(c.numberConnections), labelValues...)
}
