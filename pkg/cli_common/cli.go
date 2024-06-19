package clicommon

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	CommonConfig struct {
		jsonLog   bool
		verbose   verbosityFlag
		logsDir   string
		profileTo string
	}

	verbosityFlag int
)

func (f *verbosityFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		l, intErr := strconv.ParseInt(s, 10, 64)
		if intErr != nil {
			return err
		}
		*f = verbosityFlag(l)
	}
	if v {
		*f++
	} else if *f > 0 {
		*f--
	}
	return nil
}

func (f *verbosityFlag) Type() string {
	return "verbosity"
}

func (f *verbosityFlag) String() string {
	return strconv.FormatInt(int64(*f), 10)
}

func (*verbosityFlag) IsBoolFlag() bool { return true }

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

func SetupRoot(root *cobra.Command, commonCfg *CommonConfig) func() {

	flags := root.PersistentFlags()

	verbosity := flags.VarPF(&commonCfg.verbose, "verbose", "v", "Enable verbose logging")
	verbosity.NoOptDefVal = "true" // Allow -v to be used without a value

	flags.BoolVar(&commonCfg.jsonLog, "json-log", false, "Enable JSON logging")
	flags.StringVar(&commonCfg.logsDir, "logs-dir", "", "Directory to write logs to")
	flags.StringVar(&commonCfg.profileTo, "profiling", "", "Profile to file")

	profileClose := func() {}
	tuiClose := func() {}

	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true // Silence usage after args have been parsed

		logOpts := logging.LogOpts{
			Verbose:         commonCfg.verbose > 0,
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
		prog := tea.NewProgram(
			tui.NewModel(int(commonCfg.verbose)),
			tea.WithoutSignalHandler(),
			tea.WithContext(root.Context()),
		)
		log := zap.New(tui.NewLogCore(logOpts, int(commonCfg.verbose), prog))
		zap.ReplaceGlobals(log)
		go func() {
			_, err := prog.Run()
			if err != nil {
				zap.S().With(zap.Error(err)).Error("TUI exited with error")
			} else {
				zap.S().Debug("TUI exited")
			}
		}()
		go func() {
			<-cmd.Context().Done()
			zap.S().Debug("Shutting down TUI due to context closure")
			prog.Quit()
		}()
		zap.S().Debug("Starting TUI")
		cmd.SetContext(tui.WithProgram(cmd.Context(), prog))

		profileClose = setupProfiling(commonCfg)
		tuiClose = func() {
			zap.L().Debug("Shutting down TUI")
			prog.Quit()
			prog.Wait()
		}
	}

	return func() {
		tuiClose()
		profileClose()
		_ = zap.L().Sync()
	}
}
