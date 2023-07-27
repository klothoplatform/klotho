package cli

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/closenicely"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
)

type testCase struct {
	Source   core.Resource
	Target   core.Resource
	Document *compiler.CompilationDocument
	Length   int
	Name     string
}

var engineCfg struct {
	provider string
}

var getPathsConfig struct {
	resourceRoots   []string
	resourceTargets []string
	maxPathLength   int
	maxPaths        int
	providers       []string
	outDir          string
	verbose         bool
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

	getPaths := &cobra.Command{
		Use:     "GetPaths",
		Short:   "Outputs all paths up to a certain length from a given resource type",
		GroupID: engineGroup.ID,
		RunE:    km.GetPaths,
	}

	flags = getPaths.Flags()
	flags.IntVarP(&getPathsConfig.maxPathLength, "max-path-length", "l", 3, "maximum path length")
	flags.IntVarP(&getPathsConfig.maxPaths, "max-paths", "m", 1, "maximum number of paths to output")
	flags.StringSliceVarP(&getPathsConfig.providers, "providers", "p", []string{"klotho", "aws", "kubernetes", "docker"}, "the providers to use for target resources")
	flags.StringVarP(&getPathsConfig.outDir, "out-dir", "o", ".", "output directory")
	flags.StringSliceVarP(&getPathsConfig.resourceRoots, "resource-roots", "r", []string{}, "the resource roots to use for the paths")
	flags.StringSliceVarP(&getPathsConfig.resourceTargets, "resource-targets", "t", []string{}, "the resource targets to use for the paths")
	flags.BoolVarP(&getPathsConfig.verbose, "verbose", "v", false, "verbose output")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	root.AddCommand(getPaths)
	return nil
}

func (km *KlothoMain) GetPaths(cmd *cobra.Command, args []string) error {
	var zapCfg zap.Config
	if getPathsConfig.verbose {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Encoding = consoleEncoderName

	z, err := zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core)
	}))
	if err != nil {
		return err
	}
	defer closenicely.FuncOrDebug(z.Sync)
	zap.ReplaceGlobals(z)

	roots := getPathsConfig.resourceRoots
	cfg := config.Application{Provider: engineCfg.provider}

	plugins := &PluginSetBuilder{
		Cfg: &cfg,
	}
	err = km.PluginSetup(plugins)

	if err != nil {
		return err
	}

	var allTestCases []testCase
	for _, resourceType := range roots {
		resourceTypes := plugins.Engine.ListResources()
		startingResource, ok := filter.NewSimpleFilter[core.Resource](func(r core.Resource) bool {
			return fmt.Sprintf("%s:%s", r.Id().Provider, r.Id().Type) == resourceType
		}).Find(resourceTypes...)
		if !ok {
			return fmt.Errorf("could not find resource type %s", resourceType)
		}
		allowedTargets := filter.NewSimpleFilter[core.Resource](func(r core.Resource) bool {
			return collectionutil.Contains(getPathsConfig.providers, r.Id().Provider)
		}).Apply(resourceTypes...)

		var testCases []testCase

		for len(testCases) < getPathsConfig.maxPaths && len(allowedTargets) > 0 {
			i := rand.Intn(len(allowedTargets))
			targetResource := allowedTargets[i]
			allowedTargets = append(allowedTargets[:i], allowedTargets[i+1:]...)

			// limits the target resources to the ones specified by the user
			if len(getPathsConfig.resourceTargets) > 0 {
				if !collectionutil.Contains(getPathsConfig.resourceTargets, fmt.Sprintf("%s:%s", targetResource.Id().Provider, targetResource.Id().Type)) {
					continue
				}
			}

			// use reflection to copy startingResource and targetResource
			startingResource = reflect.New(reflect.TypeOf(startingResource).Elem()).Interface().(core.Resource)
			targetResource = reflect.New(reflect.TypeOf(targetResource).Elem()).Interface().(core.Resource)

			err = setField(startingResource, "Name", fmt.Sprintf("%s_%s", strings.ToLower(startingResource.Id().Type), "01"))
			if err != nil {
				zap.L().Sugar().Errorf("Failed to set Name on source: %s", err)
				continue
			}
			targetSuffix := "01"
			if startingResource.Id().Provider == targetResource.Id().Provider && startingResource.Id().Type == targetResource.Id().Type {
				targetSuffix = "02"
			}
			err = setField(targetResource, "Name", fmt.Sprintf("%s_%s", strings.ToLower(targetResource.Id().Type), targetSuffix))
			if err != nil {
				zap.L().Sugar().Errorf("Failed to set Name on target: %s", err)
				continue
			}
			testCase := newTestCase(startingResource, targetResource, plugins)
			if testCase != nil {
				testCases = append(testCases, *testCase)
			}
		}
		allTestCases = append(allTestCases, testCases...)
	}

	for _, tc := range allTestCases {
		zap.L().Sugar().Infof("Writing %s", tc.Name)
		outdir := filepath.Join(getPathsConfig.outDir, tc.Name)
		err = tc.Document.OutputTo(outdir)
		if err != nil {
			zap.L().Sugar().Errorf("Failed to output to %s: %s", outdir, err)
			continue
		}
	}

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

func setField(v interface{}, field string, value interface{}) error {

	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("value must be a pointer")
	}

	dv := reflect.ValueOf(v).Elem()

	if dv.Kind() != reflect.Struct {
		return fmt.Errorf("value must be a pointer to a struct/interface")
	}

	f := dv.FieldByName(field)

	if !f.CanSet() {
		return fmt.Errorf("value has no field %q or cannot be set", field)
	}

	nv := reflect.ValueOf(value)

	f.Set(nv)

	return nil
}
