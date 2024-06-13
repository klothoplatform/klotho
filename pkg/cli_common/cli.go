package clicommon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CommonConfig struct {
	verbose   bool
	jsonLog   bool
	logsDir   string
	profileTo string
}

func setupProfiling(commonCfg *CommonConfig) func() {
	if commonCfg.profileTo != "" {
		err := os.MkdirAll(filepath.Dir(commonCfg.profileTo), 0755)
		if err != nil {
			panic(fmt.Errorf("failed to create profile directory: %w", err))
		}
		profileF, err := os.OpenFile(commonCfg.profileTo, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			panic(fmt.Errorf("failed to open profile file: %w", err))
		}
		err = pprof.StartCPUProfile(profileF)
		if err != nil {
			panic(fmt.Errorf("failed to start profile: %w", err))
		}
		return func() {
			pprof.StopCPUProfile()
			profileF.Close()
		}
	}
	return func() {}
}

func SetupRoot(root *cobra.Command, commonCfg *CommonConfig) {
	flags := root.PersistentFlags()
	flags.BoolVarP(&commonCfg.verbose, "verbose", "v", false, "Enable verbose logging")
	flags.BoolVar(&commonCfg.jsonLog, "json-log", false, "Enable JSON logging")
	flags.StringVar(&commonCfg.logsDir, "logs-dir", "", "Directory to write logs to")
	flags.StringVar(&commonCfg.profileTo, "profiling", "", "Profile to file")

	profileClose := func() {}

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logOpts := logging.LogOpts{
			Verbose:         commonCfg.verbose,
			CategoryLogsDir: commonCfg.logsDir,
			DefaultLevels: map[string]zapcore.Level{
				"kb.load":       zap.WarnLevel,
				"engine.opeval": zap.WarnLevel,
				"dot":           zap.WarnLevel,
				"npm":           zap.WarnLevel,
			},
		}
		if commonCfg.jsonLog {
			logOpts.Encoding = "json"
		}
		zap.ReplaceGlobals(logOpts.NewLogger())

		profileClose = setupProfiling(commonCfg)
	}

	root.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		zap.L().Sync() //nolint:errcheck

		profileClose()
	}
}
