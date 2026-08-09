---
name: seshat-app-integration
description: 新增或修改 Seshat 支持的 Webhook App，并保持各 App 的协议、消息类型、标准化数据、展示卡片、通知选项和接入说明彼此隔离。用户要求接入新 App、调整 Jellyfin/Emby 等 App 的 payload 解析、字段映射、事件类型、默认回退、卡片展示或 App 接入文档时使用。
---

# Seshat App Integration

先读取仓库根目录 `AGENTS.md`。把每个 App 当成独立适配器，修改一个 App 时不得改变其他 App 的解析和展示。

## 工作流

1. 收集该 App 的真实 payload、事件名称、认证方式和官方配置入口。不能确认字段语义时先查官方资料或要求样例，不根据字段名臆测。
2. 修改后端前完整读取 [backend-provider.md](references/backend-provider.md)。新增独立 Provider；已有 App 只改其 Provider 文件和无业务语义的共享协议。
3. 定义已知事件映射，并始终保留 `__default__`。未知事件必须标记为 fallback，保留原始消息展示能力。
4. 设计稳定的 `Presentation` 数据，不把某个 App 的原始字段泄漏成共享展示协议。认证凭据和 Secret 不得进入 Presentation、日志或通知正文。
5. 修改前端前完整读取 [frontend-renderer.md](references/frontend-renderer.md)。为 App 创建或修改独立渲染器，并在注册表中显式注册。
6. 同步消息类型的通知设置；所有类型（包括默认类型）默认不推送。确保每个接入实例仍只绑定一个推送渠道。
7. 新增或更新该 App 的“接入说明”，覆盖目标 URL、认证参数、事件选择和连通性验证。
8. 完整读取 [testing-checklist.md](references/testing-checklist.md) 并完成测试。共享协议有变化时必须运行所有 App 的回归测试。

## 设计边界

- 禁止在共享解析器、页面或公共卡片中新增 `if app_code == ...`。
- App 专有字段别名、签名、兼容逻辑、标题和布局只属于该 App。
- 公共代码只承载注册、稳定协议、纯工具和通用视觉外壳。
- 不复用另一个 App 的 Provider 或渲染器来“近似支持”；可以复用无业务语义的工具和展示组件。
- 列表页、登录后的详情页和免登录公开详情页必须使用同一标准化展示模型；公开页不得暴露原始消息或后台信息。
