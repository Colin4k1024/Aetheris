# Branch Protection Checklist

本清单用于把 CI 与发布门禁转化为 GitHub 仓库级强制策略，避免未通过门禁的变更进入主分支。

## 适用分支

- `main`
- `master`（如果仍在使用）

## 推荐规则

1. Require a pull request before merging
2. Require approvals: 至少 1 位 reviewer
3. Dismiss stale pull request approvals when new commits are pushed
4. Require review from Code Owners（如已配置 CODEOWNERS）
5. Require status checks to pass before merging
6. Require branches to be up to date before merging
7. Require conversation resolution before merging
8. Do not allow bypassing the above settings
9. Include administrators（建议开启）

## Required Status Checks（建议）

以下检查建议设为 Required：

1. `CI / Actionlint`
2. `CI / Build and Test`
3. `CI / Lint`
4. `CI / Postgres Integration`
5. `CI / Coverage Report`
6. `Dependency Security / govulncheck`
7. `PR Serial Gate / Serial Gate`
8. `Release Gates / Release Gate Validation`（建议仅对发布分支策略或受控分支启用）

说明：
- 检查名以 GitHub UI 实际显示为准，不同仓库可能包含前缀或大小写差异。
- 如果你希望只配置一个 Required Check，可把 `CI / Coverage Report` 作为聚合出口（已依赖其他关键 job）。

## Tag 发布保护（可选但推荐）

如果使用 `v*` 标签触发发布：

1. 限制谁可以创建发布标签（仓库权限 + 组织策略）
2. 仅允许受信任维护者创建 `v*` 标签
3. 在发布流程要求 `Release Gates` 成功后再发布资产

## Merge Queue（可选）

如果仓库提交频繁，建议启用 Merge Queue：

1. 启用 Require merge queue
2. 保持 Required Status Checks 与上文一致
3. 防止并发合并导致的基线漂移

## 启用步骤（GitHub UI）

1. 打开仓库 Settings
2. 进入 Branches
3. Add rule 或 Edit rule（针对 `main` / `master`）
4. 勾选并配置上面的推荐规则
5. 在 Required status checks 中添加上述检查项
6. 保存并在一个测试 PR 上验证规则生效

## 验收清单

1. 未通过任一 Required Check 时，Merge 按钮不可点击
2. 新提交后旧审批被自动失效（若开启 stale dismissal）
3. PR 对话未解决时不可合并
4. 管理员账号也受规则约束（若开启 Include administrators）
5. 发布相关改动在主分支上能触发并通过 Release Gates
