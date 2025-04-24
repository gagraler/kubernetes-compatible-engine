# Kubernetes 兼容引擎

## 1. 项目概述

Kubernetes 兼容引擎是一个用于处理多版本 Kubernetes 集群资源操作的通用解决方案。它采用"Discovery + Dynamic"方案，通过动态发现和自动适配，实现对不同 Kubernetes 版本的透明兼容。

### 1.1 核心特性

- **自动版本发现与选择**
  - 动态发现集群支持的 API 版本
  - 智能选择最优资源版本
  - 支持版本自动降级

- **统一资源操作接口**
  - 屏蔽底层版本差异
  - 提供一致的 CRUD 操作
  - 支持资源监听和事件处理

- **字段兼容处理**
  - 自动处理版本间字段差异
  - 支持自定义字段转换规则
  - 提供字段校验功能

### 1.2 适用场景

- 需要同时支持多个 Kubernetes 版本的应用
- 需要处理版本升级和降级的场景
- 需要统一管理不同版本集群资源的场景

## 2. 快速开始

### 2.1 安装

```bash
go get github.com/your-org/kubernetes-compatible-engine
```

### 2.2 基本使用

```go
package main

import (
    "context"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/tools/clientcmd"
    "path/to/adapter"
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
    }

    err = adapter.Create(context.Background(), "default", deployment)
    if err != nil {
        panic(err)
    }
}
```

## 3. 文档目录

- [架构设计](architecture.md) - 详细说明系统架构和设计思路
- [支持的资源类型](resources.md) - 列出所有支持的资源类型和版本
- [自定义资源支持](crd.md) - 说明如何注册和使用自定义资源
- [字段校验](validation.md) - 详细介绍字段校验功能
- [示例代码](examples.md) - 提供各种使用场景的示例

## 4. 贡献指南

欢迎提交 Issue 和 Pull Request 来帮助改进项目。在提交代码前，请确保：

1. 代码符合 Go 代码规范
2. 添加必要的单元测试
3. 更新相关文档

## 5. 许可证

本项目采用 Apache License 2.0 许可证。 