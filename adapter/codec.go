package adapter

import (
	"sigs.k8s.io/yaml"
)

func MarshalToYAML(obj interface{}) ([]byte, error) {
	return yaml.Marshal(obj)
}

func UnmarshalFromYAML(data []byte, out interface{}) error {
	return yaml.Unmarshal(data, out)
}
