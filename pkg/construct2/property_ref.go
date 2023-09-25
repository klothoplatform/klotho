package construct2

import (
	"bytes"
	"fmt"
)

type PropertyRef struct {
	Resource ResourceId
	Property string
}

func (v PropertyRef) String() string {
	return v.Resource.String() + "#" + v.Property
}

func (v PropertyRef) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *PropertyRef) UnmarshalText(b []byte) error {
	parts := bytes.SplitN(b, []byte("#"), 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid PropertyRef format: %s", string(b))
	}
	err := v.Resource.UnmarshalText(parts[0])
	if err != nil {
		return err
	}
	v.Property = string(parts[1])
	return nil
}
