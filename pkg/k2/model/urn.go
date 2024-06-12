package model

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"regexp"
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
	re := regexp.MustCompile(`[^:]*`)
	matches := re.FindAllString(urnString, -1)

	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid URN format")
	}

	if matches[0] == "urn" {
		matches = matches[1:]
	}

	urn := &URN{
		AccountID: matches[0],
		Project:   matches[1],
	}

	if len(matches) > 2 && matches[2] != "" {
		urn.Environment = matches[2]
	}
	if len(matches) > 3 && matches[3] != "" {
		urn.Application = matches[3]
	}
	if len(matches) > 4 && matches[4] != "" {
		typeParts := strings.Split(matches[4], "/")
		if len(typeParts) != 2 {
			return nil, fmt.Errorf("invalid URN type format")
		}
		urn.Type = typeParts[0]
		urn.Subtype = typeParts[1]
	}

	if len(matches) > 5 && matches[5] != "" {
		resourceParts := strings.Split(matches[5], "/")
		if len(resourceParts) == 2 {
			urn.ParentResourceID = resourceParts[0]
			urn.ResourceID = resourceParts[1]
		} else {
			urn.ResourceID = matches[5]
		}
	}
	if len(matches) > 6 && matches[6] != "" {
		urn.Output = matches[6]
	}

	if len(matches) > 7 {
		return nil, fmt.Errorf("invalid URN format")
	}

	return urn, nil
}

// String returns the URN as a string
func (u *URN) String() string {
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

func (u *URN) MarshalYAML() (interface{}, error) {
	return u.String(), nil
}

func (u *URN) UnmarshalYAML(value *yaml.Node) error {
	var urnString string
	if err := value.Decode(&urnString); err != nil {
		return err
	}

	parsedUrn, err := ParseURN(urnString)
	if err != nil {
		return err
	}

	*u = *parsedUrn
	return nil
}

func (u *URN) Equals(other *URN) bool {
	if u.AccountID != other.AccountID {
		return false
	}
	if u.Project != other.Project {
		return false
	}
	if u.Environment != other.Environment {
		return false
	}
	if u.Application != other.Application {
		return false
	}
	if u.Type != other.Type {
		return false
	}
	if u.Subtype != other.Subtype {
		return false
	}
	if u.ParentResourceID != other.ParentResourceID {
		return false
	}
	if u.ResourceID != other.ResourceID {
		return false
	}
	if u.Output != other.Output {
		return false
	}
	return true
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
