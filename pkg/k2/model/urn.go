package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

// URN represents a Unique Resource Name in the Klotho ecosystem
type (
	URN struct {
		AccountID        string `yaml:"accountId"`
		Project          string `yaml:"project"`
		Environment      string `yaml:"environment,omitempty"`
		Application      string `yaml:"application,omitempty"`
		Type             string `yaml:"type,omitempty"`
		Subtype          string `yaml:"subtype,omitempty"`
		ParentResourceID string `yaml:"parentResourceId,omitempty"`
		ResourceID       string `yaml:"resourceId,omitempty"`
		Output           string `yaml:"output,omitempty"`
	}

	UrnType string
)

const (
	AccountUrnType                UrnType = "account"
	ProjectUrnType                UrnType = "project"
	EnvironmentUrnType            UrnType = "environment"
	ApplicationEnvironmentUrnType UrnType = "application_environment"
	ResourceUrnType               UrnType = "resource"
	OutputUrnType                 UrnType = "output"
	TypeUrnType                   UrnType = "type"
)

// ParseURN parses a URN string into a URN struct
func ParseURN(urnString string) (*URN, error) {
	var urn URN
	if err := urn.UnmarshalText([]byte(urnString)); err != nil {
		return nil, err
	}
	return &urn, nil
}

// String returns the URN as a string
func (u URN) String() string {
	var sb strings.Builder
	sb.WriteString("urn:")
	sb.WriteString(u.AccountID)
	sb.WriteString(":")
	sb.WriteString(u.Project)
	sb.WriteString(":")
	sb.WriteString(u.Environment)
	sb.WriteString(":")
	sb.WriteString(u.Application)
	sb.WriteString(":")
	if u.Type != "" && u.Subtype != "" {
		sb.WriteString(u.Type)
		sb.WriteString("/")
		sb.WriteString(u.Subtype)
	}
	sb.WriteString(":")
	if u.ParentResourceID != "" && u.ResourceID != "" {
		sb.WriteString(u.ParentResourceID)
		sb.WriteString("/")
		sb.WriteString(u.ResourceID)
	} else {
		sb.WriteString(u.ResourceID)
	}
	sb.WriteString(":")
	sb.WriteString(u.Output)
	sb.WriteString(":")

	// Remove trailing colons
	urn := sb.String()
	return strings.TrimRight(urn, ":")
}

func (u URN) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URN) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ":")

	if parts[0] == "urn" {
		parts = parts[1:]
	}

	if len(parts) < 2 {
		return fmt.Errorf("invalid URN format: missing account ID and/or project")
	} else if len(parts) > 7 {
		return fmt.Errorf("invalid URN format: too many parts")
	}

	u.AccountID = parts[0]
	u.Project = parts[1]

	if len(parts) > 2 {
		u.Environment = parts[2]
	}
	if len(parts) > 3 {
		u.Application = parts[3]
	}
	if len(parts) > 4 && parts[4] != "" {
		typeParts := strings.Split(parts[4], "/")
		if len(typeParts) != 2 {
			return fmt.Errorf("invalid URN type format: %s", parts[4])
		}
		u.Type = typeParts[0]
		u.Subtype = typeParts[1]
	}

	if len(parts) > 5 && parts[5] != "" {
		resourceParts := strings.Split(parts[5], "/")
		if len(resourceParts) == 2 {
			u.ParentResourceID = resourceParts[0]
			u.ResourceID = resourceParts[1]
		} else {
			u.ResourceID = parts[5]
		}
	}
	if len(parts) > 6 && parts[6] != "" {
		u.Output = parts[6]
	}

	return nil
}

func (u *URN) Equals(other any) bool {
	switch other := other.(type) {
	case URN:
		if u == nil {
			return false
		}
		return *u == other

	case *URN:
		if u == nil || other == nil {
			return u == other
		}
		return *u == *other
	}

	return false
}

func (u *URN) IsOutput() bool {
	// all fields are filled except application
	return u.AccountID != "" && u.Project != "" && u.Environment != "" && u.Type != "" &&
		u.Subtype != "" && u.ParentResourceID != "" && u.ResourceID != "" && u.Output != ""
}

func (u *URN) IsResource() bool {
	// all fields are filled except application and output
	return u.AccountID != "" && u.Project != "" && u.Environment != "" && u.Type != "" &&
		u.Subtype != "" && u.ResourceID != "" && u.Output == ""

}

func (u *URN) IsApplicationEnvironment() bool {
	return u.AccountID != "" && u.Project != "" && u.Environment != "" && u.Application != "" &&
		u.Type == "" && u.Subtype == "" && u.ParentResourceID == "" && u.ResourceID == "" && u.Output == ""
}

func (u *URN) IsType() bool {
	return u.Type != "" && u.Subtype == "" && u.ParentResourceID == "" && u.ResourceID == "" && u.Output == ""
}

func (u *URN) IsEnvironment() bool {
	return u.AccountID != "" && u.Project != "" && u.Environment != "" && u.Application == "" &&
		u.Type == "" && u.Subtype == "" && u.ParentResourceID == "" && u.ResourceID == "" && u.Output == ""
}

func (u *URN) IsProject() bool {
	return u.AccountID != "" && u.Project != "" && u.Environment == "" && u.Application == "" &&
		u.Type == "" && u.Subtype == "" && u.ParentResourceID == "" && u.ResourceID == "" && u.Output == ""
}

func (u *URN) IsAccount() bool {
	return u.AccountID != "" && u.Project == "" && u.Environment == "" && u.Application == "" &&
		u.Type == "" && u.Subtype == "" && u.ParentResourceID == "" && u.ResourceID == "" && u.Output == ""
}

func (u *URN) UrnType() UrnType {
	if u.IsAccount() {
		return AccountUrnType
	}
	if u.IsProject() {
		return ProjectUrnType
	}
	if u.IsEnvironment() {
		return EnvironmentUrnType
	}
	if u.IsApplicationEnvironment() {
		return ApplicationEnvironmentUrnType
	}
	if u.IsResource() {
		return ResourceUrnType
	}
	if u.IsOutput() {
		return OutputUrnType
	}
	if u.IsType() {
		return TypeUrnType
	}
	return ""
}

// UrnPath returns the relative filesystem path of the output for a given URN
// (e.g., project/application/environment/construct)
func UrnPath(urn URN) (string, error) {
	parts := []string{
		urn.Project,
		urn.Application,
		urn.Environment,
		urn.ResourceID,
	}

	for i, p := range parts {
		if p == "" {
			return filepath.Join(parts[:i]...), nil
		}
	}
	return filepath.Join(parts...), nil
}

func (u *URN) Compare(other URN) int {
	if u.AccountID != other.AccountID {
		return strings.Compare(u.AccountID, other.AccountID)
	}
	if u.Project != other.Project {
		return strings.Compare(u.Project, other.Project)
	}
	if u.Environment != other.Environment {
		return strings.Compare(u.Environment, other.Environment)
	}
	if u.Application != other.Application {
		return strings.Compare(u.Application, other.Application)
	}
	if u.Type != other.Type {
		return strings.Compare(u.Type, other.Type)
	}
	if u.Subtype != other.Subtype {
		return strings.Compare(u.Subtype, other.Subtype)
	}
	if u.ParentResourceID != other.ParentResourceID {
		return strings.Compare(u.ParentResourceID, other.ParentResourceID)
	}
	if u.ResourceID != other.ResourceID {
		return strings.Compare(u.ResourceID, other.ResourceID)
	}
	if u.Output != other.Output {
		return strings.Compare(u.Output, other.Output)
	}
	return 0
}
