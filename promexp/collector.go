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
	establishedCons = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "established_connections_count",
		Help: "Number of totally established connections",
	}, labels)
	maxConcurrentCons = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "max_concurrent_connections_count",
		Help: "Max concurrent established connections",
	}, labels)
	establishedConsOnClosure = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "established_connections_on_closure_count",
		Help: "Number of established connections on closure",
	}, labels)
	minResponseTimeSecs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "min_response_time_secs",
		Help: "Minimum wait for SYN-ACK",
	}, labels)
	maxResponseTimeSecs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "max_response_time_secs",
		Help: "Maximum wait for SYN-ACK",
	}, labels)
	avgResponseTimeSecs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "avg_response_time_secs",
		Help: "Average wait for SYN-ACK",
	}, labels)
	devResponseTimeSecs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "stddev_response_time_secs",
		Help: "Standard deviation of wait for SYN-ACK",
	}, labels)
	invConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: prefix + "attempted_connection_count",
		Help: "Number of connections attempted to connect",
	}, labels)
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

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	tcpclient.DefaultDialTimeoutInMs = c.connDialTimeout

	connStatusCh, connStatusTracker := mtcpclient.StartBackgroundReporting(c.numberConnections, 0)
	closureCh := mtcpclient.StartBackgroundClosureTrigger(*connStatusTracker)
	mtcpclient.MultiTCPConnect(c.numberConnections, c.delay, c.targetIp, c.targetPort, connStatusCh, closureCh)
	fmt.Fprintln(debugging.DebugOut, "Tests execution completed")
	labelValues := []string{c.targetIp, strconv.Itoa(c.targetPort), strconv.Itoa(c.delay), strconv.Itoa(c.connDialTimeout)}
	fmr := mtcpclient.NewFinalMetricsReport(*connStatusTracker)
	mr := fmr.SuccessfulConnectionReport()

	thisEstablishedCons := establishedCons.WithLabelValues(labelValues...)
	thisEstablishedCons.Set(float64(fmr.EstablishedCons()))

	thisMaxConcurrentCons := maxConcurrentCons.WithLabelValues(labelValues...)
	thisMaxConcurrentCons.Set(float64(fmr.MaxConcurrentCons()))

	thisEstablishedConsOnClosure := establishedConsOnClosure.WithLabelValues(labelValues...)
	thisEstablishedConsOnClosure.Set(float64(fmr.EstablishedConsOnClosure()))

	thisMinResponseTimeSecs := minResponseTimeSecs.WithLabelValues(labelValues...)
	thisMinResponseTimeSecs.Set(mr.Min().Seconds())

	thisMaxResponseTimeSecs := maxResponseTimeSecs.WithLabelValues(labelValues...)
	thisMaxResponseTimeSecs.Set(mr.Max().Seconds())

	thisAvgResponseTimeSecs := avgResponseTimeSecs.WithLabelValues(labelValues...)
	thisAvgResponseTimeSecs.Set(mr.Avg().Seconds())

	thisDevResponseTimeSecs := devResponseTimeSecs.WithLabelValues(labelValues...)
	thisDevResponseTimeSecs.Set(mr.StdDev().Seconds())

	thisInvConnections := invConnections.WithLabelValues(labelValues...)
	thisInvConnections.Set(float64(c.numberConnections))

	ch <- thisEstablishedCons
	ch <- thisMaxConcurrentCons
	ch <- thisEstablishedConsOnClosure
	ch <- thisMinResponseTimeSecs
	ch <- thisMaxResponseTimeSecs
	ch <- thisAvgResponseTimeSecs
	ch <- thisDevResponseTimeSecs
	ch <- thisInvConnections
}
