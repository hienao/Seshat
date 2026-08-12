# CI And Rollback

发布工作流位于 `.github/workflows/docker-publish.yml`。

## 触发与构建

- 只处理合并到 `beta` 或 `main` 的 PR `closed` 事件，并要求 `merged == true`。
- `beta` 使用 `VERSION_BETA`，生成 `beta-vX.Y.Z` 和 `beta` 标签。
- `main` 使用 `VERSION_RELEASE`，生成 `vX.Y.Z`、`release` 和 `latest` 标签。
- 构建前查询不可变标签。只有 Docker Hub 明确返回 manifest 不存在时才开始构建。
- `linux/amd64` 与 `linux/arm64` 并行构建为 digest，随后合并成多平台 manifest。
- 同一目标分支的发布任务串行执行，避免覆盖滚动标签。
- GitHub Release 发布成功后，在 `CLOUDFLARE_PAGES_ENABLED=true` 时构建并部署 `website/`，同时更新双语主页和两个渠道的固定 Feed；未启用时跳过，不影响镜像发版。

## 排错顺序

1. 确认 PR 已真正合并，目标分支是 `beta` 或 `main`。
2. 确认对应版本文件格式为单行 `vX.Y.Z`。
3. 检查 `DOCKERHUB_USERNAME`、`DOCKERHUB_TOKEN` 和仓库名，镜像名必须小写。
4. 如果任务因不可变标签存在而跳过，这是预期行为；升级版本后重新发版。
5. 如果镜像查询遇到鉴权、网络或限流错误，修复根因；不得改成“镜像不存在”。
6. 如果某个平台构建失败，只修复代码或工作流并发布新版本；不得覆盖已存在标签。
7. 发布后使用 `docker buildx imagetools inspect <image>:<tag>` 确认同时包含 amd64 和 arm64。
8. 启用 Pages 时确认主页、`/updates/v1/beta.json` 和 `/updates/v1/release.json` 均可访问且通道未混用；Pages 失败只排查 Pages 凭据、项目名和构建，不得覆盖已发布镜像标签。

## 回滚

- 部署和回滚优先使用不可变标签，不只记录 `beta`、`release` 或 `latest`。
- 回滚到上一可用的 `beta-vX.Y.Z` 或 `vX.Y.Z`。
- 不通过重写历史标签实现回滚。
