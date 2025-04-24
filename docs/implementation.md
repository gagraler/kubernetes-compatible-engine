# 实现细节

## 1. 版本发现机制

### 1.1 Discovery 实现

```go
type DiscoveryManager struct {
    client     *discovery.DiscoveryClient
    cache      *cache.Cache
    mutex      sync.RWMutex
    refreshTTL time.Duration
}

func (dm *DiscoveryManager) DiscoverAPIs() error {
    // 获取服务器支持的 API 资源列表
    apiResourceList, err := dm.client.ServerPreferredResources()
    if err != nil {
        return err
    }

    // 解析并缓存 GVR 信息
    for _, apiResource := range apiResourceList {
        gvr := schema.GroupVersionResource{
            Group:    apiResource.GroupVersion,
            Version:  apiResource.Version,
            Resource: apiResource.Name,
        }
        dm.cache.Set(gvr, apiResource)
    }

    return nil
}
```

### 1.2 版本映射

```go
type VersionMapper struct {
    preferredVersions map[string]string
    supportedVersions map[string][]string
}

func (vm *VersionMapper) GetPreferredVersion(gk schema.GroupKind) string {
    return vm.preferredVersions[gk.String()]
}

func (vm *VersionMapper) GetSupportedVersions(gk schema.GroupKind) []string {
    return vm.supportedVersions[gk.String()]
}
```

## 2. 动态客户端实现

### 2.1 资源操作封装

```go
type DynamicClient struct {
    client     dynamic.Interface
    mapper     meta.RESTMapper
    discovery  *DiscoveryManager
    converter  *ResourceConverter
}

func (dc *DynamicClient) Create(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) error {
    // 获取目标命名空间
    ns := obj.GetNamespace()
    if ns == "" {
        ns = "default"
    }

    // 创建资源
    _, err := dc.client.Resource(gvr).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
    return err
}
```

### 2.2 版本转换

```go
type ResourceConverter struct {
    schemaCache *cache.Cache
}

func (rc *ResourceConverter) ConvertToPreferredVersion(obj *unstructured.Unstructured) error {
    // 获取当前版本
    currentVersion := obj.GetAPIVersion()
    
    // 获取首选版本
    preferredVersion := rc.GetPreferredVersion(obj.GetObjectKind().GroupVersionKind().GroupKind())
    
    // 如果需要转换
    if currentVersion != preferredVersion {
        return rc.Convert(obj, preferredVersion)
    }
    
    return nil
}
```

## 3. 适配器实现

### 3.1 基础适配器

```go
type BaseAdapter struct {
    dynamicClient *DynamicClient
    gvr          schema.GroupVersionResource
    converter    *ResourceConverter
}

func (ba *BaseAdapter) Create(ctx context.Context, obj *unstructured.Unstructured) error {
    // 转换为首选版本
    if err := ba.converter.ConvertToPreferredVersion(obj); err != nil {
        return err
    }

    // 创建资源
    return ba.dynamicClient.Create(ctx, ba.gvr, obj)
}
```

### 3.2 特定资源适配器

```go
type DeploymentAdapter struct {
    *BaseAdapter
}

func (da *DeploymentAdapter) Create(ctx context.Context, deployment *appsv1.Deployment) error {
    // 转换为 unstructured
    obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
    if err != nil {
        return err
    }

    // 使用基础适配器创建
    return da.BaseAdapter.Create(ctx, &unstructured.Unstructured{Object: obj})
}
```

## 4. 字段兼容处理

### 4.1 字段映射

```go
type FieldMapper struct {
    fieldMappings map[string]map[string]string
}

func (fm *FieldMapper) MapFields(obj *unstructured.Unstructured, targetVersion string) error {
    // 获取源版本和目标版本的字段映射
    mappings := fm.fieldMappings[obj.GetAPIVersion()][targetVersion]
    
    // 应用字段映射
    for src, dst := range mappings {
        if value, exists := obj.Object[src]; exists {
            obj.Object[dst] = value
            delete(obj.Object, src)
        }
    }
    
    return nil
}
```

### 4.2 字段校验

```go
type FieldValidator struct {
    schemas map[string]*schema.Schema
}

func (fv *FieldValidator) Validate(obj *unstructured.Unstructured) error {
    // 获取对应版本的 schema
    s := fv.schemas[obj.GetAPIVersion()]
    
    // 验证字段
    return s.Validate(obj.Object)
}
```

## 5. 缓存实现

### 5.1 GVR 缓存

```go
type GVRCache struct {
    cache  map[string]schema.GroupVersionResource
    mutex  sync.RWMutex
    ttl    time.Duration
}

func (gc *GVRCache) Get(gk schema.GroupKind) (schema.GroupVersionResource, bool) {
    gc.mutex.RLock()
    defer gc.mutex.RUnlock()
    
    gvr, exists := gc.cache[gk.String()]
    return gvr, exists
}
```

### 5.2 Schema 缓存

```go
type SchemaCache struct {
    cache  map[string]*schema.Schema
    mutex  sync.RWMutex
    ttl    time.Duration
}

func (sc *SchemaCache) Get(apiVersion string) (*schema.Schema, bool) {
    sc.mutex.RLock()
    defer sc.mutex.RUnlock()
    
    s, exists := sc.cache[apiVersion]
    return s, exists
}
```

## 6. 错误处理

### 6.1 错误类型

```go
type CompatibilityError struct {
    Code    string
    Message string
    Cause   error
}

func (ce *CompatibilityError) Error() string {
    return fmt.Sprintf("%s: %s (cause: %v)", ce.Code, ce.Message, ce.Cause)
}
```

### 6.2 错误处理

```go
func handleError(err error) error {
    switch {
    case errors.Is(err, &CompatibilityError{}):
        // 处理兼容性错误
        return fmt.Errorf("compatibility error: %v", err)
    case errors.Is(err, &apierrors.StatusError{}):
        // 处理 API 错误
        return fmt.Errorf("api error: %v", err)
    default:
        // 处理其他错误
        return fmt.Errorf("unexpected error: %v", err)
    }
}
``` 