package construct

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ResourceId struct {
	Provider string `yaml:"provider" toml:"provider"`
	Type     string `yaml:"type" toml:"type"`
	// Namespace is optional and is used to disambiguate resources that might have
	// the same name. It can also be used to associate an imported resource with
	// a specific namespace such as a subnet to a VPC.
	Namespace string `yaml:"namespace" toml:"namespace"`
	Name      string `yaml:"name" toml:"name"`
}

func (id ResourceId) IsZero() bool {
	return id == ResourceId{}
}

func (id ResourceId) String() string {
	s := id.Provider + ":" + id.Type
	if id.Namespace != "" || strings.Contains(id.Name, ":") {
		s += ":" + id.Namespace
	}
	return s + ":" + id.Name
}

func (id ResourceId) QualifiedTypeName() string {
	return id.Provider + ":" + id.Type
}

func (id ResourceId) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

var (
	resourceProviderPattern  = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	resourceTypePattern      = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	resourceNamespacePattern = regexp.MustCompile(`^[a-zA-Z0-9_#./\-:\[\]]*$`)
	resourceNamePattern      = regexp.MustCompile(`^[a-zA-Z0-9_#./\-:\[\]]*$`)
)

func (id *ResourceId) UnmarshalText(data []byte) error {
	parts := strings.Split(string(data), ":")
	if len(parts) < 3 {
		return fmt.Errorf("invalid number of parts (%d) in resource id '%s'", len(parts), string(data))
	}
	if len(parts) > 4 {
		parts = append(parts[:3], strings.Join(parts[3:], ":"))
	}
	id.Provider = parts[0]
	id.Type = parts[1]
	if len(parts) == 4 {
		id.Namespace = parts[2]
		id.Name = parts[3]
	} else {
		id.Name = parts[2]
	}
	if id.IsZero() {
		return nil
	}
	var err error
	if !resourceProviderPattern.MatchString(id.Provider) {
		err = errors.Join(err, fmt.Errorf("invalid provider '%s' (must match %s)", id.Provider, resourceProviderPattern))
	}
	if !resourceTypePattern.MatchString(id.Type) {
		err = errors.Join(err, fmt.Errorf("invalid type '%s' (must match %s)", id.Type, resourceTypePattern))
	}
	if id.Namespace != "" && !resourceNamespacePattern.MatchString(id.Namespace) {
		err = errors.Join(err, fmt.Errorf("invalid namespace '%s' (must match %s)", id.Namespace, resourceNamespacePattern))
	}
	if !resourceNamePattern.MatchString(id.Name) {
		err = errors.Join(err, fmt.Errorf("invalid name '%s' (must match %s)", id.Name, resourceNamePattern))
	}
	if err != nil {
		return fmt.Errorf("invalid resource id '%s': %w", string(data), err)
	}
	return nil
}

func (id ResourceId) MarshalTOML() ([]byte, error) {
	return id.MarshalText()
}

func (id *ResourceId) UnmarshalTOML(data []byte) error {
	return id.UnmarshalText(data)
}
