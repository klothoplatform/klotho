package model

import (
	"github.com/google/uuid"
)

type UUID struct {
	uuid.UUID
}

func (u *UUID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsedUUID, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*u = UUID{parsedUUID}
	return nil
}
