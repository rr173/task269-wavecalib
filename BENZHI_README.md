# BENZHI 评测说明

基于 Go 实现的自适应光学波前校准漂移诊断后端服务，一款后端服务，完成波前帧幂等接收、残差模式分解、校准矩阵时效匹配与漂移归因、诊断快照发布与冻结。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/wavecalib --addr :8080 --db wavecalib.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/wavecalib --smoke-test
```

`--smoke-test` 会真实创建观测运行与参考星、登记校准矩阵、接收三帧波前残差并验证幂等重投、执行残差分析与漂移归因、发布诊断快照，关闭并重开数据库验证持久化与重启恢复，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/wavecalib --smoke-test
```

## HTTP API（前缀 /api）

- 运行：POST/GET /api/runs、GET /api/runs/{id}、PATCH /api/runs/{id}/status、POST /api/runs/{id}/seal、POST /api/runs/{id}/analyze
- 帧：POST /api/runs/{id}/frames、GET /api/runs/{id}/frames、GET /api/frames/{id}、POST /api/frames/{id}/exclude、POST /api/frames/{id}/restore
- 参考星：POST/GET /api/reference-stars、GET /api/reference-stars/{id}、POST /api/reference-stars/{id}/reevaluate
- 校准矩阵：POST/GET /api/calibration-matrices、GET /api/calibration-matrices/{id}
- 诊断：POST /api/runs/{id}/residual-analysis、GET /api/runs/{id}/residual-modes、POST /api/runs/{id}/drift-candidates、GET /api/runs/{id}/drift-candidates
- 候选：POST /api/candidates/{id}/confirm、POST /api/candidates/{id}/reject
- 快照：POST /api/runs/{id}/snapshots、POST /api/snapshots/{id}/publish、GET /api/runs/{id}/snapshots、GET /api/snapshots/{id}
- 其他：GET /api/health、GET /api/stats

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。七表：runs、reference_stars、calibration_matrices、wavefront_frames、residual_modes、drift_candidates、diagnosis_snapshots。帧 (run_id,seq) 幂等，快照发布后旧发布态置为替代，重启同一数据库可恢复全部状态。
