package adapter

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// Cache 用于缓存 Kind 到 GVR 的映射
type Cache struct {
	lock   sync.RWMutex
	items  map[string]schema.GroupVersionResource
	loaded bool
}

// GlobalGVRCache global cache 实例（可用于默认适配器）
var GlobalGVRCache = &Cache{items: make(map[string]schema.GroupVersionResource)}

// Get 返回缓存中的 GVR
func (c *Cache) Get(kind string) (schema.GroupVersionResource, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	gvr, ok := c.items[kind]
	return gvr, ok
}

// Set 向缓存中写入 GVR
func (c *Cache) Set(kind string, gvr schema.GroupVersionResource) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[kind] = gvr
}

// Refresh 刷新 GVR 缓存
func (c *Cache) Refresh(disco discovery.DiscoveryInterface, knownKinds map[string][]schema.GroupVersion) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	resourceLists, err := disco.ServerPreferredResources()
	if err != nil {
		return err
	}
	possible := make(map[schema.GroupVersion]string)
	for _, rl := range resourceLists {
		gv, _ := schema.ParseGroupVersion(rl.GroupVersion)
		for _, r := range rl.APIResources {
			possible[gv] = r.Name
		}
	}
	for kind, gvs := range knownKinds {
		for _, gv := range gvs {
			if res, ok := possible[gv]; ok {
				c.items[kind] = schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: res,
				}
				break
			}
		}
	}
	c.loaded = true
	return nil
}
