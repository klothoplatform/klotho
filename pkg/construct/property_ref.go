package construct

import (
	"fmt"
	"strings"
)

type PropertyRef struct {
	Resource ResourceId
	Property string
}

func (v PropertyRef) String() string {
	if v.IsZero() {
		return ""
	}
	return v.Resource.String() + "#" + v.Property
}

func (v PropertyRef) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *PropertyRef) Parse(s string) error {
	res, prop, ok := strings.Cut(s, "#")
	if !ok {
		return fmt.Errorf("invalid PropertyRef format: %s", s)
	}
	v.Property = prop
	return v.Resource.Parse(res)
}

func (v *PropertyRef) Validate() error {
	return v.Resource.Validate()
}

func (v *PropertyRef) UnmarshalText(b []byte) error {
	if err := v.Parse(string(b)); err != nil {
		return err
	}
	return v.Validate()
}

func (v *PropertyRef) Equals(ref interface{}) bool {
	other, ok := ref.(PropertyRef)
	if !ok {
		return false
	}
	return v.Resource == other.Resource && v.Property == other.Property
}

func (v *PropertyRef) IsZero() bool {
	return v.Resource.IsZero() && v.Property == ""
}
