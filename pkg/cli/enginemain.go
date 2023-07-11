package cli

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/spf13/cobra"
)

type EngineCLI struct {
	engine engine.Engine
}

var engineCfg struct {
	provider string
}

func addEngineCli(root *cobra.Command) error {

	engineCli := EngineCLI{}

	engineGroup := &cobra.Group{
		ID:    "engine",
		Title: "engine",
	}
	listResourceTypesCmd := &cobra.Command{
		Use:     "ListResourceTypes",
		Short:   "List resource types available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    engineCli.ListResourceTypes,
	}

	flags := listResourceTypesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")

	listAttributesCmd := &cobra.Command{
		Use:     "ListAttributes",
		Short:   "List attributes available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    engineCli.ListAttributes,
	}

	flags = listAttributesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	return nil
}

func (e *EngineCLI) ListResourceTypes(cmd *cobra.Command, args []string) error {
	cfg := config.Application{Provider: engineCfg.provider}
	pluginSetBuilder := PluginSetBuilder{Cfg: &cfg}
	err := pluginSetBuilder.AddEngine()
	if err != nil {
		return err
	}
	e.engine = *pluginSetBuilder.Engine
	fmt.Println(e.engine.ListResourcesByType())
	return nil
}

func (e *EngineCLI) ListAttributes(cmd *cobra.Command, args []string) error {
	cfg := config.Application{Provider: engineCfg.provider}
	pluginSetBuilder := PluginSetBuilder{Cfg: &cfg}
	err := pluginSetBuilder.AddEngine()
	if err != nil {
		return err
	}
	e.engine = *pluginSetBuilder.Engine
	fmt.Println(e.engine.ListAttributes())
	return nil
}
