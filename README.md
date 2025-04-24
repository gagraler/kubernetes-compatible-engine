# Kubernetes Compatible Engine

用于处理多版本 Kubernetes 集群的兼容性适配器

## ✨ Features

- ✅ 自动发现并选择合适的 GroupVersionResource（GVR）
- ✅ 支持 Kubernetes 多主版本兼容（已测试跨 10+ 版本）
- ✅ 支持自定义资源 Kind → GVR 映射注册（支持 CRD）
- ✅ 使用 Canonical 数据结构统一上层接口
- ✅ 内置字段结构差异自动转换
- ✅ 支持 `.spec` 字段自动校验（基于 OpenAPI Schema）
- ✅ 支持 YAML / JSON 导入导出
- ✅ 支持 controller `.metadata` 字段导出（annotations、ownerReferences 等）

## 🚀 Quick Start

```go
adapter, _ := adapter.NewCompatibleEngineAdapterFactory(disco, dyn, "Deployment")

engine := &adapter.CompatibleEngine{
    Name:     "demo-nginx",
    Kind:     "Deployment",
    Labels:   map[string]string{"app": "nginx"},
    Replicas: 2,
    Image:    "nginx:1.21",
}

_ = adapter.Create(ctx, "default", engine)
```

## 📚 Docs Folder

- [架构设计](docs/architecture.md) - 整体架构和设计思路
- [资源支持](docs/resources.md) - 支持的资源类型和版本
- [CRD 支持](docs/crd.md) - 自定义资源注册和使用
- [字段校验](docs/validation.md) - 字段校验功能说明
- [示例代码](docs/examples.md) - 使用示例

## 📦 Install

```bash
go get github.com/gagraler/kubernetes-compatible-engine
```

## 🤝 Contribute

欢迎提交 Issue 和 Pull Request！


## 📝 License

Copyright 2023 gagral.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software