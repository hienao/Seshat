# Channel Adapter Testing Checklist

## 后端

- 注册表包含新渠道且类型不重复。
- `Sanitize` 接受本渠道字段，拒绝其他渠道字段和凭据。
- 创建与编辑时默认值、历史凭据合并和完整配置校验正确。
- 请求 endpoint、header、body、签名和敏感字段位置正确。
- HTTP 2xx 下的业务成功与失败响应均有测试。
- 网络错误、限流和服务端错误保持正确的重试语义。
- 错误、日志和 API 响应不暴露凭据或下游响应正文。
- 修改一个渠道后，其他全部 Adapter 测试仍通过。

运行：

```bash
cd backend
go test ./internal/service
go test ./...
```

## 前端

- 注册表包含新渠道且类型不重复。
- 默认字段和编辑回填只包含当前渠道字段。
- `toPayload` 只产生当前渠道配置和凭据。
- 敏感字段编辑时为空，并显示“留空保持原值”。
- 表单只渲染当前渠道组件。
- 页面源码不增加渠道类型条件分支。
- 修改一个渠道后，其他渠道 Payload 测试仍通过。

运行：

```bash
cd frontend
npm run test
npm run build
```
