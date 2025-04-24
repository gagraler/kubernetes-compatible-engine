package adapter

import "k8s.io/apimachinery/pkg/runtime/schema"

var knownKindGVRs = map[string][]schema.GroupVersion{
	"Deployment": {
		{Group: "apps", Version: "v1"},
		{Group: "apps", Version: "v1beta2"},
		{Group: "extensions", Version: "v1beta1"},
	},
	"StatefulSet": {
		{Group: "apps", Version: "v1"},
	},
	"DaemonSet": {
		{Group: "apps", Version: "v1"},
	},
	"Job": {
		{Group: "batch", Version: "v1"},
	},
	"CronJob": {
		{Group: "batch", Version: "v1"},
	},
	// Add more known kinds here
}

// RegisterKind 用于注册新的 Kind 到 GVR 映射
func RegisterKind(kind string, gvs []schema.GroupVersion) {
	knownKindGVRs[kind] = gvs
}
