package model

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

// URN represents a Unique Resource Name in the Klotho ecosystem
type URN struct {
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
			return nil, fmt.Errorf("invalid type format")
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
