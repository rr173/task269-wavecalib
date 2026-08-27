package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

func TestAttributeDriftUsesFrameTimestamp(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, err := svc.Runs.CreateRun(ctx, RunInput{Name: "time-run", TelescopeID: "tel-1"})
	if err != nil {
		t.Fatal(err)
	}
	star, err := svc.Stars.RegisterStar(ctx, StarInput{Name: "time-star", SNR: 120, PositionErrorArcsec: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	nowMS := time.Now().UnixMilli()
	captureMS := nowMS - 30*24*3600*1000
	mat, err := svc.Matrices.RegisterMatrix(ctx, MatrixInput{
		Version: "v-age", Rows: 16, Cols: 16,
		EffectiveFrom:  nowMS - 31*24*3600*1000 - 3600_000,
		EffectiveUntil: nowMS + 86400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
		RunID: run.ID, Seq: 1, TimestampMS: captureMS - 1000,
		StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(1),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{
		RunID: run.ID, Seq: 2, TimestampMS: captureMS,
		StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(3),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Diagnoses.AnalyzeRunResiduals(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	cands, err := svc.Diagnoses.AttributeDrift(ctx, run.ID)
	if err != nil || len(cands) != 2 {
		t.Fatalf("attribute: %v len=%d", err, len(cands))
	}
	var frame2 *model.DriftCandidate
	for _, c := range cands {
		if c.FrameID == "wf_"+run.ID+"_2" {
			frame2 = c
		}
	}
	if frame2 == nil {
		t.Fatal("missing frame 2 candidate")
	}
	if frame2.Attribution == model.AttributionCalib {
		t.Fatalf("historical frame attributed as calib drift, want command/atmo/unknown")
	}
	if frame2.Attribution != model.AttributionCommand {
		t.Fatalf("attribution = %s, want %s", frame2.Attribution, model.AttributionCommand)
	}
}
