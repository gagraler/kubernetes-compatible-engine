package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// CompatibleEngine 为所有控制器（例如 Deployment、StatefulSet、DaemonSet）定义一个通用结构
type CompatibleEngine struct {
	Name        string
	Labels      map[string]string
	Replicas    int32
	Image       string
	APIVersions []string
	Kind        string
	Spec        map[string]interface{}
	Status      map[string]interface{}
	Metadata    map[string]interface{}
	SpecPatch   map[string]interface{}
}

// ICompatibleEngine 对 Kubernetes 资源的基本操作
type ICompatibleEngine interface {
	Create(ctx context.Context, ns string, c *CompatibleEngine) error
	Update(ctx context.Context, ns string, c *CompatibleEngine) error
	Get(ctx context.Context, ns, name string) (*CompatibleEngine, error)
	Delete(ctx context.Context, ns, name string) error
	List(ctx context.Context, ns string) ([]*CompatibleEngine, error)
	Patch(ctx context.Context, ns, name string, patch map[string]interface{}) error
	Watch(ctx context.Context, ns string, event func(eventType watch.EventType, obj *CompatibleEngine)) error
	ExportYAML(ctx context.Context, ns, name string) (string, error)
}

// compatibleEngineAdapter 适配器实现
type compatibleEngineAdapter struct {
	dynamicClient dynamic.Interface
	gvr           schema.GroupVersionResource
	kind          string
}

func NewCompatibleEngineAdapter(disco discovery.DiscoveryInterface, dyn dynamic.Interface, kind string) (ICompatibleEngine, error) {
	gvr, err := selectGVR(disco, kind)
	if err != nil {
		return nil, err
	}
	return &compatibleEngineAdapter{dynamicClient: dyn, gvr: gvr, kind: kind}, nil
}

// Create 创建资源
func (a *compatibleEngineAdapter) Create(ctx context.Context, namespace string, c *CompatibleEngine) error {
	u, err := toUnstructured(a.gvr.GroupVersion(), c)
	if err != nil {
		return err
	}
	res := a.dynamicClient.Resource(a.gvr).Namespace(namespace)
	existing, err := res.Get(ctx, c.Name, metav1.GetOptions{})
	if err == nil {
		existing.Object["spec"] = u.Object["spec"]
		existing.Object["metadata"].(map[string]interface{})["labels"] = u.Object["metadata"].(map[string]interface{})["labels"]
		_, err := res.Update(ctx, existing, metav1.UpdateOptions{})
		return err
	}
	_, err = res.Create(ctx, u, metav1.CreateOptions{})
	return err
}

// Update 更新资源
func (a *compatibleEngineAdapter) Update(ctx context.Context, namespace string, c *CompatibleEngine) error {
	u, err := toUnstructured(a.gvr.GroupVersion(), c)
	if err != nil {
		return err
	}
	res := a.dynamicClient.Resource(a.gvr).Namespace(namespace)
	existing, err := res.Get(ctx, c.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	existing.Object["spec"] = u.Object["spec"]
	existing.Object["metadata"].(map[string]interface{})["labels"] = u.Object["metadata"].(map[string]interface{})["labels"]
	_, err = res.Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// Get 获取资源
func (a *compatibleEngineAdapter) Get(ctx context.Context, namespace, name string) (*CompatibleEngine, error) {
	res := a.dynamicClient.Resource(a.gvr).Namespace(namespace)
	u, err := res.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return fromUnstructured(a.gvr.GroupVersion(), a.kind, u)
}

// Delete 删除资源
func (a *compatibleEngineAdapter) Delete(ctx context.Context, namespace, name string) error {
	return a.dynamicClient.Resource(a.gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// List 列出资源
func (a *compatibleEngineAdapter) List(ctx context.Context, namespace string) ([]*CompatibleEngine, error) {
	res := a.dynamicClient.Resource(a.gvr).Namespace(namespace)
	ul, err := res.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []*CompatibleEngine
	for _, item := range ul.Items {
		c, err := fromUnstructured(a.gvr.GroupVersion(), a.kind, &item)
		if err == nil {
			result = append(result, c)
		}
	}
	return result, nil
}

// Patch 更新资源的部分字段
func (a *compatibleEngineAdapter) Patch(ctx context.Context, namespace, name string, patch map[string]interface{}) error {
	res := a.dynamicClient.Resource(a.gvr).Namespace(namespace)
	u, err := res.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for k, v := range patch {
		u.Object[k] = v
	}
	_, err = res.Update(ctx, u, metav1.UpdateOptions{})
	return err
}

// Watch 监听资源事件
func (a *compatibleEngineAdapter) Watch(ctx context.Context, namespace string, onEvent func(eventType watch.EventType, obj *CompatibleEngine)) error {
	watcher, err := a.dynamicClient.Resource(a.gvr).Namespace(namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	ch := watcher.ResultChan()
	go func() {
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-ch:
				if !ok {
					return
				}
				if u, ok := evt.Object.(*unstructured.Unstructured); ok {
					c, err := fromUnstructured(a.gvr.GroupVersion(), a.kind, u)
					if err == nil {
						onEvent(evt.Type, c)
					}
				}
			}
		}
	}()
	return nil
}

// ExportYAML 导出资源为 YAML 格式
func (a *compatibleEngineAdapter) ExportYAML(ctx context.Context, namespace, name string) (string, error) {
	u, err := a.dynamicClient.Resource(a.gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(u.Object, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// selectGVR 选择 GVR
func selectGVR(disco discovery.DiscoveryInterface, kind string) (schema.GroupVersionResource, error) {
	resourceLists, err := disco.ServerPreferredResources()
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	targets := map[string][]schema.GroupVersion{
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
	}

	possible := make(map[schema.GroupVersion]string)
	for _, rl := range resourceLists {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rl.APIResources {
			if r.Kind == kind {
				possible[gv] = r.Name
			}
		}
	}

	for _, gv := range targets[kind] {
		if res, ok := possible[gv]; ok {
			return schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: res}, nil
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("no supported %s version found", kind)
}

// toUnstructured 将 CompatibleEngine 转换为 unstructured.Unstructured
func toUnstructured(gv schema.GroupVersion, c *CompatibleEngine) (*unstructured.Unstructured, error) {

	base := map[string]interface{}{
		"apiVersion": fmt.Sprintf("%s/%s", gv.Group, gv.Version),
		"kind":       c.Kind,
		"metadata": map[string]interface{}{
			"name":   c.Name,
			"labels": c.Labels,
		},
	}

	containerSpec := map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"name":  "main",
				"image": c.Image,
			},
		},
	}

	if c.SpecPatch != nil {
		for k, v := range c.SpecPatch {
			base["spec"].(map[string]interface{})[k] = v
		}
	}

	switch c.Kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		base["spec"] = map[string]interface{}{
			"replicas": c.Replicas,
			"selector": map[string]interface{}{
				"matchLabels": c.Labels,
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": c.Labels,
				},
				"spec": containerSpec,
			},
		}
	case "Job":
		base["spec"] = map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": c.Labels,
				},
				"spec": containerSpec,
			},
			"backoffLimit": int32(4),
		}
	case "CronJob":
		base["spec"] = map[string]interface{}{
			"schedule": "*/1 * * * *",
			"jobTemplate": map[string]interface{}{
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"metadata": map[string]interface{}{
							"labels": c.Labels,
						},
						"spec": containerSpec,
					},
				},
			},
		}
	}

	return &unstructured.Unstructured{Object: base}, nil
}

// fromUnstructured 将 unstructured.Unstructured 转换为 CompatibleEngine
func fromUnstructured(gv schema.GroupVersion, kind string, u *unstructured.Unstructured) (*CompatibleEngine, error) {
	name, _, _ := unstructured.NestedString(u.Object, "metadata", "name")
	labels, _, _ := unstructured.NestedStringMap(u.Object, "metadata", "labels")
	replicas, _, _ := unstructured.NestedInt64(u.Object, "spec", "replicas")
	spec, _, _ := unstructured.NestedMap(u.Object, "spec")
	status, _, _ := unstructured.NestedMap(u.Object, "status")
	annotations, _, _ := unstructured.NestedStringMap(u.Object, "metadata", "annotations")
	ownerRefs, _, _ := unstructured.NestedSlice(u.Object, "metadata", "ownerReferences")
	createdAt, _, _ := unstructured.NestedString(u.Object, "metadata", "creationTimestamp")
	apiVersions, _, _ := unstructured.NestedStringSlice(u.Object, "apiVersion")

	cList, _, _ := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
	var image string
	if len(cList) > 0 {
		if cMap, ok := cList[0].(map[string]interface{}); ok {
			if img, ok := cMap["image"].(string); ok {
				image = img
			}
		}
	}

	meta := map[string]interface{}{
		"annotations":       annotations,
		"ownerReferences":   ownerRefs,
		"creationTimestamp": createdAt,
	}
	return &CompatibleEngine{
		Name:        name,
		Labels:      labels,
		Replicas:    int32(replicas),
		Image:       image,
		Kind:        kind,
		APIVersions: apiVersions,
		Spec:        spec,
		Status:      status,
		Metadata:    meta,
	}, nil
}
