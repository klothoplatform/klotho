package language_host

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/command"
	"io"
	"os"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/klothoplatform/klotho/pkg/k2/cleanup"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/logging"
)

//go:embed python/python_language_host.py
var pythonLanguageHost string

type ServerAddress struct {
	Log     *zap.SugaredLogger
	Address string
	HasAddr chan struct{}
}

var listenOnPattern = regexp.MustCompile(`(?m)^\s*Listening on (\S+)$`)

func (f *ServerAddress) Write(b []byte) (int, error) {
	if f.Address != "" {
		return len(b), nil
	}
	s := string(b)
	matches := listenOnPattern.FindStringSubmatch(s)
	if len(matches) >= 2 {
		// address is the first match
		f.Address = matches[1]
		f.Log.Debugf("Found language host listening on %s", f.Address)
		close(f.HasAddr)
	}

	return len(b), nil
}

type DebugConfig struct {
	Enabled bool
	Port    int
	Mode    string
}

func copyToTempDir(name, content string) (string, error) {
	f, err := os.CreateTemp("", fmt.Sprintf("k2_%s*.py", name))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	return f.Name(), nil

}

func StartPythonClient(ctx context.Context, debugConfig DebugConfig) (*exec.Cmd, *ServerAddress, error) {
	log := logging.GetLogger(ctx).Sugar()
	hostPath, err := copyToTempDir("python_language_host", pythonLanguageHost)
	if err != nil {
		return nil, nil, fmt.Errorf("could not copy python language host to temp dir: %w", err)
	}

	args := []string{"run", "python", hostPath}
	if debugConfig.Enabled {
		if debugConfig.Port > 0 {
			args = append(args, "--debug-port", fmt.Sprintf("%d", debugConfig.Port))
		}
		if debugConfig.Mode != "" {
			args = append(args, "--debug", debugConfig.Mode)
		}
	}

	cmd := logging.Command(
		ctx,
		logging.CommandLogger{
			RootLogger:  log.Desugar().Named("python"),
			StdoutLevel: zap.DebugLevel,
			StderrLevel: zap.DebugLevel,
		},
		"pipenv", args...,
	)

	lf := &ServerAddress{
		Log:     log,
		HasAddr: make(chan struct{}),
	}
	cmd.Stdout = io.MultiWriter(cmd.Stdout, lf)
	command.SetProcAttr(cmd)

	cleanup.OnKill(func(signal syscall.Signal) error {
		cleanup.SignalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		return nil
	})

	log.Debugf("Executing: %s for %v", cmd.Path, cmd.Args)
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start Python client: %w", err)
	}
	log.Debug("Python client started")

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("Python process exited with error: %v", err)
		} else {
			log.Debug("Python process exited successfully")
		}
	}()

	return cmd, lf, nil
}
