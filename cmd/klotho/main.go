package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/updater"

	"github.com/klothoplatform/klotho/pkg/cli"
	"github.com/klothoplatform/klotho/pkg/input"

	"github.com/fatih/color"
	"github.com/klothoplatform/klotho/pkg/analytics"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var root = &cobra.Command{
	Use:  "klotho [path to source]",
	RunE: run,
}

var klothoUpdater = updater.Updater{ServerURL: updater.DefaultServer, Stream: updater.DefaultStream}

var cfg struct {
	verbose       bool
	config        string
	outDir        string
	ast           bool
	caps          bool
	provider      string
	appName       string
	strict        bool
	disableLogo   bool
	internalDebug bool
	version       bool
	uploadSource  bool
	update        bool
	cfgFormat     string
	login         string
}

var hadWarnings = atomic.NewBool(false)

func init() {
	err := zap.RegisterEncoder("klotho-cli", func(zcfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return logging.NewConsoleEncoder(cfg.verbose, hadWarnings), nil
	})

	if err != nil {
		panic(err)
	}
}

const (
	defaultOutDir = "compiled"
)

func main() {
	flags := root.Flags()

	flags.BoolVarP(&cfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.StringVarP(&cfg.config, "config", "c", "", "Config file")
	flags.StringVarP(&cfg.outDir, "outDir", "o", defaultOutDir, "Output directory")
	flags.BoolVar(&cfg.ast, "ast", false, "Print the AST to a companion file")
	flags.BoolVar(&cfg.caps, "caps", false, "Print the capabilities to a companion file")
	flags.StringVarP(&cfg.cfgFormat, "cfg-format", "F", "yaml", "The format for the compiled config file (if --config is not specified). Supports: yaml, toml, json")
	flags.StringVar(&cfg.appName, "app", "", "Application name")
	flags.StringVarP(&cfg.provider, "provider", "p", "", fmt.Sprintf("Provider to compile to. Supported: %v", "aws"))
	flags.BoolVar(&cfg.strict, "strict", false, "Fail the compilation on warnings")
	flags.BoolVar(&cfg.disableLogo, "disable-logo", false, "Disable printing the Klotho logo")
	flags.BoolVar(&cfg.uploadSource, "upload-source", false, "Upload the compressed source folder for debugging")
	flags.BoolVar(&cfg.internalDebug, "internalDebug", false, "Enable debugging for compiler")
	flags.BoolVar(&cfg.version, "version", false, "Print the version")
	flags.BoolVar(&cfg.update, "update", false, "update the cli to the latest version")
	flags.StringVar(&cfg.login, "login", "", "Login to Klotho with email. For anonymous login, use 'local'")

	err := root.Execute()
	if err != nil {
		if cfg.internalDebug {
			zap.S().Errorf("%+v", err)
		} else if !root.SilenceErrors {
			zap.S().Errorf("%v", err)
		}
		os.Exit(1)
	}
	if hadWarnings.Load() && cfg.strict {
		os.Exit(1)
	}
}

func setupLogger(analyticsClient *analytics.Client) (*zap.Logger, error) {
	var zapCfg zap.Config
	if cfg.verbose {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Encoding = "klotho-cli"
	return zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		trackingCore := analyticsClient.NewFieldListener(zapcore.WarnLevel)
		return zapcore.NewTee(core, trackingCore)
	}))
}

func readConfig(args []string) (appCfg config.Application, err error) {
	if cfg.config != "" {
		appCfg, err = config.ReadConfig(cfg.config)
		if err != nil {
			return
		}
	} else {
		appCfg.Format = cfg.cfgFormat
	}
	// TODO debug logging for when config file is overwritten by CLI flags
	if cfg.appName != "" {
		appCfg.AppName = cfg.appName
	}
	if cfg.provider != "" {
		appCfg.Provider = cfg.provider
	}
	if len(args) > 0 {
		appCfg.Path = args[0]
	}
	if cfg.outDir != "" {
		if appCfg.OutDir == "" || cfg.outDir != defaultOutDir {
			appCfg.OutDir = cfg.outDir
		}
	}

	return
}

func run(cmd *cobra.Command, args []string) (err error) {
	// color.NoColor is set if we're not a terminal that
	// supports color
	if !color.NoColor && !cfg.disableLogo {
		color.New(color.FgHiGreen).Println(cli.Logo)
		fmt.Println()
	}

	// create config directory if necessary, must run
	// before calling analytics for first time
	if err := cli.CreateKlothoConfigPath(); err != nil {
		zap.S().Warnf("failed to create .klotho directory: %v", err)
	}

	// Set up user if login is specified
	if cfg.login != "" {
		if err := analytics.CreateUser(cfg.login); err != nil {
			return errors.Wrapf(err, "could not configure user '%s'", cfg.login)
		}
		return nil
	}

	// Set up analytics
	analyticsClient, err := analytics.NewClient(map[string]interface{}{
		"version": Version,
		"strict":  cfg.strict,
	})
	if err != nil {
		return errors.New(fmt.Sprintf("Issue retrieving user info: %s. \nYou may need to run: klotho --login <email>", err))
	}

	z, err := setupLogger(analyticsClient)
	if err != nil {
		return err
	}
	defer z.Sync() // nolint:errcheck
	zap.ReplaceGlobals(z)

	errHandler := cli.ErrorHandler{
		InternalDebug: cfg.internalDebug,
		Verbose:       cfg.verbose,
	}

	defer analyticsClient.PanicHandler(&err, errHandler)

	if cfg.version {
		zap.S().Infof("Version: %s-%s-%s", Version, updater.OS, updater.Arch)
		return nil
	}

	// if update is specified do the update in place
	if cfg.update {
		if err := klothoUpdater.Update(Version); err != nil {
			analyticsClient.Error("klotho failed to update")
			return err
		}
		analyticsClient.Info("klotho was updated successfully")
		return nil
	}

	// check daily for new updates and notify users if found
	needsUpdate, err := klothoUpdater.CheckUpdate(Version)
	if err != nil {
		analyticsClient.Error(fmt.Sprintf("klotho failed to check for updates: %v", err))
		zap.S().Warnf("failed to check for updates: %v", err)
	}
	if needsUpdate {
		analyticsClient.Info("klotho update is available")
		zap.L().Info("new update is available, please run klotho --update to get the latest version")
	}

	appCfg, err := readConfig(args)
	if err != nil {
		return errors.Wrapf(err, "could not read config '%s'", cfg.config)
	}

	if appCfg.Path == "" {
		return errors.New("'path' required")
	}

	if appCfg.AppName == "" {
		return errors.New("'app' required")
	} else if len(appCfg.AppName) > 25 {
		zap.S().With(logging.SilentAnalytics(fmt.Sprintf("'app' must be less than 20 characters in length. 'app' was %s", cfg.appName)))
		return fmt.Errorf("'app' must be less than 25 characters in length. 'app' was %s", cfg.appName)
	}
	match, err := regexp.MatchString(`^[\w-.:/]+$`, cfg.appName)
	if err != nil {
		return err
	} else if !match {
		zap.S().With(logging.SilentAnalytics(fmt.Sprintf("'app' can only contain alphanumeric, -, and _. 'app' was %s", cfg.appName)))
		return fmt.Errorf("'app' can only contain alphanumeric, -, _, ., :, and /. 'app' was %s", cfg.appName)
	}

	if appCfg.Provider == "" {
		return errors.New("'provider' required")
	}

	// Update analytics with app configs
	analyticsClient.AppendProperties(map[string]interface{}{
		"provider": appCfg.Provider,
		"app":      appCfg.AppName,
	})

	analyticsClient.Info("klotho pre-compile")

	input, err := input.ReadOSDir(appCfg, cfg.config)
	if err != nil {
		return errors.Wrapf(err, "could not read root path %s", appCfg.Path)
	}

	if cfg.ast {
		if err = cli.OutputAST(input, appCfg.OutDir); err != nil {
			return errors.Wrap(err, "could not output helpers")
		}
	}
	if cfg.caps {
		if err = cli.OutputCapabilities(input, appCfg.OutDir); err != nil {
			return errors.Wrap(err, "could not output helpers")
		}
	}

	plugins := &cli.PluginSetBuilder{
		Cfg: &appCfg,
	}

	err = plugins.AddAll()
	if err != nil {
		return err
	}

	compiler := core.Compiler{
		Plugins: plugins.Plugins(),
	}

	analyticsClient.Info("klotho compiling")

	result, err := compiler.Compile(input)
	if err != nil {
		errHandler.PrintErr(err)
		analyticsClient.Error("klotho compiling failed")

		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return err
	}

	if cfg.uploadSource {
		analyticsClient.UploadSource(input)
	}

	resourceCounts, err := cli.OutputResources(result, appCfg.OutDir)
	if err != nil {
		return err
	}

	cli.CloseTreeSitter(result)

	analyticsClient.AppendProperties(map[string]interface{}{"resources": resourceCounts})
	analyticsClient.Info("klotho compile complete")

	return nil
}
