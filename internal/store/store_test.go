package store

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/model"
)

// newTestStore 打开临时数据库。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := t.TempDir() + "/test.db"
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestRunLifecycle(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	r := &model.ObservationRun{ID: "run-1", Name: "n", TelescopeID: "tel", Status: model.RunStatusCollecting}
	if err := s.CreateRun(ctx, r); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	got, err := s.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != model.RunStatusCollecting {
		t.Fatalf("status = %s", got.Status)
	}
	// 状态流转 CAS
	if err := s.UpdateRunStatus(ctx, "run-1", model.RunStatusCollecting, model.RunStatusAnalyzing); err != nil {
		t.Fatalf("UpdateRunStatus: %v", err)
	}
	// CAS 冲突：期望旧状态失败
	if err := s.UpdateRunStatus(ctx, "run-1", model.RunStatusCollecting, model.RunStatusSealed); err == nil {
		t.Fatalf("CAS 冲突未被拒绝")
	}
	// 幂等：同状态更新成功
	if err := s.UpdateRunStatus(ctx, "run-1", model.RunStatusAnalyzing, model.RunStatusAnalyzing); err != nil {
		t.Fatalf("同状态更新失败: %v", err)
	}
}

func TestFrameUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	run := &model.ObservationRun{ID: "run-x", Name: "n", TelescopeID: "tel", Status: model.RunStatusCollecting}
	_ = s.CreateRun(ctx, run)
	star := &model.ReferenceStar{ID: "star-x", Name: "s", SNR: 100, PositionErrorArcsec: 0.1, QualityScore: 90, Status: model.StarStatusUsable}
	_ = s.CreateReferenceStar(ctx, star)
	mat := &model.CalibrationMatrix{ID: "mat-x", Version: "v1", Rows: 4, Cols: 4, EffectiveFrom: 0, EffectiveUntil: 1 << 60, Status: model.MatrixStatusActive}
	_ = s.CreateCalibrationMatrix(ctx, mat)

	f := &model.WavefrontFrame{ID: "f1", RunID: "run-x", Seq: 1, TimestampMS: 100, StarID: "star-x", MatrixID: "mat-x", Residuals: []float64{1}, Status: model.FrameStatusRaw, Checksum: "abc"}
	if err := s.CreateFrame(ctx, f); err != nil {
		t.Fatalf("CreateFrame: %v", err)
	}
	// 同 (run, seq) 重复应冲突
	dup := &model.WavefrontFrame{ID: "f2", RunID: "run-x", Seq: 1, TimestampMS: 100, StarID: "star-x", MatrixID: "mat-x", Residuals: []float64{1}, Status: model.FrameStatusRaw, Checksum: "abc"}
	if err := s.CreateFrame(ctx, dup); err == nil {
		t.Fatalf("重复 (run, seq) 未被拒绝")
	}
	// 不同 seq 正常
	ok := &model.WavefrontFrame{ID: "f3", RunID: "run-x", Seq: 2, TimestampMS: 200, StarID: "star-x", MatrixID: "mat-x", Residuals: []float64{2}, Status: model.FrameStatusRaw, Checksum: "def"}
	if err := s.CreateFrame(ctx, ok); err != nil {
		t.Fatalf("CreateFrame seq=2: %v", err)
	}
}

func TestActiveMatrixAt(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	now := time.Now().UnixMilli()
	mat := &model.CalibrationMatrix{ID: "m1", Version: "v1", Rows: 4, Cols: 4, EffectiveFrom: now - 1000, EffectiveUntil: now + 1000, Status: model.MatrixStatusActive}
	if err := s.CreateCalibrationMatrix(ctx, mat); err != nil {
		t.Fatalf("CreateCalibrationMatrix: %v", err)
	}
	got, err := s.ActiveMatrixAt(ctx, now)
	if err != nil {
		t.Fatalf("ActiveMatrixAt: %v", err)
	}
	if got.ID != "m1" {
		t.Fatalf("got %s", got.ID)
	}
	// 区间外无生效矩阵
	if _, err := s.ActiveMatrixAt(ctx, now+5000); err == nil {
		t.Fatalf("区间外不应命中矩阵")
	}
}
