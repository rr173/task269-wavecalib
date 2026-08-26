package service

import (
	"context"
	"testing"
	"time"

	"task269-wavecalib/internal/acquisition"
	"task269-wavecalib/internal/model"
)

func TestAttributeDriftBindsCorrectFrameDeviations(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	run, _ := svc.Runs.CreateRun(ctx, RunInput{Name: "dev-run", TelescopeID: "tel-1"})
	star, _ := svc.Stars.RegisterStar(ctx, StarInput{Name: "dev-star", SNR: 120, PositionErrorArcsec: 0.1})
	nowMS := time.Now().UnixMilli()
	mat, _ := svc.Matrices.RegisterMatrix(ctx, MatrixInput{Version: "v1", Rows: 16, Cols: 16, EffectiveFrom: nowMS - 3600_000, EffectiveUntil: nowMS + 86400_000})
	for seq := int64(1); seq <= 3; seq++ {
		_, _ = svc.Frames.IngestFrame(ctx, &acquisition.FrameInput{RunID: run.ID, Seq: seq, TimestampMS: nowMS + seq*100, StarID: star.ID, MatrixID: mat.ID, Residuals: makeProbeResiduals(seq)})
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
