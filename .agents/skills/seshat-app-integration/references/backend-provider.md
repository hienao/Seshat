# Backend Provider

## 位置与接口

- Provider 接口和注册表：`backend/internal/webhook/registry.go`
- App 实现：`backend/internal/webhook/<app>.go`
- App 测试：`backend/internal/webhook/<app>_test.go`
- 接收、fallback 和持久化流程：`backend/internal/service/webhook.go`

每个 App 实现完整的 `Provider`：

- `Code`：返回稳定且唯一的小写 App code。
- `Definition`：声明名称、认证方式、默认事件和已知事件。
- `Verify`：只实现该 App 的认证协议。
- `DetectType`：从真实请求识别原始事件类型。
- `MapType`：把原始事件别名映射为稳定的展示事件类型。
- `ExternalEventID`：提取上游事件 ID；没有可靠 ID 时返回空值。
- `Normalize`：生成稳定的 `Presentation`。

在 `NewRegistry` 中注册新 Provider。不要把 App 专有字段或判断加到 generic Provider。

## AppDefinition

- `DefaultEventType` 使用 `DefaultEventType` 常量，即 `__default__`。
- `EventTypes` 必须包含默认类型 `{Code: DefaultEventType, Name: "其他消息", RenderMode: "raw"}`。
- 已知类型使用稳定的内部 code，不直接依赖上游可能变化的大小写或命名风格。
- 未知类型由服务层回退到默认类型，并保留 `source_event_type` 供排查。

## Presentation

优先使用稳定字段：`title`、`summary`、`severity`、`tags`、`facts`、`links` 和 `data`。媒体类 App 可以在 `data` 中使用共享的媒体、用户、播放和系统展示模型，但字段提取必须留在对应 App Provider 内。

- 缺失字段应省略，不生成误导性占位值。
- 外部图片和链接必须经过 URL 安全校验。
- 不写入 Secret、Token、Authorization、Cookie、签名或完整凭据。
- 原始 body 只用于受保护的后台日志/消息详情和默认 raw fallback，不进入免登录公开详情或推送正文。

## 通知与接入说明

- `AppDefinition.EventTypes` 是接入实例通知设置的类型来源；新增类型后验证设置接口和页面同步展示。
- 默认类型和新类型默认 `notify=false`，不得因为新增类型自动开启推送。
- 若 App 的认证方式或配置方式变化，同步更新前端接入说明及测试。
