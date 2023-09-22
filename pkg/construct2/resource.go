package construct2

import (
	"encoding/json"
	"fmt"
)

type (
	Resource struct {
		ID         ResourceId
		Properties Properties
	}

	Properties = map[string]interface{}
)

// Id is a temporary bridge to the old Resource interface. Remove in favour of direct ID field access.
func (r *Resource) Id() ResourceId {
	return r.ID
}

func (r *Resource) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{}, len(r.Properties)+1)
	m["$id"] = r.ID
	for k, v := range r.Properties {
		m[k] = v
	}
	return json.Marshal(m)
}

func (r *Resource) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	id, ok := m["$id"]
	if !ok {
		return fmt.Errorf("missing $id")
	}
	idStr, ok := id.(string)
	if !ok {
		return fmt.Errorf("$id is not a string (got %T)", id)
	}
	if err := r.ID.UnmarshalText([]byte(idStr)); err != nil {
		return err
	}
	delete(m, "$id")
	r.Properties = m
	return nil
}
