package tui

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/go-wordwrap"
)

type (
	model struct {
		mu             sync.Mutex
		verbosity      Verbosity
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

const logoColor = "#816FA6"

var (
	//go:embed logo.txt
	logo string

	logoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(logoColor))
)

func NewModel(verbosity Verbosity) *model {
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
		cm.logs = NewRingBuffer[string](10)
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
			cmd = tea.Batch(
				cm.spinner.Tick,

				// Reset the progress to 0. This isn't guaranteed, but if we switched from a progress
				// to a spinner, most likely it's changing what's measured. By setting this to 0 now,
				// we try to prevent the progress from "bouncing back" from the last value it was at (likely at 100%)
				// due to the animation of the progress bar.
				cm.progress.SetPercent(0.0),
			)
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

	if m.verbosity == VerbosityConcise {
		return m.viewCompact()
	} else if m.verbosity.CombineLogs() {
		return m.viewDebug()
	} else {
		return m.viewVerbose()
	}
}

func (m *model) viewCompact() string {
	sb := new(strings.Builder)
	sb.WriteString(logoStyle.Render(logo))
	sb.WriteString("\n")

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
	sb.WriteString(logoStyle.Render(logo))
	sb.WriteString("\n")

	for _, c := range m.constructOrder {
		cm := m.constructs[c]

		if cm.complete {
			sb.WriteString("─ ")
			sb.WriteString(c)
			sb.WriteString(" ")
			sb.WriteString(cm.status)
			sb.WriteString("\n")
			continue
		}
		sb.WriteString("┌ ")
		if c != "" {
			sb.WriteString(c)
			sb.WriteString(" ")
			sb.WriteString(cm.status)
			if cm.logs.Len() > 0 {
				sb.WriteString("\n├ ")
			} else {
				sb.WriteString("\n└ ")
			}

			switch {
			case cm.hasProgress:
				sb.WriteString(cm.progress.View())

			default:
				sb.WriteString(cm.spinner.View())
			}
			sb.WriteRune('\n')
		}

		cm.logs.ForEach(func(idx int, msg string) {
			msg = wordwrap.WrapString(msg, uint(m.consoleWidth-4))
			lines := strings.Split(msg, "\n")

			// If this is the first log message and there's no construct, the line already has a prefix of "┌ "
			// so only write the prefix if it's not the case.
			if !(idx == 0 && c == "") {
				if idx == cm.logs.Len()-1 && len(lines) == 1 {
					sb.WriteString("└ ")
				} else {
					sb.WriteString("├ ")
				}
			}

			for i, l := range lines {
				if i > 0 {
					if idx == cm.logs.Len()-1 && i == len(lines)-1 {
						sb.WriteString("└ ")
					} else {
						sb.WriteString("│ ")
					}
				}
				sb.WriteString(l)
				sb.WriteRune('\n')
			}
		})
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
