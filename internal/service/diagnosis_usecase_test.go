package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// TestAnalyzeRunResidualsBaselineFirstFrame 锁定残差分析基线选取逻辑：
// 基线必须取第一帧（seq 最小）的分解系数，而非末帧。
//
// 场景（与线上报障一致）：第 3 帧注入显著的 tilt-x 漂移。
//   - 基线正确（第一帧）时，第 3 帧 tilt-x 偏差显著 -> 归因"命令相关"；
//   - 基线若误取末帧（第三帧），第 3 帧偏差被压成 ~0 -> 归因退化为"证据不足"。
func TestAnalyzeRunResidualsBaselineFirstFrame(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "baseline-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatalf("创建运行: %v", err)
	}
	// 高质量参考星（质量分=100），避免触发"大气相关"分支
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "ref", SNR: 120, PositionErrorArcsec: 0.15})
	if err != nil {
		t.Fatalf("登记参考星: %v", err)
	}
	// 生效中、未过期的校准矩阵，避免触发"校准相关"分支
	nowMS := time.Now().UnixMilli()
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version:        "v1.0",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  nowMS - 3600_000,
		EffectiveUntil: nowMS + 86400_000,
	})
	if err != nil {
		t.Fatalf("登记校准矩阵: %v", err)
	}

	// 三帧：第 3 帧 tilt-x 显著漂移（makeProbeResiduals 在 seq==3 注入 1.2*x）
	frames := make([]*model.WavefrontFrame, 0, 3)
	for seq := int64(1); seq <= 3; seq++ {
		fi := &acquisition.FrameInput{
			RunID:       run.ID,
			Seq:         seq,
			TimestampMS: nowMS + seq*100,
			StarID:      star.ID,
			MatrixID:    mat.ID,
			Residuals:   makeProbeResiduals(seq),
		}
		f, err := svc.Frames.IngestFrame(ctx, fi)
		if err != nil {
			t.Fatalf("接收帧 %d: %v", seq, err)
		}
		frames = append(frames, f)
	}

	if _, err := svc.Runs.AnalyzeRun(ctx, run.ID); err != nil {
		t.Fatalf("推进待分析: %v", err)
	}
	if _, err := svc.Runs.TransitionRun(ctx, run.ID, model.RunStatusReviewing); err != nil {
		t.Fatalf("推进需复核: %v", err)
	}

	if _, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID); err != nil {
		t.Fatalf("残差分析: %v", err)
	}
	cands, err := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	if err != nil {
		t.Fatalf("漂移归因: %v", err)
	}
	if len(cands) != len(frames) {
		t.Fatalf("候选数量 = %d, want %d", len(cands), len(frames))
	}

	// 按 seq 建立帧 ID -> 期望归因的映射，避免依赖候选产出顺序
	want := map[string]string{
		frames[0].ID: model.AttributionUnknown,  // 基线帧自身，偏差为 0
		frames[1].ID: model.AttributionUnknown,  // 与基线同形，偏差为 0
		frames[2].ID: model.AttributionCommand,  // tilt-x 显著漂移
	}
	for _, c := range cands {
		got, ok := want[c.FrameID]
		if !ok {
			t.Errorf("未预期的候选帧 %s (归因=%s)", c.FrameID, c.Attribution)
			continue
		}
		if c.Attribution != got {
			t.Errorf("帧 %s 归因 = %q, want %q (置信度=%.2f, 详情=%s)",
				c.FrameID, c.Attribution, got, c.Confidence, c.Detail)
		}
	}
}
