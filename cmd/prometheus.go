package cmd

import (
	"errors"
	"os"
	"strconv"

	"github.com/dachad/tcpgoon/debugging"
	"github.com/dachad/tcpgoon/promexp"

	"github.com/spf13/cobra"
)

type prometheusParams struct {
	port  int
	debug bool
}

var prometheusparams prometheusParams

var prometheusCmd = &cobra.Command{
	Use:   "prometheus [flags] <prometheus listening port>",
	Short: "Run tcpgoon in prometheus exporter mode",
	Long:  ``,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := validatePrometheusArgs(&prometheusparams, args); err != nil {
			cmd.Println(err)
			cmd.Println(cmd.UsageString())
			os.Exit(1)
		}
		if prometheusparams.debug {
			debugging.EnableDebug()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		runPrometheus(prometheusparams)
	},
}

func init() {
	prometheusCmd.Flags().BoolVarP(&prometheusparams.debug, "debug", "d", false, "Print debugging information to the standard error")
}

func validatePrometheusArgs(params *prometheusParams, args []string) error {
	if len(args) != 1 {
		return errors.New("Number of required parameters doesn't match")
	}
	port, err := strconv.Atoi(args[0])
	if err != nil && port <= 0 {
		return errors.New("Port argument is not a valid integer")
	}
	params.port = port

	return nil
}

func runPrometheus(params prometheusParams) {
	promexp.RunHTTP("0.0.0.0:" + strconv.Itoa(params.port))
}
