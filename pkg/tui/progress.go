package tui

import (
	"go.uber.org/zap"
)

type Progress interface {
	Update(status string, current, total int)
}

type LogProgress struct {
	Logger *zap.Logger
}

func (p LogProgress) Update(status string, current, total int) {
	p.Logger.Sugar().Infof("%s %d/%d (%.1f%%)", status, current, total, float64(current)/float64(total)*100)
}
