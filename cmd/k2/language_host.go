package main

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/klothoplatform/klotho/pkg/k2/cleanup"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/logging"
)

type serverAddress struct {
	Log     *zap.SugaredLogger
	Address string
	HasAddr chan struct{}
}

var listenOnPattern = regexp.MustCompile(`(?m)^\s*Listening on (\S+)$`)

func (f *serverAddress) Write(b []byte) (int, error) {
	if f.Address != "" {
		return len(b), nil
	}
	s := string(b)
	matches := listenOnPattern.FindStringSubmatch(s)
	if len(matches) >= 2 {
		// address is the first match
		f.Address = matches[1]
		f.Log.Infof("Found language host listening on %s", f.Address)
		close(f.HasAddr)
	}

	return len(b), nil
}

type DebugConfig struct {
	Enabled bool
	Port    int
	Mode    string
}

func startPythonClient(ctx context.Context, debugConfig DebugConfig) (*exec.Cmd, *serverAddress) {
	log := logging.GetLogger(ctx).Sugar()

	args := []string{"run", "python", "python_language_host.py"}
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
		logging.CommandLogger{RootLogger: log.Desugar().Named("python")},
		"pipenv", args...,
	)

	lf := &serverAddress{
		Log:     log,
		HasAddr: make(chan struct{}),
	}
	cmd.Stdout = io.MultiWriter(cmd.Stdout, lf)
	cmd.Dir = "pkg/k2/language_host/python"
	setProcAttr(cmd)
	cleanup.OnKill(func(signal syscall.Signal) error {
		cleanup.SignalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		return nil
	})

	log.Debugf("Executing: %s for %v", cmd.Path, cmd.Args)
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start Python client: %v", err)
	}
	log.Info("Python client started")

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("Python process exited with error: %v", err)
		} else {
			log.Debug("Python process exited successfully")
		}
	}()

	return cmd, lf
}
