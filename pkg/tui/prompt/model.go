package prompt

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

type MultiFlagPromptModel struct {
	Prompts       []FlagPromptModel
	CurrentIndex  int
	Quit          bool
	Width         int
	Height        int
	FlagNames     []string
	Cmd           *cobra.Command
	Helpers       map[string]Helper
	PromptCreator func(string) FlagPromptModel
}

type FlagPromptModel struct {
	TextInput    textinput.Model
	Flag         *pflag.Flag
	InitialValue string
	Description  string
	IsRequired   bool
	FlagHelpers  Helper
	Completed    bool
}

func (m MultiFlagPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.CurrentIndex >= len(m.FlagNames) {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			// Go back to the previous prompt
			if m.CurrentIndex == 0 {
				return m, nil
			}
			m.Prompts[m.CurrentIndex].TextInput.Blur()
			m.CurrentIndex--
			m.Prompts[m.CurrentIndex].TextInput.Focus()
			return m, nil
		case tea.KeyDown, tea.KeyEnter:
			if msg.Type == tea.KeyDown && m.CurrentIndex == len(m.Prompts)-1 {
				// Down arrow on the last prompt should do nothing (down only submits if there's a next prompt already rendered)
				return m, nil
			}

			currentPrompt := &m.Prompts[m.CurrentIndex]
			value := currentPrompt.TextInput.Value()
			if currentPrompt.TextInput.Err != nil {
				return m, nil
			}
			if value == "" {
				if currentPrompt.InitialValue != "" {
					value = currentPrompt.InitialValue
				} else if currentPrompt.IsRequired {
					return m, nil
				}
			}

			if currentPrompt.FlagHelpers.ValidateFunc != nil {
				err := currentPrompt.FlagHelpers.ValidateFunc(value)
				if err != nil {
					currentPrompt.TextInput.Err = err
					return m, nil
				}
			}

			err := currentPrompt.Flag.Value.Set(value)
			if err != nil {
				return m, nil
			}
			currentPrompt.Flag.Changed = true
			currentPrompt.Completed = true
			currentPrompt.TextInput.Blur()
			m.CurrentIndex++

			if m.CurrentIndex >= len(m.FlagNames) {
				// If we've completed all prompts, quit immediately
				return m, tea.Quit
			}

			if m.CurrentIndex == len(m.Prompts) {
				newPrompt := m.PromptCreator(m.FlagNames[m.CurrentIndex])
				m.Prompts = append(m.Prompts, newPrompt)
			}
			m.Prompts[m.CurrentIndex].TextInput.Focus()
			return m, nil
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Quit = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		return m, tea.ClearScreen
	}

	var cmd tea.Cmd
	m.Prompts[m.CurrentIndex].TextInput, cmd = m.Prompts[m.CurrentIndex].TextInput.Update(msg)
	return m, cmd
}

func (m MultiFlagPromptModel) View() string {
	var b strings.Builder
	b.WriteString(tui.RenderLogo())
	b.WriteString("\n\n")

	b.WriteString("Please provide the following information to initialize your Klotho application:\n\n")

	for i, prompt := range m.Prompts {
		style := lipgloss.NewStyle()
		if i == m.CurrentIndex {
			style = style.Foreground(lipgloss.Color(tui.LogoColor))
		} else {
			style = style.Foreground(lipgloss.Color("230"))
		}

		initialValue := ""
		if prompt.InitialValue != "" {
			initialValueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("36"))
			styledInitialValue := initialValueStyle.Render(prompt.InitialValue)
			initialValue = fmt.Sprintf(" [%s]", styledInitialValue)
		}

		styledError := ""
		if prompt.TextInput.Err != nil {
			styledError = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf(" (%s)", prompt.TextInput.Err))
		}

		var promptView string
		if prompt.Completed {
			promptView = fmt.Sprintf("%s: %s", style.Render(prompt.Flag.Name), prompt.TextInput.View())
		} else {
			promptView = fmt.Sprintf("%s%s %s", style.Render(prompt.Flag.Name), initialValue, prompt.TextInput.View())
		}
		if styledError != "" {
			promptView += styledError
		}

		b.WriteString(promptView + "\n")
	}

	for i := 0; i < m.Height-len(m.Prompts)-3; i++ {
		b.WriteString("\n")
	}

	// Add footer
	b.WriteString("\nPress Esc or Ctrl+C to quit")

	return b.String()
}

type Helper struct {
	SuggestionResolverFunc func(string) []string
	ValidateFunc           func(string) error
}

func CreatePromptModel(flag *pflag.Flag, flagHelpers Helper, isRequired bool) FlagPromptModel {
	ti := textinput.New()
	description := flag.Usage
	if isRequired {
		style := lipgloss.NewStyle().Bold(true)
		requiredSuffix := style.Render(" (required)")
		description += requiredSuffix
	}
	ti.Placeholder = description
	ti.Validate = flagHelpers.ValidateFunc
	ti.CharLimit = 156
	ti.SetValue(flag.Value.String())
	if flagHelpers.SuggestionResolverFunc != nil {
		ti.SetSuggestions(flagHelpers.SuggestionResolverFunc(flag.Value.String()))
	}
	ti.ShowSuggestions = true
	ti.Focus()

	return FlagPromptModel{
		TextInput:    ti,
		Flag:         flag,
		InitialValue: flag.Value.String(),
		Description:  description,
		IsRequired:   isRequired,
		FlagHelpers:  flagHelpers,
		Completed:    false,
	}
}

func (m MultiFlagPromptModel) Init() tea.Cmd {
	return textinput.Blink
}
