package cli

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/spf13/cobra"
	"strings"
)

var engineCfg struct {
	provider string
}

func (km KlothoMain) addEngineCli(root *cobra.Command) error {
	engineGroup := &cobra.Group{
		ID:    "engine",
		Title: "engine",
	}
	listResourceTypesCmd := &cobra.Command{
		Use:     "ListResourceTypes",
		Short:   "List resource types available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    km.ListResourceTypes,
	}

	flags := listResourceTypesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")

	listAttributesCmd := &cobra.Command{
		Use:     "ListAttributes",
		Short:   "List attributes available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    km.ListAttributes,
	}

	flags = listAttributesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	return nil
}

func (km *KlothoMain) ListResourceTypes(cmd *cobra.Command, args []string) error {
	cfg := config.Application{Provider: engineCfg.provider}

	plugins := &PluginSetBuilder{
		Cfg: &cfg,
	}
	err := km.PluginSetup(plugins)

	if err != nil {
		return err
	}

	resourceTypes := plugins.Engine.ListResourcesByType()
	fmt.Println(strings.Join(resourceTypes, "\n"))
	return nil
}

func (km *KlothoMain) ListAttributes(cmd *cobra.Command, args []string) error {
	cfg := config.Application{Provider: engineCfg.provider}

	plugins := &PluginSetBuilder{
		Cfg: &cfg,
	}
	err := km.PluginSetup(plugins)

	if err != nil {
		return err
	}

	attributes := plugins.Engine.ListAttributes()
	fmt.Println(strings.Join(attributes, "\n"))
	return nil
}
