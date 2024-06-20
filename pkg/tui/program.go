package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	UpdateMessage struct {
		Construct string
		Status    string

		Percent       float64
		Indeterminate bool
		Complete      bool
	}

	LogMessage struct {
		Construct string
		Message   string
	}

	TuiProgress struct {
		Prog      *tea.Program
		Construct string
	}
)

var programKey contextKey = "tui-prog"

func WithProgram(ctx context.Context, p *tea.Program) context.Context {
	return context.WithValue(ctx, programKey, p)
}

func GetProgram(ctx context.Context) *tea.Program {
	return ctx.Value(programKey).(*tea.Program)
}

func (p *TuiProgress) Update(status string, current, total int) {
	p.Prog.Send(UpdateMessage{
		Construct: p.Construct,
		Status:    status,
		Percent:   float64(current) / float64(total),
	})
}

func (p *TuiProgress) UpdateIndeterminate(status string) {
	p.Prog.Send(UpdateMessage{
		Construct:     p.Construct,
		Status:        status,
		Indeterminate: true,
	})
}

func (p *TuiProgress) Complete(status string) {
	p.Prog.Send(UpdateMessage{
		Construct: p.Construct,
		Status:    status,
		Complete:  true,
	})
}
