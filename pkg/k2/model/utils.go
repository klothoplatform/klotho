package model

import (
	"fmt"

	"github.com/google/uuid"
)

type UUID struct {
	uuid.UUID
}

func (u *UUID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return fmt.Errorf("error unmarshalling YAML string: %w", err)
	}
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return fmt.Errorf("error parsing UUID: %w", err)
	}
	*u = UUID{parsedUUID}
	return nil
}
