# Release PR Template

创建发布 PR 时至少包含：

```markdown
## 发布信息

- 发布类型：Beta / Release / Hotfix
- 目标版本：vX.Y.Z
- 版本文件：VERSION_BETA / VERSION_RELEASE
- 预期镜像标签：beta-vX.Y.Z / vX.Y.Z
- 双语更新记录：release-notes/beta/vX.Y.Z.json / release-notes/release/vX.Y.Z.json

## 主要改动

- ...

## 验证结果

- `cd backend && go test ./...`
- `cd frontend && npm run test`
- `cd frontend && npm run build`
- 工作流有改动时：actionlint
- `./.agents/skills/seshat-release/scripts/validate-release-notes.sh <beta|release> vX.Y.Z`

## 部署与回滚

- 数据库迁移：无 / ...
- 配置变化：无 / ...
- 回滚标签：...
```

创建前向用户明确报告：源分支、目标分支、版本文件、目标版本和预期镜像标签。
