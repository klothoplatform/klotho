package cli

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/klothoplatform/klotho/pkg/cli_config"
	"github.com/klothoplatform/klotho/pkg/yaml_util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const configFileName = "options.yaml"

type (
	Options struct {
		Update UpdateOptions `yaml:",omitempty"`
	}

	UpdateOptions struct {
		Stream string `yaml:",omitempty"`
	}
)

func ReadOptions() (Options, error) {
	var options Options
	_, fileContents, err := readOptionsFileBytes()
	if err != nil {
		return options, err
	}
	err = yaml.Unmarshal(fileContents, &options)
	return options, err
}

func SetOptions(options map[string]string) error {
	if len(options) == 0 {
		// This isn't just an optimization. If the existing file is invalid, then we don't want an error message coming
		// from this path (since the user hasn't specified options to write, and would be confused by a message saying
		// "couldn't write CLI options" or similar)
		return nil
	}
	filePath, optionsYaml, err := readOptionsFileBytes()
	if err != nil {
		return err
	}
	optionsYaml, err = setOptions(optionsYaml, options)
	if err != nil {
		return nil
	}

	err = os.WriteFile(filePath, optionsYaml, 0600)
	if err != nil {
		err = errors.Wrap(err, "couldn't write CLI options file")
	}
	return err
}

// setOptions inserts the given options into the yaml, validating along the way that the options still form a valid
// Options.
func setOptions(optionsYaml []byte, options map[string]string) ([]byte, error) {
	logger := zap.S()
	// First, a warning if the original file isn't valid
	if err := yaml_util.CheckValid[Options](optionsYaml, yaml_util.Lenient); err != nil {
		return nil, errors.Wrap(err, "existing options file is invalid")
	} else if warns := yaml_util.CheckValid[Options](optionsYaml, yaml_util.Strict); warns != nil {
		logger.Warn(`Existing options contain extra parameters:`)
		for _, e := range yaml_util.YamlErrors(warns) {
			logger.Warnf(`▸ %s`, e)
		}
	}

	// Validate of each option entry, in two passes: first a lenient check that errors out if it finds any issues, and
	// then a strict check that warns on issues (and never errors out). We do the two passes so that you won't get
	// warnings and then an error.
	var warns []string
	for k, v := range options {
		yamlFragment, err := yaml_util.SetValue(nil, k, v)
		if err != nil {
			return nil, err
		}
		if err = yaml_util.CheckValid[Options](yamlFragment, yaml_util.Lenient); err != nil {
			return nil, err
		}
		if warn := yaml_util.CheckValid[Options](yamlFragment, yaml_util.Strict); warn != nil {
			warns = append(warns, fmt.Sprintf(`Unrecognized option "%s". We'll still set it, but it may not have any effect.`, k))
		}
	}
	for _, msg := range warns {
		logger.Warn(msg)
	}

	// Now add each value into the map. This isn't super efficient (it re-parses at each iteration), but it's not a
	// critical path.
	for k, v := range options {
		modified, err := yaml_util.SetValue(optionsYaml, k, v)
		if err != nil {
			return nil, errors.Wrapf(err, `invalid option: %s`, k)
		}
		optionsYaml = modified
	}
	// final sanity check to make sure we're setting valid json
	if err := yaml_util.CheckValid[Options](optionsYaml, yaml_util.Lenient); err != nil {
		return nil, errors.Wrapf(err, `couldn't write options (unknown error)`)
	}
	return optionsYaml, nil
}

func readOptionsFileBytes() (string, []byte, error) {
	path, err := cli_config.KlothoConfigPath(configFileName)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't find CLI options file path")
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil, nil // If the file isn't there, silently return the defaults
	} else if err != nil {
		return "", nil, errors.Wrap(err, "couldn't read CLI options file path")
	}
	return path, content, err
}

func OptionOrDefault(given string, defaultValue string) string {
	if given == "" {
		return defaultValue
	}
	return given
}

func ShouldCheckForUpdate(updateStreamOverride string, defaultUpdateStream string, currVersion string) bool {
	if updateStreamOverride == "" || updateStreamOverride == defaultUpdateStream {
		return true
	}

	streamOverrideParts := strings.Split(updateStreamOverride, ":")
	if len(streamOverrideParts) == 2 {
		streamOverrideVersion := streamOverrideParts[1]
		if streamOverrideVersion != currVersion {
			return true
		}
	}

	return false
}
