package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// TestAttributeDriftUsesFrameTimestampNotNow 回归测试：
// 回放几小时前采集的波前帧时，校准矩阵在帧采集时刻仍生效，即便归因当下已过期，
// 也不应一律归因到"校准相关"。归因的时间基准必须按帧采集时刻匹配矩阵版本。
func TestAttributeDriftUsesFrameTimestampNotNow(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	// 采集时刻：几小时前。
	acquired := time.Now().Add(-3 * time.Hour).UnixMilli()
	// 矩阵生效区间覆盖采集时刻，但当前墙钟已超出（已过期）。
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version:        "v-play",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  acquired - 60_000,
		EffectiveUntil: acquired + 60_000,
	})
	if err != nil {
		t.Fatalf("登记矩阵: %v", err)
	}
	if !calibrationIsActiveAt(mat, acquired) {
		t.Fatalf("矩阵应在采集时刻生效: now status=%s", mat.Status)
	}
	if calibrationIsActiveAt(mat, time.Now().UnixMilli()) {
		t.Fatalf("前置条件不满足: 矩阵在归因当下应已过期")
	}

	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "replay", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatalf("创建运行: %v", err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "replay-ref", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatalf("登记参考星: %v", err)
	}
	// 第一帧作为残差基线（无注入偏差），第二帧注入显著 tilt-x 偏差。
	for _, seq := range []int64{1, 3} {
		if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
			RunID:       run.ID,
			Seq:         seq,
			TimestampMS: acquired,
			StarID:      star.ID,
			MatrixID:    mat.ID,
			Residuals:   makeProbeResiduals(seq),
		}); err != nil {
			t.Fatalf("接收帧 %d: %v", seq, err)
		}
	}

	if _, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID); err != nil {
		t.Fatalf("残差分析: %v", err)
	}

	cands, err := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	if err != nil {
		t.Fatalf("漂移归因: %v", err)
	}
	if len(cands) != 2 {
		t.Fatalf("候选数量异常: got %d want 2", len(cands))
	}
	// 第二帧注入了显著 tilt-x，应归因到命令相关，而非误判为校准相关。
	var drifted *model.DriftCandidate
	for _, c := range cands {
		if c.FrameID == "wf_"+run.ID+"_3" {
			drifted = c
		}
	}
	if drifted == nil {
		t.Fatalf("未找到 seq=3 的候选")
	}
	if drifted.Attribution == model.AttributionCalib {
		t.Fatalf("矩阵在帧采集时刻仍生效，不应归因到校准相关，got %q (detail=%s)",
			drifted.Attribution, drifted.Detail)
	}
	if drifted.Attribution != model.AttributionCommand {
		t.Fatalf("注入 tilt-x 显著偏差应归因到命令相关，got %q (detail=%s)",
			drifted.Attribution, drifted.Detail)
	}
}

// calibrationIsActiveAt 复用 calibration 包的生效区间判断，避免循环依赖。
func calibrationIsActiveAt(mat *model.CalibrationMatrix, ts int64) bool {
	return mat.EffectiveFrom <= ts && ts < mat.EffectiveUntil
}
