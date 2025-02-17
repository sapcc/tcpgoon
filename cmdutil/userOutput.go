package cmdutil

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dachad/tcpgoon/mtcpclient"
)

func printClosureReport(ip string, host string, port int, gc mtcpclient.GroupOfConnections) {
	// workaround to allow last status updates - messages in channels - to be collected properly
	// TODO: This can be fixed with an extra channel
	const timeToWaitForClosureReportInMs = 100
	time.Sleep(time.Duration(timeToWaitForClosureReportInMs) * time.Millisecond)

	target := host
	if host != ip {
		target = host + "(" + ip + ")"
	}

	fmt.Println(strings.Repeat("-", 3), target+":"+strconv.Itoa(port), "tcp test statistics", strings.Repeat("-", 3))
	mtcpclient.ReportConnectionsStatus(gc, 0)
	fmt.Println(mtcpclient.NewFinalMetricsReport(gc).CliReport())
}

func AskForUserConfirmation(host string, port int, connections int) bool {
	fmt.Println("****************************** WARNING ******************************")
	fmt.Println("* You are going to run a TCP stress check with these arguments:")
	fmt.Println("*	- Host: " + host)
	fmt.Println("*	- TCP Port: " + strconv.Itoa(port))
	fmt.Println("*	- # of concurrent connections: " + strconv.Itoa(connections))
	fmt.Println("*********************************************************************")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Do you want to continue? (y/N): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Response not processed")
			os.Exit(1)
		}

		response = strings.TrimSuffix(response, "\n")
		response = strings.ToLower(response)
		switch {
		case stringInSlice(response, []string{"yes", "y"}):
			return true
		case stringInSlice(response, []string{"no", "n", ""}):
			return false
		default:
			fmt.Println("\nSorry, response not recognized. Try again, please")
		}
	}
}
