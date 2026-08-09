# Seshat Repository Instructions

本文件只保留所有任务都必须遵守的仓库约束。特定任务的详细步骤放在项目级 Skill 中，执行对应任务时必须读取并遵守相应 Skill。

## Task Skills

- Beta、Release、Hotfix、版本号调整、发布 PR、镜像构建排错或回滚：使用 `$seshat-release`（`.agents/skills/seshat-release/SKILL.md`）。
- 新增 App，或修改某个 App 的 Webhook 协议、消息类型、标准化字段、展示卡片、通知类型或接入说明：使用 `$seshat-app-integration`（`.agents/skills/seshat-app-integration/SKILL.md`）。
- 新增或修改推送渠道、渠道配置、协议发送、响应判断、代理策略或渠道表单：使用 `$seshat-notification-channel`（`.agents/skills/seshat-notification-channel/SKILL.md`）。

## UI Localization

前端同时支持英文和中文。任何新增或修改的用户可见 UI 文本都必须适配两种语言，不得只完成其中一种语言后交付。

- 页面标题、导航、按钮、表单标签与提示、占位符、校验和错误提示、Toast、确认弹窗、空状态、表格列名、筛选项、状态文案、接入说明、图片替代文本以及 `aria-label`、`title` 等无障碍文本，都必须通过统一 i18n 资源读取，禁止在组件或渠道 Adapter 中新增硬编码的单语言文本。
- 每次新增或修改翻译 key，必须同步维护英文和中文资源；两套资源的 key、插值变量和复数语义必须保持一致。App 名称、协议缩写、用户输入和服务端原始内容等无需翻译的数据除外。
- 英文是默认语言；未设置手动偏好时可根据浏览器语言自动切换中文。用户手动选择的语言必须持久化，且优先级高于浏览器自动判断。修改语言逻辑时不得破坏该优先级。
- 日期、时间、数字等区域相关内容必须使用当前界面的 locale 格式化，不得固定为某一种语言或地区格式。
- UI 文本改动必须补充或更新相关测试，并运行翻译资源完整性、语言偏好以及受影响页面或组件的测试，确保英文和中文下都不会显示缺失 key 或错误回退文本。

## Business Logging

需要在管理后台“业务日志”页面查询的手动日志，统一使用 `backend/internal/logging` 提供的分级方法：

```go
logging.Debug("webhook", "开始解析消息", logging.Fields{"request_id": requestID})
logging.Info("webhook", "消息处理完成", logging.Fields{"event_id": eventID})
logging.Warn("webhook", "消息签名无效", logging.Fields{"app_code": appCode})
logging.Error("database", "保存消息失败", logging.Fields{"error": err})
```

- 支持 `DEBUG`、`INFO`、`WARN`、`ERROR` 四个等级，调用时必须选择符合语义的等级。
- `source` 使用稳定的模块名，例如 `server`、`webhook`、`notification`，不要写动态值。
- 可使用 `request_id`、`user_id`、`app_code`、`integration_id`、`event_id` 作为结构化关联字段。
- 禁止记录 Webhook Secret、Token、Password、Cookie、Authorization、Signature、API Key 等敏感值；日志组件按字段名自动脱敏仅作为兜底。
- 标准库 `log.Printf`、`fmt.Printf` 只输出到容器控制台，不会进入业务日志页面；需要后台可查询时必须使用上述分级方法。

## App Webhook Isolation

每个 App 的 Webhook 协议、消息类型和展示逻辑必须独立维护。修改某个 App 时，默认不得改变其他 App 的解析和展示结果。

- 后端每个 App 必须有独立的 `Provider` 实现；App 专有字段、事件别名、签名和兼容逻辑只能放在该 App 的文件中，禁止在共享解析器中堆叠 `app_code` 条件。
- 共享后端代码只能包含稳定的标准化展示协议和无 App 业务语义的纯工具。
- 前端通过渲染器注册表按 `app_code + display_event_type` 选择实现；App 专有展示模型和布局必须放在独立渲染器中。
- 未知 App 或未知消息类型必须回退到默认原始消息渲染器，不得猜测为其他类型。
- 新增或修改 App 必须补充该 App 的 Provider、标准化、未知消息回退和前端渲染路由测试，并运行所有 App 的回归测试。

## Notification Channel Isolation

每个推送渠道的配置、凭据、协议和表单必须通过独立 Adapter 维护。修改一个渠道时，默认不得改变其他渠道的配置清洗、发送内容或成功判定。

- 后端每个渠道必须有独立的 `notificationChannelAdapter` 实现；渠道专有字段、默认值、配置校验、请求构造、签名和响应判断只能放在该渠道文件中。
- 禁止在共享发送器、服务或 Handler 中新增按渠道类型分支；共享代码只能包含 Adapter 注册、队列重试、网络安全、代理、HTTP/SMTP 基础设施和通用消息模型。
- 前端每个渠道必须有独立的 `NotificationChannelAdapter` 模块，拥有自己的默认字段、编辑回填、请求转换和表单组件；页面只能通过注册表选择 Adapter。
- 新增或修改渠道必须添加该 Adapter 的配置隔离、请求/响应或发送行为、凭据保护和前端注册路由测试，并运行全部渠道回归测试。

## Release Invariants

| 分支 | 用途 | 是否触发镜像发布 |
|------|------|------------------|
| `dev` | 日常开发集成 | 否 |
| `beta` | Beta 验证环境 | MR/PR 合并后触发 Beta 发布 |
| `main` | 正式环境 | MR/PR 合并后触发 Release 发布 |

| 发布类型 | 唯一版本源 | 文件格式 | Docker 不可变标签 |
|----------|------------|----------|-------------------|
| Beta | `VERSION_BETA` | `vX.Y.Z` | `beta-vX.Y.Z` |
| Release / Hotfix | `VERSION_RELEASE` | `vX.Y.Z` | `vX.Y.Z` |

- 功能分支先合并到 `dev`；禁止通过直接 push 到 `beta` 或 `main` 发版。
- Beta 和 Release 版本相互独立。一次发版只能修改对应版本文件，不得同时递增两个版本。
- 版本文件只能包含一行语义化版本 `vX.Y.Z`，不得使用 SHA、日期、滚动标签或带渠道前缀的值。
- 已发布的不可变镜像标签不得覆盖或复用；需要重新构建时必须升级对应版本号。
- 每个版本必须新增对应渠道的结构化双语更新记录：Beta 使用 `release-notes/beta/vX.Y.Z.json`，Release/Hotfix 使用 `release-notes/release/vX.Y.Z.json`；英文和中文内容必须同时存在。
- 更新提醒和累计更新内容必须严格按运行渠道筛选，Beta 不得读取 Release 更新，Release 不得读取 Beta 更新。
- 启用 Cloudflare Pages 发布后，每次 Beta 或 Release 发版必须重建双语项目主页、更新历史，以及 `/updates/v1/beta.json`、`/updates/v1/release.json` 固定静态更新源；Pages 发布失败不得被误判为镜像不存在。
- Beta 验证后通过 PR 将 `beta` 合回 `dev`。Release 验证后依次将 `main` 合回 `beta`、将最新 `beta` 合回 `dev`。
- 不得删除 Docker Hub 版本存在性检查，也不得把鉴权失败、网络异常或限流当成镜像不存在。

## Agent Guardrails

- 用户未明确发布类型和目标版本时，不得自行修改版本文件。
- 用户要求 Beta 发版时，只操作 `VERSION_BETA`，发布 PR 目标只能是 `beta`。
- 用户要求 Release 或 Hotfix 发版时，只操作 `VERSION_RELEASE`，发布 PR 目标只能是 `main`。
- 创建发布 PR 前必须报告源分支、目标分支、版本文件和预期镜像标签。
- 除非用户明确要求，不自动创建分支、提交、推送或创建 PR。
- 工作区存在用户改动时必须保留；不得为完成当前任务覆盖、还原或混入无关改动。
