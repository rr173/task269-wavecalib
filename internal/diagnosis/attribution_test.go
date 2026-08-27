package diagnosis

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/calibration"
	"task269-wavecalib/internal/model"
	"task269-wavecalib/internal/store"
)

// newAttributionEngine 打开临时数据库并构造归因引擎。
func newAttributionEngine(t *testing.T) *AttributionEngine {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/attr.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return NewAttributionEngine(s, calibration.NewManager(s))
}

// TestAttributeStaleAgeStillActive 归一一个矩阵在生效区间内但已超建议年龄
// （Status=生效，IsStale=true）的帧：应归因到"校准相关"，而非命令/大气/证据不足。
// 这复现了回放一个月前采集帧时系统只按命令/大气归因、从不标校准相关的缺陷。
func TestAttributeStaleAgeStillActive(t *testing.T) {
	e := newAttributionEngine(t)
	ctx := context.Background()

	if err := e.store.CreateRun(ctx, &model.ObservationRun{
		ID: "r1", Name: "r1", TelescopeID: "tel-1", Status: model.RunStatusCollecting,
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	star := &model.ReferenceStar{
		ID: "s1", Name: "s1", SNR: 100, QualityScore: 90, Status: model.StarStatusUsable,
	}
	if err := e.store.CreateReferenceStar(ctx, star); err != nil {
		t.Fatalf("CreateReferenceStar: %v", err)
	}
	now := time.Now().UnixMilli()
	// 生效区间覆盖当下，但生效起点在 35 天前，已超 MaxMatrixAgeDays=30。
	mat := &model.CalibrationMatrix{
		ID:             "m-stale",
		Version:        "v-stale",
		Rows:           4,
		Cols:           4,
		EffectiveFrom:  now - 35*24*3600*1000,
		EffectiveUntil: now + 24*3600*1000,
	}
	if err := calibration.NewManager(e.store).Register(ctx, mat); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if mat.Status != model.MatrixStatusActive {
		t.Fatalf("矩阵状态 = %s, want %s", mat.Status, model.MatrixStatusActive)
	}
	if !calibration.IsStale(mat, now) {
		t.Fatal("矩阵应判定为 stale（超建议年龄）")
	}

	frame := &model.WavefrontFrame{
		ID:          "f-stale",
		RunID:       "r1",
		Seq:         1,
		TimestampMS: now,
		StarID:      "s1",
		MatrixID:    mat.ID,
		Status:      model.FrameStatusValid,
	}
	if err := e.store.CreateFrame(ctx, frame); err != nil {
		t.Fatalf("CreateFrame: %v", err)
	}

	// 给一个足以触发显著偏差、且本应落入"命令相关"的倾斜主导模式，
	// 用以证明 stale 年龄优先于命令/大气归因。
	dev := map[string]float64{
		ModeTiltX: 1.0, // |dev| >= DeviationThreshold 且为倾斜模式
		ModeTiltY: 0.0,
		ModeFocus: 0.0,
		ModeAstig: 0.0,
		ModeComa:  0.0,
	}

	cand, err := e.Attribute(ctx, &Input{
		RunID:          "r1",
		Frame:          frame,
		ModesDeviation: dev,
		Star:           nil,
		Matrix:         mat,
		NowMS:          now,
	})
	if err != nil {
		t.Fatalf("Attribute: %v", err)
	}
	if cand.Attribution != model.AttributionCalib {
		t.Fatalf("归因 = %s, want %s（矩阵超建议年龄应优先标校准相关）",
			cand.Attribution, model.AttributionCalib)
	}
}
