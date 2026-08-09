# App Integration Testing Checklist

## 后端

- Provider 类型已独立注册，未落入 generic Provider。
- 使用真实 payload 覆盖每个新增或修改的上游事件别名。
- `DetectType`、`MapType`、`ExternalEventID` 和 `Normalize` 均有断言。
- 已知事件生成正确的 `display_event_type` 和标准化字段。
- 未知事件回退到 `__default__`，设置 `is_fallback` 并保留原始消息。
- 缺失可选字段、字段大小写差异和嵌套结构不会导致 panic。
- 认证成功、失败和缺失凭据均有测试，测试日志不包含敏感值。
- 通知类型包含默认类型，新类型默认不推送。

至少运行：

```bash
cd backend
go test ./internal/webhook ./internal/service
```

共享协议或注册表变化时运行：

```bash
cd backend
go test ./...
```

## 前端

- 已知 App 被路由到自己的渲染器。
- 需要专属布局的事件按 `display_event_type` 路由正确。
- 未知事件使用 `RawEventCard`，未知 App 使用 `DefaultEventCard`。
- 列表、后台详情和公开详情均展示同一组标准化核心字段。
- 公开详情不展示 raw payload、接口日志入口或后台操作。
- 接入说明包含该 App 的真实配置步骤。
- 通知设置展示全部事件，初始状态均为不推送。

至少运行：

```bash
cd frontend
npm run test
npm run build
```

## 隔离回归

- 修改 Jellyfin 时运行 Emby、GitHub 和默认 Webhook 测试；修改其他 App 时同理。
- 检查共享文件差异，确认其中没有新增 App 专有字段名、事件别名或 `app_code` 分支。
- 对同一份其他 App fixture 比较修改前后的 `display_event_type` 和 Presentation，结果应保持不变。
