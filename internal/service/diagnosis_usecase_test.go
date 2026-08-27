package service

import (
	"context"
	"testing"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// frameInputForTest 构造带轻微模式偏移的测试帧，第 3 帧注入显著 tilt-x 漂移。
func frameInputForTest(runID, starID, matrixID string, seq int64) *acquisition.FrameInput {
	return &acquisition.FrameInput{
		RunID:       runID,
		Seq:         seq,
		TimestampMS: seq * 100,
		StarID:      starID,
		MatrixID:    matrixID,
		Residuals:   makeProbeResiduals(seq),
	}
}

// seedRunForResidual 建立一条已推进到"需复核"的运行并接收三帧，返回 runID。
func seedRunForResidual(t *testing.T, svc *Service) string {
	t.Helper()
	ctx := context.Background()
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "dup-residual", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatalf("创建运行: %v", err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "ref-dup", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatalf("登记参考星: %v", err)
	}
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version:        "v-dup",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  1,
		EffectiveUntil: 9_999_999_999,
	})
	if err != nil {
		t.Fatalf("登记校准矩阵: %v", err)
	}
	for i := int64(1); i <= 3; i++ {
		if _, err := svc.Frames.IngestFrame(ctx, frameInputForTest(run.ID, star.ID, mat.ID, i)); err != nil {
			t.Fatalf("接收帧 %d: %v", i, err)
		}
	}
	if _, err := svc.Runs.AnalyzeRun(ctx, run.ID); err != nil {
		t.Fatalf("推进待分析: %v", err)
	}
	if _, err := svc.Runs.TransitionRun(ctx, run.ID, model.RunStatusReviewing); err != nil {
		t.Fatalf("推进需复核: %v", err)
	}
	return run.ID
}

func TestAnalyzeRunResidualsRepeatable(t *testing.T) {
	svc := newTestService(t)
	runID := seedRunForResidual(t, svc)

	first, err := svc.Diagnoses.AnalyzeRunResiduals(context.Background(), runID)
	if err != nil {
		t.Fatalf("首次残差分析失败: %v", err)
	}
	if first.AnalyzedFrames != 3 {
		t.Fatalf("首次分析帧数 = %d, want 3", first.AnalyzedFrames)
	}

	// 再次分析同一运行，应成功完成而非冲突报错。
	second, err := svc.Diagnoses.AnalyzeRunResiduals(context.Background(), runID)
	if err != nil {
		t.Fatalf("重复残差分析失败: %v", err)
	}
	if second.AnalyzedFrames != 3 {
		t.Fatalf("重复分析帧数 = %d, want 3", second.AnalyzedFrames)
	}
	// 重复分析后模式数应保持稳定（每帧 5 种，共 15 条），不出现重复堆积。
	if len(second.Modes) != len(first.Modes) {
		t.Fatalf("重复分析后模式数变化: first=%d second=%d", len(first.Modes), len(second.Modes))
	}
}
