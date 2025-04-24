# 自定义资源 (CRD) 支持

## 1. 注册方式

### 1.1 YAML 文件注册

通过 YAML 文件注册自定义资源的 GVR 映射：

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
        - version: v1beta1
    - kind: AnotherCustomResource
      group: anothergroup.example.com
      versions:
        - version: v1alpha1
          preferred: true
```

注册方法：

```go
err := adapter.LoadKindGVRFromFile("kind-gvr.yaml")
if err != nil {
    // 处理错误
}
```

### 1.2 代码注册

通过代码直接注册：

```go
mapping := &adapter.KindGVRMapping{
    Kind: "MyCustomResource",
    Group: "mygroup.example.com",
    Versions: []adapter.VersionMapping{
        {
            Version: "v1",
            Preferred: true,
        },
        {
            Version: "v1beta1",
        },
    },
}

err := adapter.RegisterKindGVR(mapping)
if err != nil {
    // 处理错误
}
```

## 2. 使用示例

### 2.1 创建自定义资源

```go
adapter, err := adapter.NewCompatibleEngineAdapterFactory(disco, dyn, "MyCustomResource")
if err != nil {
    // 处理错误
}

resource := &adapter.CompatibleEngine{
    Name: "my-custom-resource",
    Kind: "MyCustomResource",
    Spec: map[string]interface{}{
        "field1": "value1",
        "field2": 123,
    },
}

err = adapter.Create(ctx, "default", resource)
if err != nil {
    // 处理错误
}
```

### 2.2 更新自定义资源

```go
resource.Spec["field1"] = "new-value"
err = adapter.Update(ctx, "default", resource)
if err != nil {
    // 处理错误
}
```

## 3. 字段校验

### 3.1 自动校验

```go
err := adapter.ValidateSpecFromSchema(
    resource.Spec,
    "mygroup.example.com",
    "v1",
    "MyCustomResource",
)
if err != nil {
    // 处理校验错误
}
```

### 3.2 自定义校验规则

```go
validator := &adapter.CustomValidator{
    Rules: []adapter.ValidationRule{
        {
            Field: "spec.field1",
            Required: true,
            Type: "string",
        },
        {
            Field: "spec.field2",
            Type: "integer",
            Min: 0,
            Max: 100,
        },
    },
}

err := adapter.RegisterValidator("MyCustomResource", validator)
if err != nil {
    // 处理错误
}
```

## 4. 版本转换

### 4.1 自动转换

当自定义资源有多个版本时，适配器会自动处理版本转换：

```go
// 使用 v1 版本创建
resource := &adapter.CompatibleEngine{
    Kind: "MyCustomResource",
    Spec: map[string]interface{}{
        "newField": "value", // v1 新增字段
    },
}

// 如果集群只支持 v1beta1，会自动转换
err := adapter.Create(ctx, "default", resource)
```

### 4.2 自定义转换规则

```go
converter := &adapter.CustomConverter{
    FromVersion: "v1",
    ToVersion: "v1beta1",
    ConvertFunc: func(spec map[string]interface{}) map[string]interface{} {
        // 自定义转换逻辑
        converted := make(map[string]interface{})
        // ...
        return converted
    },
}

err := adapter.RegisterConverter("MyCustomResource", converter)
if err != nil {
    // 处理错误
}
```

## 5. 注意事项

1. **版本兼容性**
   - 确保注册的版本在集群中可用
   - 注意版本间的字段差异
   - 处理版本升级时的数据迁移

2. **性能考虑**
   - 大量 CRD 注册可能影响性能
   - 建议使用缓存机制
   - 定期刷新版本信息

3. **安全考虑**
   - 验证 CRD 的访问权限
   - 限制资源操作范围
   - 记录操作日志 