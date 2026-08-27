package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

func TestAttributeDriftDetectsStaleActiveMatrix(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, _ := svc.Runs.CreateRun(ctx, RunInput{Name: "age-run", TelescopeID: "tel-1"})
	star, _ := svc.Stars.RegisterStar(ctx, StarInput{Name: "age-star", SNR: 120, PositionErrorArcsec: 0.1})
	nowMS := time.Now().UnixMilli()
	captureMS := nowMS - 5*24*3600*1000
	mat, _ := svc.Matrices.RegisterMatrix(ctx, MatrixInput{Version: "v-age", Rows: 16, Cols: 16, EffectiveFrom: nowMS - 40*24*3600*1000, EffectiveUntil: nowMS + 86400_000})
	_, _ = svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: 1, TimestampMS: captureMS - 1000, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(1)})
	_, _ = svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: 2, TimestampMS: captureMS, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(3)})
	_, _ = svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID)
	cands, _ := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	var frame2 *model.DriftCandidate
	for _, c := range cands {
		if c.FrameID == "wf_"+run.ID+"_2" {
			frame2 = c
		}
	}
	if frame2 == nil {
		t.Fatal("missing frame 2 candidate")
	}
	if frame2.Attribution != model.AttributionCalib {
		t.Fatalf("attribution = %s, want %s", frame2.Attribution, model.AttributionCalib)
	}
}
