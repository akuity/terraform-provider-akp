package marshal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v2"
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

func YamlRemarshalTo(obj interface{}, target interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tflog.Info(context.Background(), fmt.Sprintf("------data:%s", string(data)))
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}
