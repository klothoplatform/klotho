package tui

import (
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/go-wordwrap"
)

type (
	model struct {
		mu             sync.Mutex
		verbosity      int
		constructs     map[string]*constructModel
		constructOrder []string

		consoleWidth   int
		constructWidth int
		statusWidth    int
	}

	constructModel struct {
		logs *RingBuffer[string]

		status      string
		hasProgress bool
		complete    bool

		progress progress.Model
		spinner  spinner.Model
	}
)

func NewModel(verbosity int) *model {
	return &model{
		verbosity:  verbosity,
		constructs: make(map[string]*constructModel),
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) constructModel(construct string) *constructModel {
	cm, ok := m.constructs[construct]
	if !ok {
		cm = &constructModel{
			progress: progress.New(),
			spinner:  spinner.New(spinner.WithSpinner(spinner.Dot)),
		}
		switch m.verbosity {
		case 0:
		default:
			cm.logs = NewRingBuffer[string](10)
		}
		m.constructs[construct] = cm
		m.constructOrder = append(m.constructOrder, construct)
	}
	return cm
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.consoleWidth = msg.Width
		// TODO

	case LogMessage:
		if m.verbosity > 0 && msg.Message != "" {
			cm := m.constructModel(msg.Construct)
			cm.logs.Push(msg.Message)
		}

	case UpdateMessage:
		cm := m.constructModel(msg.Construct)

		m.constructWidth = 0
		m.statusWidth = 0
		for c, cm := range m.constructs {
			m.constructWidth = max(m.constructWidth, len(c))
			m.statusWidth = max(m.statusWidth, len(cm.status))
		}
		cm.hasProgress = !msg.Indeterminate
		cm.complete = msg.Complete

		if cm.hasProgress {
			cmd = cm.progress.SetPercent(msg.Percent)
		} else {
			cmd = cm.spinner.Tick
		}
		cm.status = msg.Status

	case progress.FrameMsg:
		cmds := make([]tea.Cmd, 0, len(m.constructs))
		for _, cm := range m.constructs {
			pm, cmd := cm.progress.Update(msg)
			cm.progress = pm.(progress.Model)
			cmds = append(cmds, cmd)
		}
		cmd = tea.Batch(cmds...)

	case spinner.TickMsg:
		cmds := make([]tea.Cmd, 0, len(m.constructs))
		for _, cm := range m.constructs {
			sm, cmd := cm.spinner.Update(msg)
			cm.spinner = sm
			cmds = append(cmds, cmd)
		}
		cmd = tea.Batch(cmds...)
	}

	return m, cmd
}

func (m *model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.verbosity {
	case 0:
		return m.viewCompact()
	case 1:
		return m.viewVerbose()
	default:
		return m.viewDebug()
	}
}

func (m *model) viewCompact() string {
	sb := new(strings.Builder)
	for _, c := range m.constructOrder {
		cm := m.constructs[c]

		if pad := m.constructWidth - len(c); pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(c)
		sb.WriteString(" ")

		if pad := m.statusWidth - len(cm.status); pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(cm.status)
		sb.WriteString(" ")

		switch {
		case cm.complete:
			// Do nothing

		case cm.hasProgress:
			sb.WriteString(cm.progress.View())

		default:
			sb.WriteString(cm.spinner.View())
		}

		sb.WriteRune('\n')
	}
	return sb.String()
}

func (m *model) viewVerbose() string {
	sb := new(strings.Builder)
	for _, c := range m.constructOrder {
		cm := m.constructs[c]

		if cm.complete {
			sb.WriteString("─")
			sb.WriteString(c)
			sb.WriteRune('\n')
			continue
		}
		sb.WriteString("┌ ")
		sb.WriteString(c)
		sb.WriteString(" ")
		sb.WriteString(cm.status)
		sb.WriteString("\n├ ")

		switch {
		case cm.hasProgress:
			sb.WriteString(cm.progress.View())

		default:
			sb.WriteString(cm.spinner.View())
		}
		sb.WriteRune('\n')

		cm.logs.ForEach(func(_ int, msg string) {
			msg = wordwrap.WrapString(msg, uint(m.consoleWidth-4))
			lines := strings.Split(msg, "\n")
			sb.WriteString("├ ")
			for i, l := range lines {
				if i > 0 {
					sb.WriteString("│ ")
				}
				sb.WriteString(l)
				sb.WriteRune('\n')
			}
		})
		sb.WriteString("└")
		sb.WriteRune('\n')
	}
	s := sb.String()
	return s
}

func (m *model) viewDebug() string {
	// for now, only difference is that the logs show in the top before the TUI,
	// which is handled outside of the model. Don't show logs twice, so use the
	// compact view.
	return m.viewCompact()
}
