# Backend Channel Adapter

## 位置

- Adapter 接口、注册表和共享工具：`backend/internal/service/notification_channels.go`
- 渠道实现：`backend/internal/service/notification_channel_<type>.go`
- 队列、重试、HTTP 网络与代理：`backend/internal/service/notification_sender.go`
- 渠道 CRUD：`backend/internal/service/notification.go`

## 接口职责

每个渠道完整实现 `notificationChannelAdapter`：

- `Type`：返回稳定、唯一的小写渠道类型。
- `Sanitize`：只接受本渠道公开配置和敏感凭据字段，设置本渠道默认值。
- `Validate`：校验合并历史凭据后的完整配置。
- `Send`：实现本渠道协议发送。

HTTP 渠道同时实现 `notificationHTTPChannelAdapter`：

- `BuildRequest`：生成 endpoint、header、body、锁定主机和公网目标要求。
- `ValidateResponse`：判断 HTTP 2xx 后的业务响应是否成功。

在 `newNotificationChannelRegistry` 中注册 Adapter。除此之外，不修改共享发送路径来识别新类型。

## 共享边界

允许共享：

- delivery 队列、领取、重试和状态持久化。
- HTTP Client、代理、DNS/IP 校验、重定向限制和超时。
- SMTP 连接等无业务语义的传输辅助函数。
- 标准通知消息、详情链接拼接、JSON 编码和通用错误类型。

必须留在渠道文件：

- 配置与凭据字段名、默认值和组合校验。
- 固定 API Host、Path、签名算法和认证头。
- Payload 结构、标题/正文格式差异。
- 业务成功码和响应结构。

固定第三方官方 API 的渠道应设置 `requirePublicHost=true`；支持用户自建服务的渠道根据产品策略决定，但始终保留云元数据和危险地址防护。
