package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUIDUnmarshalYAML(t *testing.T) {
	u := &UUID{}
	err := u.UnmarshalYAML(func(v interface{}) error {
		*v.(*string) = "123e4567-e89b-12d3-a456-426614174000"
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to unmarshal UUID: %v", err)
	}
	expectedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	if u.UUID != expectedUUID {
		t.Errorf("Expected UUID to be %s, got %s", expectedUUID, u.UUID)
	}
}
