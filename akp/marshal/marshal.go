package marshal

import (
	"google.golang.org/protobuf/types/known/structpb"

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

// ApiModelToPBStruct convert an object to a protobuf struct by marshalling and unmarshalling it.
func ApiModelToPBStruct(obj interface{}) (*structpb.Struct, error) {
	m := map[string]interface{}{}
	if err := RemarshalTo(obj, &m); err != nil {
		return nil, err
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("new struct: %w", err)
	}
	return s, nil
}
