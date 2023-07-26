package marshal

import (
	"encoding/json"
	"fmt"
)

// RemarshalTo convert an object to a target object by marshalling and unmarshalling it.
func RemarshalTo(obj interface{}, target interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}
