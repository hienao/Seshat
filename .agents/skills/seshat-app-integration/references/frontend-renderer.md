# Frontend Renderer

## 位置

- 入口：`frontend/src/components/webhook/event-card.tsx`
- 注册表：`frontend/src/components/webhook/renderers/registry.ts`
- 类型：`frontend/src/components/webhook/renderers/types.ts`
- App 渲染器：`frontend/src/components/webhook/renderers/<app>-event-card.tsx`
- 默认和原始回退：`default-event-card.tsx`、`raw-event-card.tsx`

## 注册规则

在注册表中按 `app_code` 注册 App 的默认渲染器；某些事件需要不同布局时，再通过 `eventRenderers[display_event_type]` 注册专属渲染器。

解析顺序必须保持：

1. `is_fallback` 或 `display_event_type === '__default__'` 使用 `RawEventCard`。
2. 已注册 App 根据 `display_event_type` 选择专属渲染器，否则使用该 App 的默认渲染器。
3. 未注册 App 使用 `DefaultEventCard`。

禁止在页面、`EventCard`、公共 View 或其他 App 渲染器中添加当前 App 的条件分支。

## 展示模型

- App 渲染器把标准化 Presentation 转换成 `EventCardModel`。
- 公共 `EventCardView` 只负责视觉外壳和通用交互。
- 标题、字段优先级、媒体信息、客户端/设备、简介和特殊布局由 App 渲染器决定。
- 列表与详情通过 `compact`、`detail` 使用同一渲染器，避免形成第二套解析逻辑。
- 公开详情复用标准化渲染器，但隐藏 raw preview、后台关联入口和需要登录的信息。
