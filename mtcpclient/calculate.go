package mtcpclient

import (
	"math"
	"time"

	"github.com/dachad/tcpgoon/tcpclient"
)

type metricsCollectionStats struct {
	avg                 time.Duration
	min                 time.Duration
	max                 time.Duration
	total               time.Duration
	stdDev              time.Duration
	numberOfConnections int
}

func newMetricsCollectionStats() *metricsCollectionStats {
	mr := new(metricsCollectionStats)
	mr.avg = 0
	mr.min = time.Duration(time.Duration(tcpclient.DefaultDialTimeoutInMs) * time.Millisecond)
	mr.max = 0
	mr.total = 0
	mr.stdDev = 0
	mr.numberOfConnections = 0
	return mr
}

func (m *metricsCollectionStats) Avg() time.Duration       { return m.avg }
func (m *metricsCollectionStats) Min() time.Duration       { return m.min }
func (m *metricsCollectionStats) Max() time.Duration       { return m.max }
func (m *metricsCollectionStats) Total() time.Duration     { return m.total }
func (m *metricsCollectionStats) StdDev() time.Duration    { return m.stdDev }
func (m *metricsCollectionStats) NumberOfConnections() int { return m.numberOfConnections }

func (gc GroupOfConnections) calculateMetricsReport() (mr *metricsCollectionStats) {
	mr = newMetricsCollectionStats()
	if mr.numberOfConnections = len(gc.connections); mr.numberOfConnections > 0 {
		for _, item := range gc.connections {
			mr.min = time.Duration(math.Min(float64(mr.min), float64(item.GetTCPProcessingDuration())))
			mr.max = time.Duration(math.Max(float64(mr.max), float64(item.GetTCPProcessingDuration())))
			mr.total += item.GetTCPProcessingDuration()
		}
		mr.avg = mr.total / time.Duration(mr.numberOfConnections)
		mr.stdDev = gc.calculateStdDev(mr.avg)
	}
	return mr
}

func (gc GroupOfConnections) calculateStdDev(avg time.Duration) time.Duration {
	var sd float64

	if len(gc.connections) == 0 {
		return 0
	}

	for _, item := range gc.connections {
		sd += math.Pow(float64(item.GetTCPProcessingDuration())-float64(avg), 2)
	}

	return time.Duration(math.Sqrt(sd / float64(len(gc.connections))))
}
