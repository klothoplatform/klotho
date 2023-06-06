package cli_config

import (
	"os"
	"strings"
)

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

// GetBool returns the env var as a boolean.
//
// The value is false if the env var is:
//
//   - unset
//   - the empty string ("")
//   - "0" or "false" (case-insensitive)
//
// The value is true for all other values, including other false-looking strings like "no".
func (s EnvVar) GetBool() bool {
	switch strings.ToLower(os.Getenv(string(s))) {
	case "", "0", "false":
		return false
	default:
		return true
	}
}

func (s EnvVar) IsSet() bool {
	_, isSet := os.LookupEnv(string(s))
	return isSet
}
