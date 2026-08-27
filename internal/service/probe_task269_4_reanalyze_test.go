package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
)

func TestAnalyzeRunResidualsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "reanalyze-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatal(err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "reanalyze-star", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	nowMS := time.Now().UnixMilli()
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version: "v1", Rows: 16, Cols: 16,
		EffectiveFrom: nowMS - 3600_000, EffectiveUntil: nowMS + 86400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	for seq := int64(1); seq <= 2; seq++ {
		if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
			RunID: run.ID, Seq: seq, TimestampMS: nowMS + seq*100,
			StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(seq),
		}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	if err != nil {
		t.Fatalf("second analyze failed: %v", err)
	}
	if second.AnalyzedFrames != 2 || len(second.Modes) != len(first.Modes) {
		t.Fatalf("re-analyze frames=%d modes=%d", second.AnalyzedFrames, len(second.Modes))
	}
}
