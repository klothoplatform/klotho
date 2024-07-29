package tui

import (
	"fmt"
	"io"
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

		outputs     map[string]any
		outputOrder []string

		status      string
		hasProgress bool
		complete    bool

		progress progress.Model
		spinner  spinner.Model
	}
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
			outputs:  make(map[string]any),
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

	case OutputMessage:
		cm := m.constructModel(msg.Construct)
		if _, ok := cm.outputs[msg.Name]; !ok {
			cm.outputOrder = append(cm.outputOrder, msg.Name)
		}
		cm.outputs[msg.Name] = msg.Value

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
	sb.WriteString(RenderLogo())
	sb.WriteString("\n")

	for _, c := range m.constructOrder {
		cm := m.constructs[c]

		if pad := m.constructWidth - len(c); pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
		sb.WriteString(boxTitleStyle.Render(c))
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

		outputPad := strings.Repeat(" ", m.constructWidth-1)
		for i, name := range cm.outputOrder {
			out := cm.outputs[name]
			sb.WriteString(outputPad)
			if i == len(cm.outputs)-1 {
				fmt.Fprintf(sb, "└ %s: %v\n", name, out)
			} else {
				fmt.Fprintf(sb, "├ %s: %v\n", name, out)
			}
		}
	}
	return sb.String()
}

var (
	boxTitleStyle          = lipgloss.NewStyle().Bold(true)
	boxSectionHeadingStyle = lipgloss.NewStyle().Underline(true)
)

type boxLine struct {
	Content string
	NoWrap  bool
}

// renderConstructBox renders a construct box to the given writer. There are 2 kinds of boxes:
// The general logs box:
//
//	┌ General
//	├ Logs
//	├  12.1s DBG > populated default value ...
//	├  ...
//	└  40.6s DBG > Shutting down TUI
//
// And a construct box:
//
//	┌ my-api Success (dry run)
//	├ Logs
//	├  29.3s DBG > AddEdge ...
//	├  ...
//	├ Outputs
//	└ └ Endpoint: <aws:api_stage:my-api-api:my-api-stage#StageInvokeUrl>
//
// Long lines are wrapped to fit the console width:
//
//	├  44.6s INF pulumi.preview >
//	│ error: Preview failed: resource 'preview(id=aws:subnet:default-network-vpc:default-network-public-subnet-1)' does
//	│ not exist
func (m *model) renderConstructBox(lines []boxLine, w io.Writer) {
	write := func(s string) {
		_, _ = w.Write([]byte(s))
	}
	for i, elem := range lines {
		msg := elem.Content
		if !elem.NoWrap {
			msg = wordwrap.WrapString(elem.Content, uint(m.consoleWidth-4))
		}
		elemLines := strings.Split(msg, "\n")
		for j, line := range elemLines {

			switch {
			case len(lines) == 1 && len(elemLines) == 1:
				// Single line special case
				write("─ ")

			case i == 0 && j == 0:
				// First line in the box
				write("┌ ")

			case i == len(lines)-1 && j == len(elemLines)-1:
				// Last line in the box
				write("└ ")

			case j == 0:
				// A list element
				write("├ ")

			default:
				// A continuation line
				write("│ ")
			}

			write(line + "\n")
		}
	}
}

func (m *model) viewVerbose() string {
	sb := new(strings.Builder)
	sb.WriteString(RenderLogo())
	sb.WriteString("\n")

	for _, c := range m.constructOrder {
		cm := m.constructs[c]

		var lines []boxLine
		addLine := func(s string) { // convenience function because most lines are regular content
			lines = append(lines, boxLine{Content: s})
		}

		if c == "" {
			addLine(boxTitleStyle.Render("General"))
		} else {
			addLine(boxTitleStyle.Render(c) + " " + cm.status)
			if !cm.complete {
				// Don't use addLine in the following to disable wrapping
				switch {
				case cm.hasProgress:
					lines = append(lines, boxLine{Content: cm.progress.View(), NoWrap: true})

				default:
					lines = append(lines, boxLine{Content: cm.spinner.View(), NoWrap: true})
				}
			}
		}
		cm.logs.ForEach(func(idx int, msg string) {
			if idx == 0 {
				// Only render the heading if there are actually logs to show
				// Check inside the ForEach instead of using Len to prevent a race condition
				addLine(boxSectionHeadingStyle.Render("Logs"))
			}
			addLine(msg)
		})
		for i, name := range cm.outputOrder {
			if i == 0 {
				addLine(boxSectionHeadingStyle.Render("Outputs"))
			}
			out := cm.outputs[name]
			if i == len(cm.outputs)-1 {
				addLine(fmt.Sprintf("└ %s: %v", name, out))
			} else {
				addLine(fmt.Sprintf("├ %s: %v", name, out))
			}
		}
		m.renderConstructBox(lines, sb)
		sb.WriteRune('\n') // extra newline to separate constructs
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
