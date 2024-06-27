package tui

import (
	"go.uber.org/zap"
)

type Progress interface {
	// Update updates the progress status with the current and total count.
	Update(status string, current, total int)
	UpdateIndeterminate(status string)
	Complete(status string)
}

type LogProgress struct {
	Logger *zap.Logger
}

func (p LogProgress) Update(status string, current, total int) {
	p.Logger.Sugar().Infof("%s %d/%d (%.1f%%)", status, current, total, float64(current)/float64(total)*100)
}

func (p LogProgress) UpdateIndeterminate(status string) {
	p.Logger.Info(status)
}

func (p LogProgress) Complete(status string) {
	p.Logger.Sugar().Debugf("Complete: %s", status)
}
