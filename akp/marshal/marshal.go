package marshal

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// RemarshalTo convert an object to a target object by marshalling and unmarshalling it.
func RemarshalTo(obj, target interface{}) error {
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

// ProtoToMap converts a protobuf message to a map[string]any using protojson serialization.
// This produces camelCase keys matching protobuf JSON conventions, which BuildStateFromAPI
// then converts to snake_case to match tfsdk struct tags.
func ProtoToMap(msg proto.Message) (map[string]any, error) {
	data, err := protojson.MarshalOptions{
		EmitUnpopulated: false,
	}.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("protojson marshal: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return m, nil
}
