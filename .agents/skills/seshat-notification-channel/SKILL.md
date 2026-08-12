---
name: seshat-notification-channel
description: 新增或修改 Seshat 推送渠道，并保持 Webhook、Telegram、Apprise、邮箱、Server酱、Bark、DingTalk、Feishu、WhatsApp、WxPusher 等渠道的配置、凭据、发送协议、响应判断和前端表单彼此隔离。用户要求增加推送渠道、调整渠道参数、签名、代理、消息格式、API 成功条件或渠道表单时使用。
---

# Seshat Notification Channel

先读取仓库根目录 `AGENTS.md`。把每种推送渠道当成独立 Adapter；修改一个渠道时，不得在共享发送器、服务或页面中增加渠道专有条件。

## 工作流

1. 获取渠道官方 API/协议资料，确认配置字段、敏感凭据、目标地址、请求格式、签名、成功响应、限流和重试语义。不能确认时先查官方资料或要求样例。
2. 修改后端前完整读取 [backend-adapter.md](references/backend-adapter.md)。为渠道创建或修改独立 Adapter 文件，并只在注册表增加一个注册项。
3. 把渠道专有字段白名单、默认值、校验、请求构造、签名和响应判断放在对应 Adapter 中。只复用无渠道业务语义的网络、代理、重试和消息工具。
4. 修改前端前完整读取 [frontend-adapter.md](references/frontend-adapter.md)。让渠道独立提供默认字段、编辑回填、请求转换和表单组件。
5. 对敏感配置执行最小暴露：查询 API 不返回凭据，编辑留空保持原值，日志和错误不包含 Token、Secret、完整目标 URL 或下游响应正文。
6. 完整读取 [testing-checklist.md](references/testing-checklist.md)，添加当前 Adapter 测试并运行所有渠道回归测试。

## 禁止事项

- 禁止在 `notification_sender.go`、`notification.go`、Handler 或页面中新增 `switch channel.Type`、`if type === ...` 等渠道分支。
- 禁止把两个渠道的字段白名单或业务响应规则合并到共享 Adapter。
- 禁止为了复用而让一个渠道 Adapter 根据另一个渠道类型改变行为。
- 禁止把凭据放入公开 `config`、API 响应、业务日志、发送记录错误或通知正文。
- 禁止只添加后端或只添加前端注册，造成支持类型不一致。
