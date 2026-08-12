---
name: seshat-release
description: 执行 Seshat 的 Beta、Release 和生产 Hotfix 发版，包含版本号选择与校验、发布分支和 PR 流程、发版前测试、Docker Hub 多平台镜像发布排错、发布后分支回合并及回滚。用户要求发 Beta、新版本、正式版、Hotfix、检查发布工作流或处理镜像标签与构建失败时使用。
---

# Seshat Release

先读取仓库根目录 `AGENTS.md`，再执行本 Skill。把发布视为高风险流程，逐项核对分支、版本文件、PR 目标和镜像标签。

## 工作流

1. 确认用户要求的是 Beta、Release 还是 Hotfix，并确认目标版本。未明确时停止修改版本文件并询问用户。
2. 检查当前分支、工作区状态和远端分支。保留所有已有改动，不把无关文件混入发版提交。
3. 完整读取 [release-workflows.md](references/release-workflows.md)，按对应发布类型选择源分支、目标分支、版本文件和回合并路径。
4. 修改版本前运行 `scripts/validate-version.sh <beta|release|hotfix>`；修改后新增对应渠道的 `release-notes/<channel>/vX.Y.Z.json`，附加目标版本再次运行。每条更新必须同时包含英文和中文。
5. 创建发布 PR 前运行 `scripts/preflight.sh <beta|release|hotfix> [目标版本]`。任何检查失败都必须修复或明确报告，不得跳过后宣称通过。
6. 创建 PR 前读取 [pr-template.md](references/pr-template.md)，并向用户报告源分支、目标分支、版本文件和预期不可变镜像标签。
7. 只有用户明确要求时才能创建分支、提交、推送、创建或合并 PR。
8. 排查 GitHub Actions、Docker Hub、manifest、多架构镜像或回滚问题时，读取 [ci-and-rollback.md](references/ci-and-rollback.md)。

## 版本选择

- 缺陷修复或不改变接口的调整使用 Patch。
- 向后兼容的新功能使用 Minor。
- 不兼容变更使用 Major。
- Beta 与 Release 独立递增；不得因为发布其中一个渠道而顺带修改另一个版本源。
- 不可变标签已经存在时必须选择新版本，禁止覆盖标签。
- Beta 更新记录只能写入 `release-notes/beta/`；Release/Hotfix 更新记录只能写入 `release-notes/release/`，不得跨渠道复用。

## 完成条件

- 发版前检查全部通过，并记录实际命令结果。
- PR 的源分支、目标分支、版本文件、目标版本和镜像标签互相一致。
- 目标版本的双语更新记录通过校验，并在镜像发布成功后生成对应 GitHub Release；启用 Cloudflare Pages 时同步重建双语主页、更新历史和 Beta/Release 固定静态更新源。
- 发布后验证不可变标签包含 `linux/amd64` 和 `linux/arm64`。
- 完成对应的回合并 PR，且回合并时不额外修改版本文件。
