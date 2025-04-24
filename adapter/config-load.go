package adapter

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"sigs.k8s.io/yaml"
)

type KindGVRMapping map[string][]schema.GroupVersion

type RawGVR struct {
	Group   string `json:"group" yaml:"group"`
	Version string `json:"version" yaml:"version"`
}

// LoadKindGVRFromFile 从文件加载 Kind 到 GVR 的映射
func LoadKindGVRFromFile(filePath string) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var rawMap map[string][]RawGVR
	if err := yaml.Unmarshal(raw, &rawMap); err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	for kind, list := range rawMap {
		var gvs []schema.GroupVersion
		for _, entry := range list {
			gvs = append(gvs, schema.GroupVersion{
				Group:   entry.Group,
				Version: entry.Version,
			})
		}
		RegisterKind(kind, gvs)
	}
	return nil
}
