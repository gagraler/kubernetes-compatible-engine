package adapter

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// NewCompatibleEngineAdapterFactory 适配器工厂
func NewCompatibleEngineAdapterFactory(
	disco discovery.DiscoveryInterface, dyn dynamic.Interface, kind string) (ICompatibleEngine, error) {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
		return NewCompatibleEngineAdapter(disco, dyn, kind)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}
