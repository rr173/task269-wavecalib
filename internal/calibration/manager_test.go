package calibration

import (
	"testing"
	"time"

	"task269-wavecalib/internal/model"
)

func TestIsStaleUsesEffectiveFromAge(t *testing.T) {
	now := time.Now().UnixMilli()
	mat := &model.CalibrationMatrix{
		EffectiveFrom:  now - 31*24*3600*1000,
		EffectiveUntil: now + 3600_000,
		Status:         model.MatrixStatusActive,
	}
	if !IsStale(mat, now) {
		t.Fatal("matrix older than MaxMatrixAgeDays should be stale")
	}
	fresh := &model.CalibrationMatrix{
		EffectiveFrom:  now - 3600_000,
		EffectiveUntil: now + 3600_000,
		Status:         model.MatrixStatusActive,
	}
	if IsStale(fresh, now) {
		t.Fatal("recent matrix should not be stale")
	}
}

func TestAgeDaysNonNegative(t *testing.T) {
	now := time.Now().UnixMilli()
	mat := &model.CalibrationMatrix{EffectiveFrom: now + 3600_000}
	if AgeDays(mat, now) != 0 {
		t.Fatalf("future effective_from age = %d, want 0", AgeDays(mat, now))
	}
}
