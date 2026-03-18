# CI/CD Guide

本指南描述 Aetheris 的 GitHub Actions 门禁策略、触发矩阵、本地复现方法与常见故障排查。

## 目标

- 保证 CI 结果可置信，减少误报与漏报。
- 保证发布门禁在主分支与标签场景严格执行。
- 保证失败路径可诊断、可复现、可回归。

## 工作流与触发矩阵

### CI

文件：`.github/workflows/ci.yml`

触发：
- `push` 到 `main` / `master`
- `pull_request` 到 `main` / `master`

关键门禁：
- Workflow 语法检查：`actionlint`
- 编译：`go build -v ./...`
- 静态检查：`go vet ./...`
- 格式检查：`gofmt -l .`
- 单测：`go test -v -race -count=1 -short ./...`
- 覆盖率：总覆盖率必须 `>= 50%`
- 覆盖率上传：Codecov 上传失败即工作流失败
- Postgres 集成测试：`internal/runtime/jobstore`、`internal/agent/job`

执行策略：
- 启用并发控制：同分支新运行会取消旧运行（`cancel-in-progress: true`）。
- 覆盖率报告 job 依赖所有关键 job，作为结果汇总出口。

### Release Gates

文件：`.github/workflows/release-gates.yml`

触发：
- `push` 到 `main` / `master`
- `push` tag `v*`
- `workflow_dispatch`

强制策略：
- 在工作流环境中设置：
  - `CI=true`
  - `RELEASE_STRICT_P0=true`
- 在严格模式下，以下门禁始终强制开启：
  - `RUN_P0_PERF=1`
  - `RUN_P0_DRILLS=1`
  - `RUN_DB_DRILL=1`
  - `RUN_TENANT_REGRESSION=1`

诊断策略：
- 失败时自动采集 stack 状态和 compose 日志到 `artifacts/release/diagnostics/`。
- 始终上传 `artifacts/release/`，便于回溯。

### Dependency Security

文件：`.github/workflows/dependency-review.yml`

触发：
- `push` / `pull_request` 到 `main` / `master`
- 每周定时扫描
- 手动触发

关键门禁：
- `govulncheck` 二进制模式扫描。

### PR Serial Gate（Merge Queue 替代）

文件：`.github/workflows/pr-serial-gate.yml`

触发：
- PR 打开、同步、标记 ready、加/减标签

规则：
- PR 必须带 `ready-to-merge` 标签才可进入串行门禁。
- 所有带该标签的 PR 按编号升序排队。
- 只有队首 PR 通过门禁，其余 PR 阻塞等待。

使用建议：
- 代码评审完成后再加 `ready-to-merge` 标签。
- 若暂不想排队，移除该标签。
- 合并后，下一个 PR 同步一次分支（或触发 PR 事件）即可重新判定。

## 本地复现

### 快速复现核心 CI

```bash
make ci-local
```

包含：
- `fmt-check`
- `go vet`
- `go test -race -short ./...`
- 本地 2.0 栈启动
- Postgres 集成测试
- 自动停栈

### 分步复现

```bash
make fmt-check
make vet
go test -v -race -count=1 -short ./...
./scripts/local-2.0-stack.sh start
TEST_JOBSTORE_DSN=postgres://postgres:postgres@localhost:5432/aetheris?sslmode=disable go test -v ./internal/runtime/jobstore ./internal/agent/job
./scripts/local-2.0-stack.sh stop
```

### 发布门禁本地演练

```bash
CI=true RELEASE_STRICT_P0=true ./scripts/release-2.0.sh
```

## 常见故障排查

### 1) 覆盖率阈值失败

现象：CI 提示总覆盖率低于 50%。

排查：
- 本地运行：
  - `go test -coverprofile=coverage.out -covermode=atomic ./...`
  - `go tool cover -func=coverage.out | grep total:`
- 重点检查新增或改动包是否缺测试。

### 2) Codecov 上传失败

现象：测试通过但上传步骤失败，工作流整体失败。

排查：
- 检查 Codecov 服务状态。
- 检查仓库侧 token/配置是否有效。
- 必要时重试工作流，但不要跳过该步骤。

### 3) Postgres 集成测试偶发失败

现象：`Apply schema` 或集成测试连接失败。

排查：
- 查看 `Wait for PostgreSQL readiness` 步骤输出。
- 查看服务健康状态与 compose 日志。
- 本地使用 `./scripts/local-2.0-stack.sh status` 与 `logs` 复现。

### 4) Release Gates 超时或失败

现象：发布门禁 60 分钟内步骤失败或超时。

排查：
- 下载 `release-gate-artifacts`，优先看 `diagnostics` 目录。
- 检查 P0 门禁是否被按预期强制执行（脚本会打印生效开关）。

## 维护建议

- 将 `CI` 与 `Release Gates` 设为分支保护必需检查项。
- 对 workflow 变更必须通过 `actionlint`。
- 每次新增关键包时同步补充测试，避免覆盖率债务积累。

## 关联文档

- 分支保护落地清单：`docs/branch-protection-checklist.md`
