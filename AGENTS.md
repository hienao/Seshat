# Seshat Repository Instructions

本文件是本仓库人工操作和自动化 Agent 的发版约束。执行版本修改、创建 MR/PR 或调整发布工作流前，必须先阅读并遵守本文件。

## Branch Model

| 分支 | 用途 | 是否触发镜像发布 |
|------|------|------------------|
| `dev` | 日常开发集成 | 否 |
| `beta` | Beta 验证环境 | MR/PR 合并后触发 Beta 发布 |
| `main` | 正式环境 | MR/PR 合并后触发 Release 发布 |

- 功能分支应先合并到 `dev`。
- 禁止通过直接 push 到 `beta` 或 `main` 发版。发布工作流只监听真正合并的 MR/PR。
- `beta` 和 `main` 应配置为受保护分支并要求通过 MR/PR 合并。
- 当前仓库首次启用此流程前，需要由仓库管理员创建 `beta` 分支。

## Version Sources

Beta 和 Release 版本相互独立，禁止共用或同时递增：

| 发布类型 | 唯一版本源 | 文件示例 | Docker 不可变标签 |
|----------|------------|----------|-------------------|
| Beta | `VERSION_BETA` | `v0.0.1` | `beta-v0.0.1` |
| Release | `VERSION_RELEASE` | `v0.0.1` | `v0.0.1` |

版本文件必须满足以下规则：

- 只包含一行 `vX.Y.Z`，例如 `v0.2.3`。
- 不允许写成 `0.2.3`、`beta-v0.2.3`、`release-v0.2.3` 或 SHA。
- Beta 发版只修改 `VERSION_BETA`，不得顺手修改 `VERSION_RELEASE`。
- Release 发版只修改 `VERSION_RELEASE`，不得顺手修改 `VERSION_BETA`。
- 已发布的不可变标签不得覆盖或复用。需要重新构建时必须升级对应版本号。

版本递增遵循语义化版本：

- Patch：缺陷修复或不改变接口的调整，例如 `v0.0.1` -> `v0.0.2`。
- Minor：向后兼容的新功能，例如 `v0.1.0` -> `v0.2.0`。
- Major：不兼容变更，例如 `v1.4.0` -> `v2.0.0`。

## Beta Release

Beta 发版标准流程：

1. 确认准备发布的功能已经进入 `dev`，并同步最新远端代码。
2. 从最新 `dev` 创建发布分支，推荐命名 `release/beta-vX.Y.Z`。
3. 只将 `VERSION_BETA` 修改为目标版本，例如 `v0.0.2`。
4. 执行下方的发版前校验。
5. 提交版本修改，推荐提交信息：`chore(release): prepare beta-v0.0.2`。
6. 推送发布分支并创建 MR/PR，目标分支必须是 `beta`。
7. MR/PR 合并后，GitHub Actions 检查 Docker Hub 中的 `beta-v0.0.2`。
8. 标签不存在时构建并推送 `beta-v0.0.2`，同时更新滚动标签 `beta`；标签存在时跳过构建。
9. 在 Beta 环境完成验证，记录对应的不可变镜像标签。

不要通过 `dev -> main` 直接完成正式发版，正式版本应基于已经验证的 Beta 代码准备。

## Release

Release 发版标准流程：

1. 确认目标 Beta 镜像已经验证通过，并确认 `beta` 分支是正式版候选代码。
2. 从最新 `beta` 创建发布分支，推荐命名 `release/vX.Y.Z`。
3. 只将 `VERSION_RELEASE` 修改为目标版本，例如 `v0.0.2`。
4. 执行下方的发版前校验。
5. 提交版本修改，推荐提交信息：`chore(release): prepare v0.0.2`。
6. 推送发布分支并创建 MR/PR，目标分支必须是 `main`。
7. MR/PR 合并后，GitHub Actions 检查 Docker Hub 中的 `v0.0.2`。
8. 标签不存在时构建并推送 `v0.0.2`，同时更新滚动标签 `release` 和 `latest`；标签存在时跳过构建。
9. 正式部署和回滚必须优先使用 `vX.Y.Z` 不可变标签，不要只记录 `latest`。

## Hotfix

生产 Hotfix 流程：

1. 从最新 `main` 创建 `hotfix/<description>` 分支。
2. 完成最小修复并只升级 `VERSION_RELEASE`。
3. 校验后创建目标为 `main` 的 MR/PR。
4. 合并并验证正式镜像后，将修复同步回 `beta` 和 `dev`，避免后续版本重新引入问题。
5. 同步分支时不要为了同步而修改版本文件；如果不可变标签已经存在，发布工作流会自动跳过重复构建。

## Pre-release Checks

发起 Beta 或 Release MR/PR 前至少执行：

```bash
cd backend && go test ./...
cd ../frontend && npm run test && npm run build
```

如果修改了 `.github/workflows/`，还必须执行：

```bash
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/docker-publish.yml
```

检查版本文件内容：

```bash
cat VERSION_BETA
cat VERSION_RELEASE
```

MR/PR 描述至少应包含：

- 发布类型：Beta 或 Release。
- 目标版本和预期 Docker 标签。
- 主要改动摘要。
- 测试结果。
- 数据库迁移、配置变化和回滚注意事项。

## CI Behavior

发布工作流位于 `.github/workflows/docker-publish.yml`：

- 仅处理合并到 `beta` 或 `main` 的 MR/PR `closed` 事件，并要求 `merged == true`。
- Docker Hub 仓库默认为 `<DOCKERHUB_USERNAME>/seshat`。
- 构建前查询对应不可变标签是否存在。
- 只有 Docker Hub 明确返回 manifest 不存在时才开始构建。
- 鉴权失败、网络异常和限流必须使任务失败，禁止把这些错误当作镜像不存在。
- Beta 发布维护 `beta` 滚动标签；Release 发布维护 `release` 和 `latest` 滚动标签。
- 相同目标分支的发布任务串行执行，避免并发覆盖滚动标签。

GitHub 仓库必须配置：

- Secret `DOCKERHUB_USERNAME`。
- Secret `DOCKERHUB_TOKEN`，需要目标仓库读写权限。
- 可选 Variable `DOCKERHUB_REPOSITORY`，默认值为 `seshat`。

## Failure And Rollback

- 如果 CI 因标签已存在而跳过，这是预期行为。需要新镜像时升级对应版本并重新创建 MR/PR。
- 如果镜像查询失败，先修复 Docker Hub 凭据或网络问题，不要删除存在性检查。
- 如果构建失败，修复应进入新的提交；不要手工覆盖已发布的不可变标签。
- 回滚时选择上一可用的 `beta-vX.Y.Z` 或 `vX.Y.Z`，不要通过重写历史标签回滚。

## Agent Guardrails

- 用户未明确发布类型和目标版本时，不得自行同时修改两个版本文件。
- 用户要求 Beta 发版时，只操作 `VERSION_BETA`，MR/PR 目标只能是 `beta`。
- 用户要求 Release 发版时，只操作 `VERSION_RELEASE`，MR/PR 目标只能是 `main`。
- 创建 MR/PR 前必须向用户明确报告源分支、目标分支、版本文件和预期镜像标签。
- 不得把 SHA、日期或滚动标签当作正式版本号。
- 不得删除 Docker Hub 版本存在性检查或通过强制覆盖规避版本冲突。
- 除非用户明确要求，不自动创建分支、提交、推送或创建 MR/PR。
