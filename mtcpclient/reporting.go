package mtcpclient

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/dachad/tcpgoon/tcpclient"
)

func collectConnectionsStatus(connectionsStatusRegistry *GroupOfConnections, statusChannel <-chan tcpclient.Connection) {
	concurrentEstablished := 0
	for {
		newConnectionStatusReported := <-statusChannel
		concurrentEstablished = updateConcurrentEstablished(concurrentEstablished, newConnectionStatusReported, connectionsStatusRegistry)
		connectionsStatusRegistry.connections[newConnectionStatusReported.ID] = newConnectionStatusReported
	}
}

func updateConcurrentEstablished(concurrentEstablished int, newConnectionStatusReported tcpclient.Connection, connectionsStatusRegistry *GroupOfConnections) int {
	if tcpclient.IsOk(newConnectionStatusReported) {
		concurrentEstablished++
		connectionsStatusRegistry.metrics.maxConcurrentEstablished = int(math.Max(float64(concurrentEstablished),
			float64(connectionsStatusRegistry.metrics.maxConcurrentEstablished)))
	} else if tcpclient.IsOk(connectionsStatusRegistry.connections[newConnectionStatusReported.ID]) {
		concurrentEstablished--
	}
	return concurrentEstablished
}

// ReportConnectionsStatus keeps printing on screen the summary of connections states
func ReportConnectionsStatus(gc GroupOfConnections, intervalBetweenUpdates int) {
	for {
		if intervalBetweenUpdates == 0 {
			break
		}
		fmt.Println(gc)
		time.Sleep(time.Duration(intervalBetweenUpdates) * time.Second)
	}
}

// StartBackgroundReporting starts some goroutines (so it's not blocking) to capture and report data from the tcpclient
// routines. It initializes and returns the channel that will be used for these communications
func StartBackgroundReporting(numberConnections int, rinterval int) (chan tcpclient.Connection, *GroupOfConnections) {
	// A connection may report up to 3 messages: Dialing -> Established -> Closed
	const maxMessagesWeMayGetPerConnection = 3
	connStatusCh := make(chan tcpclient.Connection, numberConnections*maxMessagesWeMayGetPerConnection)

	connStatusTracker := newGroupOfConnections(numberConnections)

	go ReportConnectionsStatus(*connStatusTracker, rinterval)
	go collectConnectionsStatus(connStatusTracker, connStatusCh)

	return connStatusCh, connStatusTracker
}

type FinalMetricsReport struct {
	establishedCons          int
	maxConcurrentCons        int
	establishedConsOnClosure int
	allConnections           GroupOfConnections
	connectionsOK            GroupOfConnections
	connectionsError         GroupOfConnections
}

func (f *FinalMetricsReport) EstablishedCons() int          { return f.establishedCons }
func (f *FinalMetricsReport) MaxConcurrentCons() int        { return f.maxConcurrentCons }
func (f *FinalMetricsReport) EstablishedConsOnClosure() int { return f.establishedConsOnClosure }

func NewFinalMetricsReport(gc GroupOfConnections) *FinalMetricsReport {
	return &FinalMetricsReport{
		establishedCons:          len(gc.getConnectionsThatWentWell(true).connections),
		maxConcurrentCons:        gc.metrics.maxConcurrentEstablished,
		establishedConsOnClosure: len(gc.getConnectionsThatAreOk().connections),
		allConnections:           gc,
		connectionsOK:            gc.getConnectionsThatWentWell(true),
		connectionsError:         gc.getConnectionsThatWentWell(false),
	}
}

func (fmr *FinalMetricsReport) SuccessfulConnectionReport() *metricsCollectionStats {
	return fmr.connectionsOK.calculateMetricsReport()
}

func (fmr *FinalMetricsReport) ErrorConnectionReport() *metricsCollectionStats {
	return fmr.connectionsError.calculateMetricsReport()
}

// FinalMetricsReport creates the final reporting summary
func (fmr *FinalMetricsReport) CliReport() (output string) {
	// Report Established Connections
	output += "--- tcpgoon execution statistics ---\n" +
		"Total established connections: " +
		strconv.Itoa(fmr.establishedCons) + "\n" +
		"Max concurrent established connections: " +
		strconv.Itoa(fmr.maxConcurrentCons) + "\n" +
		"Number of established connections on closure: " +
		strconv.Itoa(fmr.establishedConsOnClosure) + "\n"

	if fmr.allConnections.atLeastOneConnectionOK() {
		output += fmr.connectionsOK.pingStyleReport(successfulExecution)
	}
	if fmr.allConnections.AtLeastOneConnectionInError() {
		output += fmr.connectionsError.pingStyleReport(failedExecution)
	}

	return output
}
