package main

import (
	"context"
	"io"
	"os/exec"
	"strings"

	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
)

type serverAddress struct {
	Address string
	HasAddr chan struct{}
}

func (f *serverAddress) Write(b []byte) (int, error) {
	if f.Address != "" {
		return len(b), nil
	}
	s := string(b)
	if strings.HasPrefix(s, "Listening on") {
		f.Address = strings.TrimSpace(strings.TrimPrefix(s, "Listening on "))
		zap.S().Infof("Found language host listening on %s", f.Address)
		close(f.HasAddr)
	}
	return len(b), nil
}

func startPythonClient() (*exec.Cmd, *serverAddress) {
	cmd := logging.Command(
		context.TODO(),
		logging.CommandLogger{RootLogger: zap.L().Named("python")},
		"pipenv", "run", "python", "python_language_host.py",
	)

	lf := &serverAddress{
		HasAddr: make(chan struct{}),
	}
	cmd.Stdout = io.MultiWriter(cmd.Stdout, lf)
	cmd.Dir = "pkg/k2/language_host/python"
	setProcAttr(cmd)

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
