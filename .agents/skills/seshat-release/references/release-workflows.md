# Release Workflows

## Beta

1. 确认待发布功能已进入最新 `dev`。
2. 从 `dev` 创建 `release/beta-vX.Y.Z`。
3. 只把 `VERSION_BETA` 改为 `vX.Y.Z`，并新增 `release-notes/beta/vX.Y.Z.json` 双语更新记录。
4. 执行发版前检查。
5. 提交信息使用 `chore(release): prepare beta-vX.Y.Z`。
6. 推送发布分支并创建目标为 `beta` 的 PR。
7. PR 合并后，工作流发布不可变标签 `beta-vX.Y.Z`，更新滚动标签 `beta`，并在镜像验证成功后发布同名 GitHub Prerelease 和累计更新 Feed。
8. 验证镜像后创建 `beta -> dev` PR；同步时不修改版本文件。

不要通过 `dev -> main` 直接发布正式版。正式版必须基于已经验证的 `beta`。

## Release

1. 确认目标 Beta 镜像已验证，并以最新 `beta` 作为候选代码。
2. 从 `beta` 创建 `release/vX.Y.Z`。
3. 只把 `VERSION_RELEASE` 改为 `vX.Y.Z`，并新增 `release-notes/release/vX.Y.Z.json` 双语更新记录。
4. 执行发版前检查。
5. 提交信息使用 `chore(release): prepare vX.Y.Z`。
6. 推送发布分支并创建目标为 `main` 的 PR。
7. PR 合并后，工作流发布不可变标签 `vX.Y.Z`，更新 `release` 和 `latest`，并在镜像验证成功后发布同名 GitHub Release 和累计更新 Feed。
8. 验证镜像后依次创建 `main -> beta`、`beta -> dev` PR；同步时不修改版本文件。

## Hotfix

1. 从最新 `main` 创建 `hotfix/<description>`。
2. 完成最小修复，只升级 `VERSION_RELEASE`，并新增 `release-notes/release/vX.Y.Z.json` 双语更新记录。
3. 执行发版前检查并创建目标为 `main` 的 PR。
4. 验证正式镜像后依次创建 `main -> beta`、`beta -> dev` PR。
5. 同步分支时不修改版本文件。

## GitHub 配置

- `beta` 和 `main` 应是受保护分支并要求通过 PR 合并。
- Secret `DOCKERHUB_USERNAME` 必须存在。
- Secret `DOCKERHUB_TOKEN` 必须具有目标仓库读写权限。
- Variable `DOCKERHUB_REPOSITORY` 可选，默认值为 `seshat`。
