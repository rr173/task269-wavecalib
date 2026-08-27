package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// TestAttributeDriftUsesCurrentFrameDeviation 回归测试：漂移归因必须用当前帧自己的残差模式偏差，
// 而不是混入其它帧的偏差。第 3 帧注入显著 tilt-x 漂移，应归因到"命令相关"，
// 不应被错判为"证据不足"（历史 bug：dev map 用了 m.FrameID != f.ID，取了别的帧的偏差）。
func TestAttributeDriftUsesCurrentFrameDeviation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	now := time.Now().UTC()
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "drift-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatalf("创建运行: %v", err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "ref", SNR: 120, PositionErrorArcsec: 0.15})
	if err != nil {
		t.Fatalf("登记参考星: %v", err)
	}
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version:        "v1",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  now.Add(-time.Hour).UnixMilli(),
		EffectiveUntil: now.Add(24 * time.Hour).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("登记校准矩阵: %v", err)
	}
	base := now.UnixMilli()
	for seq := int64(1); seq <= 3; seq++ {
		if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
			RunID:       run.ID,
			Seq:         seq,
			TimestampMS: base + seq*100,
			StarID:      star.ID,
			MatrixID:    mat.ID,
			Residuals:   makeProbeResiduals(seq),
		}); err != nil {
			t.Fatalf("接收帧 %d: %v", seq, err)
		}
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
	if len(cands) != 3 {
		t.Fatalf("候选数量 = %d, want 3", len(cands))
	}
	// 候选按置信度降序返回，找到第 3 帧的候选。
	var third *model.DriftCandidate
	for _, c := range cands {
		if c.FrameID == "wf_"+run.ID+"_3" {
			third = c
		}
	}
	if third == nil {
		t.Fatalf("未找到第 3 帧的漂移候选，cands=%+v", cands)
	}
	if third.Attribution == model.AttributionUnknown {
		t.Fatalf("第 3 帧被错判为 %s（证据不足），dev 用错了别的帧的偏差；详情: %s",
			third.Attribution, third.Detail)
	}
	if third.Attribution != model.AttributionCommand {
		t.Fatalf("第 3 帧 tilt-x 显著漂移应归因为命令相关，got %s；详情: %s",
			third.Attribution, third.Detail)
	}
}
