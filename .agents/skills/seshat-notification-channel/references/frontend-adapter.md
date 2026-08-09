# Frontend Channel Adapter

## 位置

- Adapter 类型和共享字段工具：`frontend/src/components/notification-channels/types.ts`
- 注册表：`frontend/src/components/notification-channels/registry.ts`
- 渠道实现：`frontend/src/components/notification-channels/channels/<type>.tsx`
- 页面外壳：`frontend/src/pages/notification-channels-page.tsx`

每个 `NotificationChannelAdapter` 必须提供：

- `type` 和用户可见 `label`。
- `defaultFields`：创建时的本渠道字段，不能包含其他渠道字段。
- `fieldsFromChannel`：从公开 `channel.config` 回填编辑表单；敏感字段保持空值。
- `toPayload`：只生成本渠道的 `config` 和 `credentials`。
- `Form`：本渠道独立表单组件。
- `validate`：需要跨字段校验时提供，返回用户可读错误。

页面只管理名称、类型、启用状态、代理开关和当前 Adapter 的 `fields`。切换类型时使用新 Adapter 的 `defaultFields`，编辑时使用 `fieldsFromChannel`。

允许抽取没有渠道语义的输入布局，或完全相同的机器人字段组件；DingTalk、Feishu 等仍必须拥有独立 Adapter 和独立请求转换。
