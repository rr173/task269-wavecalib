package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

func TestAnalyzeRunBaselineSkipsExcludedFrames(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, _ := svc.Runs.CreateRun(ctx, RunInput{Name: "base-run", TelescopeID: "tel-1"})
	star, _ := svc.Stars.RegisterStar(ctx, StarInput{Name: "base-star", SNR: 120, PositionErrorArcsec: 0.1})
	nowMS := time.Now().UnixMilli()
	mat, _ := svc.Matrices.RegisterMatrix(ctx, MatrixInput{Version: "v1", Rows: 16, Cols: 16, EffectiveFrom: nowMS - 3600_000, EffectiveUntil: nowMS + 86400_000})
	var firstID string
	f1, _ := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: 1, TimestampMS: nowMS + 100, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(3)})
	firstID = f1.ID
	_, _ = svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: 2, TimestampMS: nowMS + 200, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(1)})
	_, _ = svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: 3, TimestampMS: nowMS + 300, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(3)})
	if _, err := svc.Frames.ExcludeFrame(ctx, firstID); err != nil {
		t.Fatal(err)
	}
	_, _ = svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	cands, _ := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	var frame3 *model.DriftCandidate
	for _, c := range cands {
		if c.FrameID == "wf_"+run.ID+"_3" {
			frame3 = c
		}
	}
	if frame3 == nil {
		t.Fatal("missing frame 3 candidate")
	}
	if frame3.Attribution != model.AttributionCommand {
		t.Fatalf("frame3 attribution = %s, want %s", frame3.Attribution, model.AttributionCommand)
	}
}
