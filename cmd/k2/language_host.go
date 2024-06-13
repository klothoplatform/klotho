package main

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"syscall"

	"github.com/klothoplatform/klotho/pkg/k2/cleanup"

	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
)

type serverAddress struct {
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
		zap.S().Infof("Found language host listening on %s", f.Address)
		close(f.HasAddr)
	}

	return len(b), nil
}

type DebugConfig struct {
	Enabled bool
	Port    int
	Mode    string
}

func startPythonClient(debugConfig DebugConfig) (*exec.Cmd, *serverAddress) {
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
		context.TODO(),
		logging.CommandLogger{RootLogger: zap.L().Named("python")},
		"pipenv", args...,
	)

	lf := &serverAddress{
		HasAddr: make(chan struct{}),
	}
	cmd.Stdout = io.MultiWriter(cmd.Stdout, lf)
	cmd.Dir = "pkg/k2/language_host/python"
	setProcAttr(cmd)
	cleanup.OnKill(func(signal syscall.Signal) error {
		cleanup.SignalProcessGroup(cmd.Process.Pid, syscall.SIGTERM)
		return nil
	})

	zap.S().Debugf("Executing: %s for %v", cmd.Path, cmd.Args)
	if err := cmd.Start(); err != nil {
		zap.S().Fatalf("failed to start Python client: %v", err)
	}
	zap.L().Info("Python client started")

	go func() {
		err := cmd.Wait()
		if err != nil {
			zap.S().Errorf("Python process exited with error: %v", err)
		} else {
			zap.L().Debug("Python process exited successfully")
		}
	}()

	return cmd, lf
}
