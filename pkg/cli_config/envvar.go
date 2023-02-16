package cli_config

import "os"

// EnvVar represents an environment variable, specified by its key name.
// wrapper around  os.Getenv. This string's value is the env var key. Use GetOr to get its value, or a
// default if the value isn't set.
type EnvVar string

// GetOr uses os.Getenv to get the env var specified by the target EnvVar. If that env var's value is unset or empty,
// it returns the defaultValue.
func (s EnvVar) GetOr(defaultValue string) string {
	value := os.Getenv(string(s))
	if value == "" {
		return defaultValue
	} else {
		return value
	}
}
