package service

import (
	"context"
	"encoding/json"
	"testing"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

// TestSnapshotExcludesExcludedFrames 验证快照汇总的帧数应排除已标记"排除"的失真帧。
// 复现：工程师排除一条失真帧后创建诊断快照，快照里 frames 仍为全部帧数而非排除后的帧数。
func TestSnapshotExcludesExcludedFrames(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	run := mustSetupRun(t, svc)
	star := mustSetupStar(t, svc)
	mat := mustSetupMatrix(t, svc)

	// 登记三帧，其中 seq=3 是失真帧（注入显著漂移）。
	for i := int64(1); i <= 3; i++ {
		f, err := svc.Frames.IngestFrame(ctx, newFrameInput(run.ID, star.ID, mat.ID, i, 0))
		if err != nil {
			t.Fatalf("接收帧 seq=%d: %v", i, err)
		}
		if i == 3 {
			// 排除失真帧
			if _, err := svc.Frames.ExcludeFrame(ctx, f.ID); err != nil {
				t.Fatalf("排除帧 %s: %v", f.ID, err)
			}
		}
	}

	snap, err := svc.Snaps.CreateSnapshot(ctx, run.ID, mat.ID)
	if err != nil {
		t.Fatalf("创建快照: %v", err)
	}

	var payload struct {
		Frames int `json:"frames"`
	}
	if err := json.Unmarshal([]byte(snap.Content), &payload); err != nil {
		t.Fatalf("解析快照内容: %v", err)
	}

	if payload.Frames != 2 {
		t.Fatalf("快照帧数应为 2（排除一条失真帧后），got %d", payload.Frames)
	}
}

// newFrameInput 构造一帧波前输入，seq==3 时注入显著漂移（失真帧）。
func newFrameInput(runID, starID, matrixID string, seq, baseMS int64) *acquisition.FrameInput {
	n := 16
	residuals := make([]float64, n*n)
	mid := float64(n-1) / 2.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			x := float64(j) - mid
			y := float64(i) - mid
			v := 0.01*x + 0.02*y + 0.005*(x*x+y*y)
			if seq == 3 {
				v += 1.2 * x // 注入显著 tilt-x 漂移
			}
			residuals[i*n+j] = v
		}
	}
	return &acquisition.FrameInput{
		RunID:       runID,
		Seq:         seq,
		TimestampMS: baseMS + seq*100,
		StarID:      starID,
		MatrixID:    matrixID,
		Residuals:   residuals,
	}
}

func mustSetupRun(t *testing.T, svc *Service) *model.ObservationRun {
	t.Helper()
	r, err := svc.Runs.CreateRun(context.Background(), RunInput{Name: "snap-test", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatalf("创建运行: %v", err)
	}
	return r
}

func mustSetupStar(t *testing.T, svc *Service) *model.ReferenceStar {
	t.Helper()
	st, err := svc.Stars.RegisterStar(context.Background(), StarInput{Name: "ref-snap", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatalf("登记参考星: %v", err)
	}
	return st
}

func mustSetupMatrix(t *testing.T, svc *Service) *model.CalibrationMatrix {
	t.Helper()
	mat, err := svc.Matrices.RegisterMatrix(context.Background(), MatrixInput{
		Version:        "v1.0",
		Rows:           16,
		Cols:           16,
		EffectiveFrom:  1,
		EffectiveUntil: 1 << 60,
	})
	if err != nil {
		t.Fatalf("登记校准矩阵: %v", err)
	}
	return mat
}
