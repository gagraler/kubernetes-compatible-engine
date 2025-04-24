# 字段校验

## 1. 概述

适配器提供了强大的字段校验功能，可以确保资源的字段符合规范。校验功能包括：

- 自动从 OpenAPI Schema 提取校验规则
- 支持自定义校验规则
- 支持版本特定的校验规则
- 支持嵌套字段校验

## 2. 自动校验

### 2.1 基于 Schema 的校验

```go
// 从 OpenAPI Schema 自动提取校验规则
err := adapter.ValidateSpecFromSchema(
    resource.Spec,
    "apps",
    "v1",
    "Deployment",
)
if err != nil {
    // 处理校验错误
}
```

### 2.2 校验规则示例

```go
// Deployment 的校验规则示例
rules := map[string]interface{}{
    "spec": map[string]interface{}{
        "replicas": map[string]interface{}{
            "type": "integer",
            "minimum": 0,
        },
        "template": map[string]interface{}{
            "spec": map[string]interface{}{
                "containers": map[string]interface{}{
                    "type": "array",
                    "minItems": 1,
                    "items": map[string]interface{}{
                        "required": ["name", "image"],
                        "properties": map[string]interface{}{
                            "name": map[string]interface{}{
                                "type": "string",
                            },
                            "image": map[string]interface{}{
                                "type": "string",
                            },
                        },
                    },
                },
            },
        },
    },
}
```

## 3. 自定义校验

### 3.1 注册校验器

```go
validator := &adapter.CustomValidator{
    Rules: []adapter.ValidationRule{
        {
            Field: "spec.replicas",
            Type: "integer",
            Min: 0,
            Max: 10,
        },
        {
            Field: "spec.template.spec.containers[].image",
            Type: "string",
            Pattern: "^[a-zA-Z0-9./:-]+$",
        },
    },
}

err := adapter.RegisterValidator("Deployment", validator)
if err != nil {
    // 处理错误
}
```

### 3.2 自定义校验函数

```go
validator := &adapter.CustomValidator{
    ValidateFunc: func(spec map[string]interface{}) error {
        // 自定义校验逻辑
        if replicas, ok := spec["replicas"].(int); ok {
            if replicas < 0 || replicas > 10 {
                return fmt.Errorf("replicas must be between 0 and 10")
            }
        }
        return nil
    },
}
```

## 4. 版本特定校验

### 4.1 注册版本特定校验器

```go
// v1 版本的校验规则
v1Validator := &adapter.CustomValidator{
    Rules: []adapter.ValidationRule{
        {
            Field: "spec.strategy.type",
            Enum: []string{"RollingUpdate", "Recreate"},
        },
    },
}

// v1beta1 版本的校验规则
v1beta1Validator := &adapter.CustomValidator{
    Rules: []adapter.ValidationRule{
        {
            Field: "spec.strategy.type",
            Enum: []string{"RollingUpdate"},
        },
    },
}

err := adapter.RegisterVersionValidator("Deployment", "v1", v1Validator)
err = adapter.RegisterVersionValidator("Deployment", "v1beta1", v1beta1Validator)
```

## 5. 错误处理

### 5.1 校验错误类型

```go
type ValidationError struct {
    Field   string
    Message string
    Value   interface{}
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: field %s: %s (value: %v)", 
        e.Field, e.Message, e.Value)
}
```

### 5.2 错误处理示例

```go
err := adapter.ValidateSpec(resource.Spec)
if err != nil {
    if verr, ok := err.(*adapter.ValidationError); ok {
        // 处理校验错误
        fmt.Printf("Field %s is invalid: %s\n", verr.Field, verr.Message)
    } else {
        // 处理其他错误
    }
}
```

## 6. 最佳实践

1. **校验时机**
   - 在创建资源前进行校验
   - 在更新资源前进行校验
   - 在导入 YAML 时进行校验

2. **性能优化**
   - 缓存校验规则
   - 批量校验时复用校验器
   - 避免重复校验

3. **错误提示**
   - 提供清晰的错误信息
   - 包含字段路径和期望值
   - 支持多语言错误消息

4. **扩展性**
   - 支持自定义校验规则
   - 支持插件式校验器
   - 支持条件校验 