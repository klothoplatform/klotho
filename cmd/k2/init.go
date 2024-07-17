package main

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	clicommon "github.com/klothoplatform/klotho/pkg/cli_common"
	"github.com/klothoplatform/klotho/pkg/k2/initialize"
	"github.com/klothoplatform/klotho/pkg/tui/prompt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var initConfig struct {
	projectName     string
	appName         string
	environment     string
	outputDirectory string
	defaultRegion   string
	programFileName string
	interactive     bool
	nonInteractive  bool
	skipInstall     bool
}

var awsDefaultRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"af-south-1", "ap-east-1", "ap-south-1", "ap-northeast-1",
	"ap-northeast-2", "ap-northeast-3", "ap-southeast-1",
	"ap-southeast-2", "ca-central-1", "eu-central-1",
	"eu-west-1", "eu-west-2", "eu-west-3", "eu-south-1",
	"eu-north-1", "me-south-1", "sa-east-1",
}

func newInitCommand() *cobra.Command {
	var initCommand = &cobra.Command{
		Use:     "init",
		Short:   "Initialize a new Klotho application",
		PreRunE: prerunInit,
		RunE:    runInit,
	}
	flags := initCommand.Flags()
	flags.StringVarP(&initConfig.appName, "app", "a", "", "App name")
	flags.StringVarP(&initConfig.environment, "environment", "e", "", "Environment")
	flags.BoolVarP(&initConfig.interactive, "interactive", "i", false, "Interactive mode")
	flags.StringVarP(&initConfig.outputDirectory, "output", "o", "", "Output directory")
	flags.StringVarP(&initConfig.programFileName, "program", "p", "infra.py", "Program file name")
	flags.StringVarP(&initConfig.projectName, "project", "P", "default-project", "Project name")
	flags.StringVarP(&initConfig.defaultRegion, "default-region", "R", "", "AWS default region")
	flags.BoolVarP(&initConfig.nonInteractive, "non-interactive", "", false, "Non-interactive mode")
	flags.BoolVarP(&initConfig.skipInstall, "skip-install", "", false, "Skip installing dependencies")

	exitOnError(initCommand.MarkFlagRequired("app"))
	exitOnError(initCommand.MarkFlagRequired("program"))
	exitOnError(initCommand.MarkFlagRequired("project"))

	exitOnError(initCommand.MarkFlagDirname("output"))
	exitOnError(initCommand.MarkFlagFilename("program"))

	return initCommand
}

func prerunInit(cmd *cobra.Command, args []string) error {
	if initConfig.nonInteractive && initConfig.interactive {
		return fmt.Errorf("cannot specify both interactive and non-interactive flags")
	}

	if !term.IsTerminal(os.Stdout.Fd()) {
		if initConfig.interactive {
			return fmt.Errorf("interactive mode is only supported in a terminal environment")
		}
		return nil
	}

	if initConfig.nonInteractive {
		return nil
	}

	return promptInputs(cmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing Klotho application...")
	err := initialize.Application(
		initialize.ApplicationRequest{
			Context:         context.Background(),
			ProjectName:     initConfig.projectName,
			AppName:         initConfig.appName,
			OutputDirectory: initConfig.outputDirectory,
			DefaultRegion:   initConfig.defaultRegion,
			Runtime:         "python",
			ProgramFileName: initConfig.programFileName,
			Environment:     initConfig.environment,
			SkipInstall:     initConfig.skipInstall,
		})

	if err != nil {
		fmt.Println("Error initializing Klotho application:", err)
		return err
	}

	fmt.Println("Klotho application initialized successfully")
	return nil
}

func promptInputs(cmd *cobra.Command) error {
	helpers := map[string]prompt.Helper{
		"program": {
			SuggestionResolverFunc: func(input string) []string {
				var err error
				outputDir := initConfig.outputDirectory
				if outputDir == "" {
					outputDir, err = os.Getwd()
					if err != nil {
						outputDir = "."
					}
				}
				if err == nil {
					_, err = os.Stat(filepath.Join(outputDir, initConfig.programFileName))
				}
				if err == nil {
					return []string{fmt.Sprintf("%s.py", initConfig.appName)}
				}
				return []string{"infra.py"}
			},
			ValidateFunc: func(input string) error {
				outputDir := initConfig.outputDirectory
				if outputDir == "" {
					var err error
					outputDir, err = os.Getwd()
					if err != nil {
						return nil
					}
				}
				if _, err := os.Stat(filepath.Join(outputDir, input)); err == nil {
					return fmt.Errorf("file '%s' already exists", input)
				}
				return nil
			},
		},
		"environment": {
			SuggestionResolverFunc: func(input string) []string {
				return []string{"dev", "staging", "prod", "test", "qa", "default", "development", "production"}
			},
		},
		"default-region": {
			SuggestionResolverFunc: func(input string) []string {
				return awsDefaultRegions
			},
		},
	}

	interactiveFlags := []string{"project", "app", "program", "default-region", "environment"}
	var flagsToPrompt []string

	for _, flagName := range interactiveFlags {
		flag := cmd.Flags().Lookup(flagName)
		isRequired := false
		if required, found := flag.Annotations[cobra.BashCompOneRequiredFlag]; found && required[0] == "true" {
			isRequired = true
		}
		if initConfig.interactive || (isRequired && flag.Value.String() == "") {
			flagsToPrompt = append(flagsToPrompt, flagName)
			continue
		}

		// Set any flags that have a default value to changed if they are not empty strings to allow a flag to both be required and have a default value
		if flag.Value.String() != "" {
			flag.Changed = true
		}

	}

	if len(flagsToPrompt) == 0 {
		return nil
	}

	promptCreator := func(flagName string) prompt.FlagPromptModel {
		flag := cmd.Flags().Lookup(flagName)
		return prompt.CreatePromptModel(flag, helpers[flagName], clicommon.IsFlagRequired(flag))
	}

	firstPrompt := promptCreator(flagsToPrompt[0])

	model := prompt.MultiFlagPromptModel{
		Prompts:       []prompt.FlagPromptModel{firstPrompt},
		FlagNames:     flagsToPrompt,
		Cmd:           cmd,
		Helpers:       helpers,
		PromptCreator: promptCreator,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	finalMultiPromptModel := finalModel.(prompt.MultiFlagPromptModel)
	if finalMultiPromptModel.Quit {
		return fmt.Errorf("operation cancelled by user")
	}

	return nil
}

func exitOnError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
