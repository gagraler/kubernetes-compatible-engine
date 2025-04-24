# 支持的资源类型

## 1. 内置资源

| Kind         | 默认 GVR 优先级                     | 说明 |
|--------------|-------------------------------------|------|
| Deployment   | apps/v1 > apps/v1beta2 > extensions/v1beta1 | 支持多版本自动降级 |
| StatefulSet  | apps/v1                             | 仅支持稳定版本 |
| DaemonSet    | apps/v1                             | 仅支持稳定版本 |
| Job          | batch/v1                            | 仅支持稳定版本 |
| CronJob      | batch/v1                            | 仅支持稳定版本 |

## 2. 版本兼容性

### 2.1 Deployment

- **apps/v1**
  - 完全支持
  - 推荐版本
  - 所有字段可用

- **apps/v1beta2**
  - 部分支持
  - 自动转换到 v1
  - 某些字段可能被忽略

- **extensions/v1beta1**
  - 有限支持
  - 仅用于兼容旧版本
  - 部分字段可能无法使用

### 2.2 StatefulSet

- **apps/v1**
  - 完全支持
  - 所有功能可用
  - 推荐版本

### 2.3 DaemonSet

- **apps/v1**
  - 完全支持
  - 所有功能可用
  - 推荐版本

### 2.4 Job/CronJob

- **batch/v1**
  - 完全支持
  - 所有功能可用
  - 推荐版本

## 3. 字段支持

### 3.1 通用字段

所有资源类型都支持的通用字段：

```go
type CommonFields struct {
    Name        string            // 资源名称
    Namespace   string            // 命名空间
    Labels      map[string]string // 标签
    Annotations map[string]string // 注解
}
```

### 3.2 特定字段

#### 3.2.1 Deployment

```go
type DeploymentFields struct {
    Replicas     int32             // 副本数
    Image        string            // 容器镜像
    Ports        []ContainerPort   // 端口配置
    Env          []EnvVar          // 环境变量
    Resources    ResourceRequirements // 资源限制
}
```

#### 3.2.2 StatefulSet

```go
type StatefulSetFields struct {
    Replicas     int32             // 副本数
    ServiceName  string            // 服务名称
    VolumeClaims []PersistentVolumeClaim // 持久卷声明
}
```

## 4. 使用限制

1. **版本限制**
   - 最低支持 Kubernetes 1.9
   - 建议使用 1.16 及以上版本

2. **功能限制**
   - 某些高级特性可能不支持
   - 自定义资源需要额外注册
   - 部分字段可能无法自动转换

3. **性能限制**
   - 大规模集群可能需要优化
   - 频繁的版本切换可能影响性能
   - 建议使用缓存机制 