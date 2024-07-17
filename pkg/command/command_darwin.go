package command

import (
	"os/exec"
	"syscall"
)

func SetProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
