---
status: deprecated
doc_type: draft
owner: engineering
last_reviewed: 2026-11-05
source_of_truth:
  - docs/features/current/peer-relay-manager.md
  - docs/api/route-index.md
  - docs/architecture/auth-and-permissions.md
  - docs/architecture/data-and-contracts.md
  - api/openapi/openapi.yaml
related_docs:
  - docs/templates/draft-spec.md
  - docs/architecture/auth-and-permissions.md
  - docs/architecture/data-and-contracts.md
---

# 授权编排白板 PRD 与技术选型（已完成提案归档）

> 本文件记录实现前的产品与技术方案。计划已落地；当前行为以 front matter 中的 canonical 文档和 API 契约为准。

## 背景

授权编排页目前以一台源设备为中心：用户从设备列表选择源设备，在 Relay 候选列表中勾选授权，再预览并保存。这能完成授权修改，但无法一次理解多台设备与多个 Peer Relay 之间的整体关系，也不支持自由调整节点位置。

本提案把该页面升级为全网授权画布，便于有权限的用户查看设备关系、识别授权来源，并在一个草稿中批量调整授权。画布表达 Headscale Policy 中的授权关系，不表达实时网络路径或 Relay 健康状态。

## 当前状态

- **已实现**：全网授权画布、权限范围内的节点与关系、搜索筛选、节点布局与视口本地保存、内存授权草稿及批量审阅。
- **已实现**：新增 `GET /graph/all` 和 `PUT /assignments/batch`，并保留单设备接口兼容。
- 当前权限、关系分类、Policy 更新和持久化行为见 canonical 文档；本归档仅保留实现前的决策背景与验收目标。

## 提案

### 产品目标与用户

- 管理员和获授权的审核员可以查看并编辑其权限范围内的设备授权；普通用户只看得到自己的源设备，但可以查看可用的 Peer Relay 目标。
- 用户可以在一个画布上理解现有授权、快速定位设备，并对多个源设备的授权变更进行一次审阅和提交。
- 保留 Tailor 和 Headscale Policy 作为授权事实来源；本地布局只影响显示，不改变授权数据。

### 页面与交互

1. `/editor` 显示全网授权关系。每台 Headscale 设备只对应一个节点；注册为 Peer Relay 的设备在同一节点上显示 Relay 角色标记。
2. 首次打开时按设备名称将节点放入网格。用户可拖动节点并吸附到网格，使用画布缩放、平移、适配视图和小地图导航。
3. 搜索和筛选支持设备名、用户名、IP、标签、Relay 角色和授权来源。选中节点时显示设备状态、所有者、Relay 元数据以及相关授权；相关边高亮，其他边降低视觉权重。
4. 用户从可编辑源设备节点拖动连接到可授权的 Relay 节点，或通过节点详情面板用键盘选择目标。断开边表示从草稿移除该授权。相同源设备与 Relay 之间最多一条边。
5. 关系面板显示平台授权、外部授权、继承授权、未知关系和异常状态。外部、继承和未知关系不可修改；平台管理的关系和无现存关系按每条关系及源设备的 `editable` 权限决定是否可编辑。不可新增不可选 Relay 的授权，但可按现有规则移除仍可编辑的既有授权。
6. 所有变更先进入草稿。保存前展示新增、移除和受影响源设备的摘要；保存成功后刷新图和 Policy 基线。放弃草稿或离开页面会提示未保存变更。
7. 节点坐标和画布视口保存在当前浏览器的本地存储，按 Tailor 用户名和布局版本隔离；新增节点放入下一个空网格位置。提供重置布局操作。授权草稿不写入本地存储，布局也不上传服务端。

### 授权与安全规则

- 画布只创建源设备到已注册 Peer Relay 的授权边；不提供边权重、优先级或一般 Headscale ACL 编辑。
- 每个源节点单独表达 `canEdit`，服务端逐个验证批量提交中的源设备、设备标签、Relay 状态和用户权限。前端禁用状态只用于提示，不能替代后端授权检查。
- `policyStale`、Policy 读取失败或 `policyWritesEnabled=false` 时，画布保持可读但不能提交。离线状态只作信息展示；缺失设备或不能确认 IP/Policy 关系的节点与边显示为不可确认状态并保持只读。
- 外部、继承、未知授权及 Policy 中其他字段不被批量编辑覆盖。冲突时拒绝整批提交，保留前端草稿，刷新后要求用户检查差异；不得静默重放。
- Tailor 将一批源设备变更合并为一次完整 Policy 更新，并依照现有基线哈希和写后读取校验流程确认结果。Headscale Policy REST 写入没有跨写入者 compare-and-swap；外部写入者仍可能与 Tailor 竞争，部署继续遵循单写入者或共享写锁要求。

### API 与数据流提案

保留现有单设备 API，新增全图读取和批量写入能力。最终字段定义须同步写入 `api/openapi/openapi.yaml` 并生成 Go/React 客户端。

| 接口提案 | 责任 |
| --- | --- |
| `GET /graph/all` | 一次返回当前用户可读的节点、注册 Relay 角色、授权关系及每个源节点的编辑权限，并携带统一的 Policy 哈希、过期和写入状态。服务端读取一次 Policy 并聚合设备、Relay 和平台授权快照。 |
| `PUT /assignments/batch` | 接收基线哈希和所有发生变化的源设备及其目标 Relay 集合；先验证整批，再合并平台管理的精确授权并执行一次 Policy 写入。任一源设备无权限、关系不合法或基线过期时，拒绝整批。 |

现有 `GET /graph?sourceNodeId=...` 和 `PUT /assignments` 保持可用。布局不新增后端表或 API。

### 技术选型

| 方案 | 决策 | 原因 |
| --- | --- | --- |
| `@xyflow/react` | **采用** | 与 React 19/TypeScript 前端匹配，提供节点和边交互、网格吸附、自定义 React 节点与连线校验；可复用其视口控制和画布状态序列化能力。项目采用 MIT 许可证。[React Flow API](https://reactflow.dev/api-reference/react-flow)、[自定义节点](https://reactflow.dev/learn/customization/custom-nodes)、[保存与恢复](https://reactflow.dev/examples/interaction/save-and-restore)、[许可证](https://github.com/xyflow/xyflow/blob/main/LICENSE) |
| tldraw | 不采用 | 面向通用白板的能力范围更大；本需求的核心是有明确来源、目标、权限和只读状态的关系图，节点连线规则需要额外领域适配。 |
| 自建 SVG/Canvas | 不采用 | 需要自行维护拖动、缩放、平移、命中检测、连线选择、键盘交互和视口状态，超出本次页面重构的必要投入。 |

实现时注册稳定的 `nodeTypes` / `edgeTypes`，复用现有 Tailwind 主题、React Query、i18n 和 UI 组件；用 `snapToGrid` 与网格背景实现定位。按需使用可视区域渲染，避免复杂阴影和持续动画；React Flow 文档也建议对大型图优化可见节点和节点样式。[性能指南](https://reactflow.dev/learn/advanced-use/performance)

### 范围边界

- **包含**：全网设备与授权关系浏览、搜索筛选、拖放布局、草稿编辑、批量安全保存、个人本地布局。
- **不包含**：实时流量或路径遥测、Relay 健康探测、自动部署 Relay 服务、多人实时协作、服务端共享画布、Relay 注册字段编辑、一般 Policy 编辑器。
- **设备规模**：以单个 Headscale 实例最多 500 台设备作为 v1 目标；连线密度以代表性授权数据进行性能验证。桌面端提供完整编辑体验，窄屏支持浏览、定位和面板操作。

### 验收标准

- 用户在一个画布内看到其权限范围内的全部设备及适用 Relay 目标；同一物理设备不重复显示，并正确显示 Relay 角色。
- 用户能搜索、筛选、拖动、缩放和定位；刷新后恢复该用户在该浏览器中的节点位置和视口，其他用户的本地布局互不覆盖。
- 用户能创建和移除多台源设备的授权草稿，并在保存前看到清晰变更摘要；非法目标和只读关系不能被 UI 操作修改。
- 一次保存将多源变更合为一次 Policy 更新；成功结果与 Headscale 回读一致。过期基线、任一源权限失败、非法节点或写入失败均不得报告部分成功，草稿保留供检查。
- 外部授权、继承授权、未知授权和 Policy 的非授权字段在提交后保留；Policy 写入关闭或同步状态不可用时页面明确只读原因。
- 使用 500 台设备及代表性授权密度验证首屏可操作、搜索筛选、拖动和关系聚焦；出现性能不足时先调整可视区域渲染与筛选呈现，再评估更换渲染技术。

### 实现阶段测试建议

- 前端组件：来源与目标筛选、网格拖动、Relay 角色标记、键盘添加关系、只读边、草稿差异、保存错误保留草稿、本地布局按用户恢复。
- 后端服务：整批权限验证、重复/非法关系拒绝、基线冲突整批拒绝、一次 Policy 写入、平台授权精确合并、外部规则保留和写后校验。
- 契约：OpenAPI 校验与生成一致性；旧单设备端点回归。
- 性能：500 台设备和代表性边密度下的首屏、过滤和交互基准。

## 待解决问题

- 无待确认的产品决策。批量接口的字段、错误结构和响应载荷在实现前按本提案与现有 OpenAPI 风格补齐，并通过 `make openapi-lint` 与 `make generate-check` 验证。

## 显式声明

- 本文件是已完成的历史提案，不是当前行为说明或待实施计划。
- 当前功能行为见 `docs/features/current/peer-relay-manager.md`；接口契约见 `api/openapi/openapi.yaml` 与 `docs/api/route-index.md`；权限和数据边界见对应 architecture 文档。
- 本文件不得覆盖 `docs/ai/repo-rules.md`、`docs/architecture/`、`docs/api/` 或 `docs/features/current/`。
