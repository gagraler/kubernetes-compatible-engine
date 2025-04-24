# 架构设计

当需要同时对接多个（跨度较大，例如跨 10 个以上的主版本）Kubernetes 版本，并且希望在运行时根据可用的 API 版本自动选择合适的资源版本进行操作时，**"Discovery + Dynamic"** 是一个较为通用和灵活的组合方案。下面整理一套思路，帮助开发者在应用层做统一的逻辑抽象，从而在运行时针对不同 K8s 版本的集群进行自动适配。

---

## 1. 整体思路概览
1. **通过 Discovery**  
先使用 K8s 的 Discovery 接口，获取当前集群所有可用的"Group-Version-Resource (GVR)"信息，也就是"API 资源列表"和对应可用的版本。
    - Discovery 结果会告诉：
        * 集群中存在哪些 group？
        * 每个 group 下有哪些版本可用？
        * 每个版本下可以操作哪些资源（resource）？
        * 某些资源是否只读（namespaced / cluster scope 等信息）
    - 对于同一个资源（如 `Deployment`），可能能发现它在某些旧版本集群里叫 `extensions/v1beta1`，在新版本里叫 `apps/v1` 等。
2. **对业务所需资源做"版本映射与选择"**
    - 需要在的业务逻辑中预先知道：
        * 自己究竟要操作哪些资源（例如 `Deployment`、`StatefulSet`、`Ingress`、`Job` 等）。
        * 这些资源在不同 K8s 版本中的**优先版本**或**已弃用版本**。
    - 基于 Discovery 返回的可用版本列表，为每个资源选择**最优**或**最兼容**的那个版本。例如：
        * 如果能发现 `apps/v1` 的 `Deployment`，就用它；
        * 如果 `apps/v1` 不存在，但 `extensions/v1beta1` 存在，则退回使用 `extensions/v1beta1`；
        * 若想操作某些 CRD，则同样通过 Discovery 确认 `group/version` 是否可用。
    - 需要维护一份**可选版本清单**和**优先顺序**。当出现多个候选版本时，按定义的逻辑（"越新越优先"或"从 GA 到 Beta 到 Alpha"）来确定实际使用哪个版本。
3. **使用 Dynamic Client 做资源的 CRUD**
    - 在经过上一步的"版本选择"后，已经知道：要对某个资源（比如 `Deployment`）究竟要调用 `apps/v1` 还是 `extensions/v1beta1`。
    - 通过 `dynamic.Interface`，构造对应的 `ResourceInterface`，然后进行 `Create/Get/Update/Patch/Delete` 等操作。
    - 读写时，需要对传入/传出的数据做必要的**结构转换**（因为不同版本可能字段略有差异）。
    - 在很多场合，可以直接对 `Unstructured`（即 JSON map）操作，不必做强类型反序列化；但如果需要深度操作资源字段（比如更新 Pod Spec 的具体字段），就要考虑在应用层提供一些"字段兼容适配"的逻辑。
4. **封装在一层"兼容适配模块"**
    - 为了让上层业务不必关心"我现在到底是操作 `apps/v1` 还是 `extensions/v1beta1`"，通常会在代码里封装一层"**Resource Adapter**"或"**Version Adapter**"逻辑。
    - 这层逻辑里包含：
        * "发现 + 选择"对应的 API 版本；
        * 针对不同版本的字段差异，做数据的转换（如可能某些 beta 字段在 GA 版本中更名或者结构稍有不同）。
    - 上层就只要调用一个统一的函数，例如：

```go
// Pseudocode
// canonicalDeployment 是一份与最新版本或自定义结构对齐的 Deployment 配置
adapter := NewCompatibleEngineAdapter(dynamicClient, discoveryClient, "Deployment")
err := adapter.Create(canonicalDeployment)
```

这时，由 `NewCompatibleEngineAdapter` 在底层决定要调用哪个版本的 GVR，并做合适的字段处理。

---

## 2. Discovery 与资源版本选择的细节
### 2.1 Discovery Client 获取可用资源列表
+ 在 `client-go` 里可以使用 `discovery.Client`（或其封装）来获取 API 资源：

```go
discoClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
if err != nil { ... }

// ServerPreferredResources 会返回所有 group/version 下的资源信息
apiResourceLists, err := discoClient.ServerPreferredResources()
if err != nil { ... }
```

+ `apiResourceLists` 是一个数组，每个元素对应一个 `APIResourceList`，里面包含 `GroupVersion` 和若干 `APIResource`。比如：

```go
// 举例: "apps/v1" 下有哪些 resources ?
for _, rl := range apiResourceLists {
    gvk := rl.GroupVersion // e.g. "apps/v1"
    for _, r := range rl.APIResources {
        // r.Kind, r.Name, r.Verbs, r.Namespaced, etc.
    }
}
```

### 2.2 做资源版本的匹配与优先级选择
基于拿到的 `apiResourceLists`，需要为每个关心的资源，进行以下逻辑：

1. **找出所有匹配其 kind/name 的 (group, version, resource)**
    - 例如要找 `Deployment`，可能会在 `apps/v1` 里看到 `deployments`，在 `extensions/v1beta1` 里也看到 `deployments`。
2. **根据优先级排序，选出最终使用的那个 GVR**
    - 通常会优先用最新稳定的，如 `apps/v1`；
    - 如果不存在，再尝试早期的 beta；
    - 如果还没有，就说明此集群可能太老或太新，不支持此资源，需要做相应的处理或报错。
3. **记录在缓存**
    - 因为 Discovery 不一定每次都要重复调用，可以在进程启动时或定期刷新一次，把结果缓存起来。
    - 假设要长期运行，建议做一下**版本变动**监控或周期性刷新，以防集群升级后版本变化（尤其涉及 CRD 时更常见）。

**提示**：如果需要管理非常多的版本、资源，Discovery 请求会比较大，官方也提供了 `ServerPreferredNamespacedResources()` 和 `ServerResourcesForGroupVersion(gv)` 等方法，或使用更细粒度的"注意缓存"和"Watch"机制来优化。

---

## 3. Dynamic Client 的使用要点
在 `client-go` 里使用动态客户端主要有以下要点：

1. **创建 `dynamic.Interface`**

```go
dynamicClient, err := dynamic.NewForConfig(restConfig)
if err != nil {
    // handle
}
```

2. **确定 GVR**
    - 例如 `GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}`。
3. **Namespace 范围**
    - 大多数资源是 namespaced 的，所以需要 `.Namespace("default")`；对于集群级（cluster-scoped）资源，则 `.Namespace("")` 或 `.NamespaceIfScoped("")`。
4. **操作 Unstructured 对象**
    - `Create` / `Update` / `Patch` / `Delete` / `Get` / `List` 等方法，返回或接收 `*unstructured.Unstructured`。
    - 可以直接在 `Object` 字段中操作 `map[string]interface{}`。
5. **字段兼容与结构转换**
    - 不同版本的 `Deployment` 可能在 `.spec` 里有细微差异，或者 `.metadata` 里会有一些 beta 字段。
    - 需要根据业务的需求，对这些字段做兼容处理或最小集合的读取/写入。
    - 在最简单的场景下，如果只对常规字段（name、labels、spec.replicas 等）做读写，这些字段多数情况下都能兼容；但如果深入到某些特定的字段（如 Ingress v1 vs v1beta1 的 pathType、backend 定义等），就需要手动区分不同版本的字段结构。

---

## 4. 核心逻辑的常见设计与封装
假设打算封装一套适配不同 K8s 版本的"上层逻辑"，可以做如下设计：

1. **Canonical 数据模型**
    - 在自己程序内部，定义一个（或一套）**Canonical 结构体**，它代表对于某个资源（如 Deployment）所需要的所有字段和含义。
    - 这个结构体可以按照最常见或较新（更稳定）的版本去设计（例如对 Deployment 就参照 `apps/v1` 的字段）。
    - 当需要**向集群写入**资源时，可以把 Canonical 结构体转换为"实际目标版本"所需的 JSON 结构（或 `Unstructured`）。
    - 当**从集群读取**资源时，也可以反过来，从不同版本的 JSON 对象 parse 回 Canonical 结构体，在应用层做统一处理。
2. **Version Adapter（资源适配器）**
    - 对于每一种资源（Deployment, Ingress, Job, etc.），都实现一个适配器：

```go
type compatibleEngineAdapter struct {
    dynamicClient dynamic.Interface
    gvr           schema.GroupVersionResource
    kind          string
}

func (d *compatibleEngineAdapter) Create(ctx context.Context, namespace string, c *CompatibleEngine) error { ... }
func (d *compatibleEngineAdapter) Get(ctx context.Context, namespace, name string) (*CompatibleEngine, error) { ... }
// ...
```

    - `compatibleEngineAdapter` 的构造函数里，会用 Discovery 的结果决定 `preferredGVR` 是什么。如果集群只支持 `extensions/v1beta1`，就用那个；如果能支持 `apps/v1`，就用 `apps/v1` 等。
    - 在 `Create` 里，把传入的 CompatibleEngine 结构体转为对应版本的 `Unstructured`。针对不同版本可能要兼容处理字段差异（例如某些字段只有在 `apps/v1` 中才存在，或者 `extensions/v1beta1` 与 `apps/v1` 下的容器 spec 有些位置不同等）。
    - 这部分"转换"逻辑可以写成一段映射代码，也可以借助一些模版化/反射技巧，但大多数时候还是手写比较直观可控。
3. **多版本兼容策略**
    - 当发现集群**完全不支持**想要管理的资源版本时，需要在适配器的初始化阶段就抛错，或在 CRUD 时返回错误。
    - 如果要支持跨非常多版本（例如 K8s 1.9 到 1.25 这种跨度），就需要梳理**所有可能出现的 GVR** 以及字段变动，写入到适配器的逻辑里。
    - 一般来说，**高版本对 GA 资源的字段变动不太大**；只是当某些 Beta 特性在大版本间变更时，才需要做额外适配。
4. **缓存与刷新**
    - 在实际生产环境里，集群可能升级，导致启动时发现的可用版本和后续变动不一致。
    - 如果需要长期运行并具备对 K8s 升级的应对能力，可以做一个定时/事件触发的 Discovery 刷新，然后动态更新 `preferredGVR`（需要考虑已经在管理中的资源如何迁移/升级版本的问题）。
    - 如果不需要在运行中动态应对升级，那么只在启动时做一次 Discovery 就够了。
5. **对 CRD 的适配**
    - CRD 的版本管理与内置资源类似，也有 group/version/resource。
    - 如果的系统还需要对接大量自定义资源（CRDs），同样可以按上述逻辑在 Discovery 里找到相应 GVR，并做类似的 Canonical 转换。
    - CRD 可能在不同集群版本下被安装成不同版本（`v1alpha1`、`v1beta1`、`v1`），也可以在应用层做"同一 CRD 的多版本兼容"逻辑。

---

## 5. 跨十多个版本场景下的注意事项
+ **API Deprecated & Removed**:  
Kubernetes 在演进时，会对一些 Beta 版本进行废弃或升级，在某些主版本直接移除。比如 `extensions/v1beta1` 在 1.16+ 就逐步废弃了。
    - 若要兼容 1.9 或更老的集群，需要知道那时的资源版本是啥；但对更高版本（如 1.25）的集群来说，该版本资源已经移除。
    - 这就意味着的逻辑需要知道：`extensions/v1beta1` 在某些新集群里根本不会出现，需要 fallback 到 `apps/v1`。
    - 还可能出现**字段结构的变化**（如 Ingress v1beta1 vs v1 在 spec 里字段差异较大）。
+ **核心资源字段的变动**:  
大多数核心字段在升级时会保持兼容，但一些 Beta 字段可能会发生更名或结构变动。
    - 对于需要深度操作 spec 的场景，需要在"Canonical <-> 目标版本"的转换逻辑里进行分支判断。
    - 如果只是读/写常规元数据和少量必需字段，可能不会有太大差异。
+ **强类型 vs Unstructured**:
    - 如果业务场景极为复杂，需要非常强的 IDE 提示、类型安全和测试，可能更倾向用强类型 struct 并"多套"生成（`client-go` + code-generator）或手动编写对应版本的 struct，这维护成本较高。
    - 如果接受动态方案，则通过 `Unstructured`、`map[string]interface{}` 形式来处理，然后自己做 JSON path 的取值与赋值，这样可以用相同的逻辑去兼容多版本，但也要自己小心字段拼写和结构的变动。

---

## 6. 小结
**基于 Discovery + Dynamic Client** 来设计一个适配多版本 K8s（尤其是跨很多历史版本）的方法，大体可以归纳为以下关键步骤：

1. **Discovery**：在应用启动或需要时，动态查询当前集群所支持的资源版本列表。
2. **映射与选择**：针对需要的资源，按照自定义的"版本优先级"逻辑，匹配到一个当前集群可用的 GVR；若多个版本都可用，就选择一个优先级最高（通常是最新、稳定版）的版本。
3. **Dynamic Client CRUD**：借助 `dynamic.Interface` 以 `Unstructured` 形式进行读写。
4. **字段适配**：在"代码封装层"维护一套 Canonical 结构与多版本字段之间的转换逻辑，把繁杂的多版本差异对上层屏蔽。
5. **缓存 & 刷新**：可选择在应用启动时做一次 Discovery 并缓存结果，必要时定时或主动刷新，以应对集群升级或 CRD 版本更新。
6. **兼容策略**：对于已废弃或已移除的旧版本，做降级或抛错；对于新增的版本，做好扩展或字段兼容处理。

通过这种方案，可以把复杂的多版本 API 差异，集中在一个"版本适配层"里处理，让上层业务逻辑只面向相对稳定的"Canonical 数据结构"或统一的方法接口，从而实现对多个 K8s 版本的透明兼容。

---

## 7. 整体架构

### 7.1 核心组件

- **适配器层 (Adapter Layer)**
  - 提供统一的资源操作接口
  - 处理版本兼容性
  - 管理资源生命周期

- **发现层 (Discovery Layer)**
  - 自动发现集群支持的 API 版本
  - 维护 GVR 映射关系
  - 处理版本优先级

- **转换层 (Conversion Layer)**
  - 处理不同版本间的字段差异
  - 提供字段校验功能
  - 支持 YAML/JSON 编解码

### 7.2 数据流

1. 用户通过适配器接口发起请求
2. 适配器通过发现层确定合适的 GVR
3. 转换层处理版本差异和字段校验
4. 通过 Dynamic Client 与 Kubernetes API 交互

### 7.2 扩展性设计

1. **插件化架构**
   - 支持自定义资源注册
   - 可扩展的校验规则
   - 灵活的转换策略

2. **缓存机制**
   - GVR 信息缓存
   - Schema 缓存
   - 定期刷新机制

## 8. 核心接口

### 8.1 ICompatibleEngine

```go
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
```

### 8.2 CompatibleEngine

```go
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
```

## 9. 性能优化

1. **缓存策略**
   - 使用内存缓存存储 GVR 信息
   - 定期刷新缓存
   - 支持手动刷新

2. **并发处理**
   - 支持并发操作
   - 资源锁机制
   - 批量处理优化