package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tendermintLogger "github.com/tendermint/tendermint/libs/log"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"

	"github.com/metis-seq/themis/helper"
	"github.com/metis-seq/themis/version"
)

const (
	bridgeDBFlag       = "bridge-db"
	bridgeSqliteDBFlag = "bridge-sqlite-db"
	metricsServerFlag  = "metrics-server-addr"
	rpcServerFlag      = "rpc-server-addr"
	metisChainIDFlag   = "metis-chain-id"
	logsTypeFlag       = "logs-type"
)

var (
	logger = helper.Logger.With("module", "bridge/cmd/")

	metricsServer http.Server
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "themis-bridge",
	Aliases: []string{"bridge"},
	Short:   "Themis bridge deamon",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Use != version.Cmd.Use {
			// initialize tendermint viper config
			initTendermintViperConfig(cmd)

			// init metrics server
			initMetrics(cmd)
		}
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		return metricsServer.Shutdown(ctx)
	},
}

// BridgeCommands returns command for bridge service
func BridgeCommands(v *viper.Viper, loggerInstance tendermintLogger.Logger, caller string) *cobra.Command {
	DecorateWithBridgeRootFlags(rootCmd, v, loggerInstance, caller)
	return rootCmd
}

// DecorateWithBridgeRootFlags is called when bridge flags needs to be added to command
func DecorateWithBridgeRootFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance tendermintLogger.Logger, caller string) {
	cmd.PersistentFlags().StringP(helper.TendermintNodeFlag, "n", helper.DefaultTendermintNode, "Node to connect to")

	if err := v.BindPFlag(helper.TendermintNodeFlag, cmd.PersistentFlags().Lookup(helper.TendermintNodeFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, helper.TendermintNodeFlag), "Error", err)
	}

	cmd.PersistentFlags().String(helper.HomeFlag, helper.DefaultNodeHome, "directory for config and data")

	if err := v.BindPFlag(helper.HomeFlag, cmd.PersistentFlags().Lookup(helper.HomeFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, helper.HomeFlag), "Error", err)
	}

	// bridge storage db
	cmd.PersistentFlags().String(
		bridgeDBFlag,
		"",
		"Bridge db path (default <home>/bridge/storage)",
	)

	if err := v.BindPFlag(bridgeDBFlag, cmd.PersistentFlags().Lookup(bridgeDBFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, bridgeDBFlag), "Error", err)
	}

	// bridge chain id
	cmd.PersistentFlags().String(
		metisChainIDFlag,
		helper.DefaultMetisChainID,
		"Metis chain id",
	)

	// bridge logging type
	cmd.PersistentFlags().String(
		logsTypeFlag,
		helper.DefaultLogsType,
		"Use json logger",
	)

	// bridge metrics server listen addr
	cmd.PersistentFlags().String(
		metricsServerFlag,
		helper.DefaultMetricsListenAddr,
		"Metrics server listen addr, default to :2112",
	)

	// bridge rpc server listen addr
	cmd.PersistentFlags().String(
		rpcServerFlag,
		helper.DefaultRPCListenAddr,
		"RPC server listen addr, default to :8646",
	)

	if err := v.BindPFlag(metisChainIDFlag, cmd.PersistentFlags().Lookup(metisChainIDFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, metisChainIDFlag), "Error", err)
	}
}

// initMetrics initializes metrics server with the default handler
func initMetrics(cmd *cobra.Command) {
	cfg := rpcserver.DefaultConfig()
	metricsServerListenAddr := cmd.Flag(metricsServerFlag).Value.String()

	metricsServer = http.Server{
		Addr:              metricsServerListenAddr,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			logger.Error("failed to start metrics server", "error", err)
			os.Exit(1)
		}
	}()
}

// function is called to set appropriate bridge db path
func AdjustBridgeDBValue(cmd *cobra.Command, v *viper.Viper) {
	tendermintNode, _ := cmd.Flags().GetString(helper.TendermintNodeFlag)
	homeValue, _ := cmd.Flags().GetString(helper.HomeFlag)
	withThemisConfigValue, _ := cmd.Flags().GetString(helper.WithThemisConfigFlag)
	bridgeDBValue, _ := cmd.Flags().GetString(bridgeDBFlag)
	bridgeSqliteDBValue, _ := cmd.Flags().GetString(bridgeSqliteDBFlag)
	metisChainIDValue, _ := cmd.Flags().GetString(metisChainIDFlag)
	logsTypeValue, _ := cmd.Flags().GetString(logsTypeFlag)

	// bridge-db directory (default storage)
	if bridgeDBValue == "" {
		bridgeDBValue = filepath.Join(homeValue, "bridge", "storage")
	}

	if bridgeSqliteDBValue == "" {
		bridgeSqliteDBValue = filepath.Join(homeValue, "bridge", "sqlite")
	}

	// set to viper
	viper.Set(helper.TendermintNodeFlag, tendermintNode)
	viper.Set(helper.HomeFlag, homeValue)
	viper.Set(helper.WithThemisConfigFlag, withThemisConfigValue)
	viper.Set(bridgeDBFlag, bridgeDBValue)
	viper.Set(bridgeSqliteDBFlag, bridgeSqliteDBValue)
	viper.Set(metisChainIDFlag, metisChainIDValue)
	viper.Set(logsTypeFlag, logsTypeValue)
}

// initTendermintViperConfig sets global viper configuration needed to themis
func initTendermintViperConfig(cmd *cobra.Command) {
	// set appropriate bridge DB
	AdjustBridgeDBValue(cmd, viper.GetViper())

	// start themis config
	helper.InitThemisConfig("")
}
