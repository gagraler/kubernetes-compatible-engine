# 示例代码

## 1. 基础操作

### 1.1 创建 Deployment

```go
package main

import (
    "context"
    "fmt"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/tools/clientcmd"
    "https://github.com/gagraler/kubernetes-compatible-engine/adapter"
)

func main() {
    // 创建 Kubernetes 客户端
    config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
    if err != nil {
        panic(err)
    }

    disco, err := discovery.NewDiscoveryClientForConfig(config)
    if err != nil {
        panic(err)
    }

    dyn, err := dynamic.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    // 创建适配器
    adapter, err := adapter.NewCompatibleEngineAdapterFactory(disco, dyn, "Deployment")
    if err != nil {
        panic(err)
    }

    // 创建 Deployment
    deployment := &adapter.CompatibleEngine{
        Name:     "nginx-deployment",
        Kind:     "Deployment",
        Labels:   map[string]string{"app": "nginx"},
        Replicas: 3,
        Image:    "nginx:1.14.2",
        SpecPatch: map[string]interface{}{
            "template": map[string]interface{}{
                "spec": map[string]interface{}{
                    "containers": []interface{}{
                        map[string]interface{}{
                            "name":  "nginx",
                            "image": "nginx:1.14.2",
                            "ports": []interface{}{
                                map[string]interface{}{
                                    "containerPort": 80,
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    err = adapter.Create(context.Background(), "default", deployment)
    if err != nil {
        panic(err)
    }
}
```

### 1.2 更新 Deployment

```go
// 获取现有 Deployment
deployment, err := adapter.Get(context.Background(), "default", "nginx-deployment")
if err != nil {
    panic(err)
}

// 更新副本数
deployment.Replicas = 5

// 更新镜像
deployment.Image = "nginx:1.16.1"

// 应用更新
err = adapter.Update(context.Background(), "default", deployment)
if err != nil {
    panic(err)
}
```

### 1.3 删除 Deployment

```go
err := adapter.Delete(context.Background(), "default", "nginx-deployment")
if err != nil {
    panic(err)
}
```

## 2. 高级操作

### 2.1 监听资源变化

```go
err := adapter.Watch(context.Background(), "default", func(eventType watch.EventType, obj *adapter.CompatibleEngine) {
    fmt.Printf("Event: %s, Name: %s\n", eventType, obj.Name)
    
    switch eventType {
    case watch.Added:
        fmt.Printf("New Deployment created: %s\n", obj.Name)
    case watch.Modified:
        fmt.Printf("Deployment modified: %s\n", obj.Name)
    case watch.Deleted:
        fmt.Printf("Deployment deleted: %s\n", obj.Name)
    }
})
if err != nil {
    panic(err)
}
```

### 2.2 批量操作

```go
// 列出所有 Deployment
deployments, err := adapter.List(context.Background(), "default")
if err != nil {
    panic(err)
}

// 批量更新
for _, dep := range deployments {
    if dep.Labels["app"] == "nginx" {
        dep.Image = "nginx:1.16.1"
        err = adapter.Update(context.Background(), "default", dep)
        if err != nil {
            fmt.Printf("Failed to update %s: %v\n", dep.Name, err)
        }
    }
}
```

## 3. 自定义资源操作

### 3.1 注册自定义资源

```yaml
# kind-gvr.yaml
apiVersion: v1
kind: KindGVRMapping
metadata:
  name: custom-resources
spec:
  mappings:
    - kind: MyCustomResource
      group: mygroup.example.com
      versions:
        - version: v1
          preferred: true
```

```go
// 注册自定义资源
err := adapter.LoadKindGVRFromFile("kind-gvr.yaml")
if err != nil {
    panic(err)
}
```

### 3.2 操作自定义资源

```go
// 创建自定义资源适配器
customAdapter, err := adapter.NewCompatibleEngineAdapterFactory(disco, dyn, "MyCustomResource")
if err != nil {
    panic(err)
}

// 创建自定义资源
resource := &adapter.CompatibleEngine{
    Name: "my-custom-resource",
    Kind: "MyCustomResource",
    Spec: map[string]interface{}{
        "field1": "value1",
        "field2": 123,
    },
}

err = customAdapter.Create(context.Background(), "default", resource)
if err != nil {
    panic(err)
}
```

## 4. 字段校验

### 4.1 基本校验

```go
// 校验 Deployment 规格
err := adapter.ValidateSpecFromSchema(
    deployment.Spec,
    "apps",
    "v1",
    "Deployment",
)
if err != nil {
    panic(err)
}
```

### 4.2 自定义校验规则

```go
// 注册自定义校验器
validator := &adapter.CustomValidator{
    Rules: []adapter.ValidationRule{
        {
            Field: "spec.replicas",
            Type: "integer",
            Min: 1,
            Max: 10,
        },
    },
}

err := adapter.RegisterValidator("Deployment", validator)
if err != nil {
    panic(err)
}
```

## 5. 错误处理

### 5.1 基本错误处理

```go
err := adapter.Create(context.Background(), "default", deployment)
if err != nil {
    switch {
    case adapter.IsNotFound(err):
        fmt.Println("Resource not found")
    case adapter.IsAlreadyExists(err):
        fmt.Println("Resource already exists")
    case adapter.IsInvalid(err):
        fmt.Println("Invalid resource specification")
    default:
        fmt.Printf("Unexpected error: %v\n", err)
    }
}
```

### 5.2 重试机制

```go
func createWithRetry(ctx context.Context, adapter adapter.ICompatibleEngine, deployment *adapter.CompatibleEngine) error {
    var lastErr error
    for i := 0; i < 3; i++ {
        err := adapter.Create(ctx, "default", deployment)
        if err == nil {
            return nil
        }
        lastErr = err
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return lastErr
}
```

## 6. 性能优化

### 6.1 批量操作优化

```go
// 使用 goroutine 并行处理
var wg sync.WaitGroup
for _, dep := range deployments {
    wg.Add(1)
    go func(d *adapter.CompatibleEngine) {
        defer wg.Done()
        err := adapter.Update(context.Background(), "default", d)
        if err != nil {
            fmt.Printf("Failed to update %s: %v\n", d.Name, err)
        }
    }(dep)
}
wg.Wait()
```

### 6.2 缓存优化

```go
// 启用 GVR 缓存
adapter.EnableGVRCache(true)

// 设置缓存刷新间隔
adapter.SetCacheRefreshInterval(time.Minute * 5)
``` 
