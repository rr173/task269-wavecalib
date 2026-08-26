package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
)

func TestAnalyzeRunResidualsSkipsExcludedFrames(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, _ := svc.Runs.CreateRun(ctx, RunInput{Name: "skip-run", TelescopeID: "tel-1"})
	star, _ := svc.Stars.RegisterStar(ctx, StarInput{Name: "skip-star", SNR: 120, PositionErrorArcsec: 0.1})
	nowMS := time.Now().UnixMilli()
	mat, _ := svc.Matrices.RegisterMatrix(ctx, MatrixInput{Version: "v1", Rows: 16, Cols: 16, EffectiveFrom: nowMS - 3600_000, EffectiveUntil: nowMS + 86400_000})
	var midID string
	for seq := int64(1); seq <= 3; seq++ {
		f, _ := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: seq, TimestampMS: nowMS + seq*100, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(seq)})
		if seq == 2 {
			midID = f.ID
		}
	}
	if _, err := svc.Frames.ExcludeFrame(ctx, midID); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.AnalyzedFrames != 2 {
		t.Fatalf("analyzed frames = %d, want 2", res.AnalyzedFrames)
	}
}
