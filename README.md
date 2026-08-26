# task269-wavecalib 自适应光学波前校准漂移诊断服务

## 业务简介

望远镜自适应光学系统依赖波前传感器与变形镜组成的控制闭环保持成像质量。
当传感器校准矩阵过期或参考星质量退化时，残差会沿特定模式持续偏移，
导致补偿失配。`wavecalib` 服务把波前帧、参考星质量、校准矩阵版本与镜面命令关联起来，
定位漂移根源（大气/校准/命令/证据不足），并发布不可变诊断快照供追溯。

## 核心实体与状态机

- **观测运行**：采集中 → 待分析 → 需复核 → 确认 → 封存（封存后输入不可变）
- **波前帧**：原始 → 有效 / 参考不足 / 异常 → 排除
- **漂移候选**：候选 → 已确认 / 已否决（归因：大气相关/校准相关/命令相关/证据不足）
- **诊断快照**：草稿 → 发布 → 替代（同运行只保留一个已发布快照）

## 快速开始

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...
go run ./cmd/wavecalib --smoke-test
go run ./cmd/wavecalib --addr :8080 --db wavecalib.db
```

## API 入口

全部接口以 `/api` 为前缀，详见 `BENZHI_README.md` 的 API 一览表。
示例闭环：

```bash
# 创建运行 -> 登记参考星 -> 登记校准矩阵 -> 接收帧 -> 残差分析 -> 漂移归因 -> 发布快照
curl -X POST localhost:8080/api/runs -d '{"name":"n1","telescope_id":"tel-1"}'
curl -X POST localhost:8080/api/reference-stars -d '{"name":"ref","snr":120,"position_error_arcsec":0.1}'
curl -X POST localhost:8080/api/calibration-matrices \
  -d '{"version":"v1","rows":16,"cols":16,"effective_from_ms":...,"effective_until_ms":...}'
curl -X POST localhost:8080/api/runs/<run_id>/frames -d '{...residuals...}'
curl -X POST localhost:8080/api/runs/<run_id>/residual-analysis
curl -X POST localhost:8080/api/runs/<run_id>/drift-candidates
curl -X POST localhost:8080/api/runs/<run_id>/snapshots -d '{"baseline_matrix_id":"<mat_id>"}'
curl -X POST localhost:8080/api/snapshots/<snap_id>/publish
```

## 持久化

- SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 无关）
- 建表：runs / reference_stars / calibration_matrices / wavefront_frames / residual_modes / drift_candidates / diagnosis_snapshots
- 幂等键：`(run_id, seq)` 唯一 + sha256 校验和；`(frame_id, mode_name)` 唯一；`(run_id, version)` 唯一
- 重启恢复：`--smoke-test` 关闭重开同一数据库验证全部数据恢复

## 组件版本

见 `component-versions.json`：Go 1.26.3，SQLite 3.46.1（经由 modernc.org/sqlite）。
